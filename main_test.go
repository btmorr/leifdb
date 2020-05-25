package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/btmorr/leifdb/internal/fileutils"
	. "github.com/btmorr/leifdb/internal/types"
	pb "github.com/btmorr/leifdb/internal/persistence"
	"github.com/golang/protobuf/proto"
)

func CreateTestDir() (string, error) {
	tmpDir := os.TempDir()
	dataDir := filepath.Join(tmpDir, ".tmp-leifdb")
	err := EnsureDirectory(dataDir)
	return dataDir, err
}

func RemoveTestDir(path string) error {
	return os.RemoveAll(path)
}

func TestHealthRoute(t *testing.T) {
	addr := "localhost:8080"

	testDir, _ := CreateTestDir()
	defer RemoveTestDir(testDir)

	config := NewNodeConfig(testDir, addr)
	node, _ := NewNode(config)
	router := buildRouter(node)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status")
	}
}

func TestVote(t *testing.T) {
	addr := "localhost:8080"

	// Node should come up and become the leader of a single-node cluster
	testDir, _ := CreateTestDir()
	defer RemoveTestDir(testDir)

	config := NewNodeConfig(testDir, addr)
	node, _ := NewNode(config)
	router := buildRouter(node)

	// --- Part 1 ---
	// Construct a vote for the same term as the leader (which guarantees
	// the term will be expired) from a hypothetical 2nd node
	body1 := VoteBody{
		Term:         node.Term,
		CandidateId:  "localhost:12345",
		LastLogIndex: 0,
		LastLogTerm:  0}

	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/vote", br1)
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusConflict {
		// When successful election has already happened for term N,
		// vote requests with term N should return status 409 CONFLICT
		t.Error("Vote response was not 409")
	}

	raw1, _ := ioutil.ReadAll(w1.Body)

	var vote1 VoteResponse
	json.Unmarshal(raw1, &vote1)

	if vote1.VoteGranted {
		t.Error("Vote should not be granted for expired term")
	}
	// check node state, ensure that node remains leader and term does not increment

	// --- Part 2 ---
	// Construct a vote for a later term, from a hypothetical 3rd node
	body2 := VoteBody{
		Term:         node.Term + 1,
		CandidateId:  "localhost:12346",
		LastLogIndex: 0,
		LastLogTerm:  0}

	b2, _ := json.Marshal(body2)
	br2 := bytes.NewReader(b2)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/vote", br2)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("Vote response was not 200")
	}

	raw2, _ := ioutil.ReadAll(w2.Body)

	var vote2 VoteResponse
	err := json.Unmarshal(raw2, &vote2)
	if err != nil {
		t.Error("Unmarshalling error")
	}

	if !vote2.VoteGranted {
		fmt.Println("Response: ", vote2)
		t.Error("Vote should be granted for a new term")
	}

	// --- Part 3 ---
	// Todo:
	// After going back to Follower, check that node's election timer fires
	// again, becoming candidate, then failing vote because the other 2
	// known nodes do not respond (go into election cycling).
	//
	// Mock out Node.SetState to record transitions.

}

func TestPersistence(t *testing.T) {
	addr := "localhost:8080"

	testDir, _ := CreateTestDir()
	defer RemoveTestDir(testDir)

	config := NewNodeConfig(testDir, addr)

	// Termfile persistence
	testTerm := "5 localhost:8181\n"
	fileutils.Write(config.TermFile, testTerm)

	termData, e1 := fileutils.Read(config.TermFile)
	if e1 != nil {
		t.Error(e1)
	}
	if termData != testTerm {
		t.Error("Term data file roundtrip failed")
	}
}

func TestAppend(t *testing.T) {
	addr := "localhost:8080"
	testDir, _ := CreateTestDir()
	defer RemoveTestDir(testDir)
	config := NewNodeConfig(testDir, addr)

	testTerm := "5 localhost:8181\n"
	fileutils.Write(config.TermFile, testTerm)

	logs := pb.Log{
		Records: []pb.LogRecord{
			pb.LogRecord{
				Term: 1, 
				Action: pb.LogRecord_SET, 
				Key: "Harry", 
				Value: "present"},
			pb.LogRecord{
				Term: 2, 
				Action: pb.LogRecord_SET, 
				Key: "Ron", 
				Value: "absent"},
			pb.LogRecord{
				Term: 5, 
				Action: pb.LogRecord_SET, 
				Key: "Hermione", 
				Value: "present"}}}

	out, err := proto.Marshal(&logs)
	if err != nil {
		log.Fatalln("Failed to encode logs:", err)
	}
	if err := ioutil.WriteFile(config.LogFile, out, 0644); err != nil {
		log.Fatalln("Failed to write log file:", err)
	}

	node, _ := NewNode(config)
	router := buildRouter(node)

	validLeaderId := "localhost:8181"
	invalidLeaderId := "localhost:12345"
	// --- Part 1 ---
	// Construct an empty append request for an expired term
	body1 := AppendBody{
		Term:         node.Term - 1,
		LeaderId:     validLeaderId,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      make([]pb.LogRecord, 0, 0),
		LeaderCommit: 0}

	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/append", br1)
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusConflict {
		t.Error("Append response status for expired term should be 409")
	}

	// --- Part 2 ---
	// Construct an empty append request from an invalid leader
	body2 := AppendBody{
		Term:         node.Term,
		LeaderId:     invalidLeaderId,
		PrevLogIndex: 2,
		PrevLogTerm:  5,
		Entries:      make([]pb.LogRecord, 0, 0),
		LeaderCommit: 2}

	b2, _ := json.Marshal(body2)
	br2 := bytes.NewReader(b2)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/append", br2)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Error("Append response status for invalid leader should be 409")
	}

	// --- Part 3 ---
	// Construct a valid append request
	body3 := AppendBody{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: 2,
		PrevLogTerm:  5,
		Entries:      make([]pb.LogRecord, 0, 0),
		LeaderCommit: 2}

	b3, _ := json.Marshal(body3)
	br3 := bytes.NewReader(b3)

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/append", br3)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Error("Append response status for valid request should be 200")
	}

	// --- Part 4 ---
	// Construct a valid append request with entries
	record := pb.LogRecord{
		Term: node.Term, 
		Action: pb.LogRecord_SET, 
		Key: "Ginny", 
		Value: "adventuring"}

	body4 := AppendBody{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      []pb.LogRecord{record},
		LeaderCommit: 0}

	b4, _ := json.Marshal(body4)
	br4 := bytes.NewReader(b4)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/append", br4)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Error("Append response status for valid request should be 200")
	}

	expectedLog := pb.Log{Records: append(logs.Records, record)}
	for idx, l := range node.log.Records {
		if l != expectedLog.Records[idx] {
			t.Error("Log failed to update on valid append")
		}
	}
}
