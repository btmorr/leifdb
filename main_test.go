package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	. "github.com/btmorr/leifdb/internal/node"
	pb "github.com/btmorr/leifdb/internal/raft"
	"github.com/gin-gonic/gin"
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

func setupServer(t *testing.T) (*gin.Engine, *Node) {
	addr := "localhost:8080"

	testDir, err := CreateTestDir()
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		RemoveTestDir(testDir)
	})

	store := db.NewDatabase()

	config := NewNodeConfig(testDir, addr)
	node, _ := NewNode(config, store)
	router := buildRouter(node)
	return router, node
}

func TestHealthRoute(t *testing.T) {
	log.Println("~~~ TestHealthRoute")
	router, _ := setupServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status")
	}
}

func TestReadAfterWrite(t *testing.T) {
	log.Println("~~~ TestReadAfterWrite")
	router, _ := setupServer(t)

	v := "testy"
	body1 := WriteBody{
		Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/db/stuff", br1)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status in POST:", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/db/stuff", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("Non-200 health status in GET:", w2.Code)
	}

	raw, _ := ioutil.ReadAll(w2.Body)
	if string(raw) != v {
		t.Error("Incorrect response:", string(raw))
	}
}

func TestDelete(t *testing.T) {
	log.Println("~~~ TestDelete")
	router, _ := setupServer(t)

	v := "testy"
	body1 := WriteBody{
		Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	uri := "/db/stuff"

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", uri, br1)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status in POST:", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", uri, nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("Non-200 health status in GET:", w2.Code)
	}

	raw, _ := ioutil.ReadAll(w2.Body)
	if string(raw) != v {
		t.Error("Incorrect response:", string(raw))
	}

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("DELETE", uri, nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Error("Non-200 health status in DELETE:", w3.Code)
	}

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", uri, nil)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Error("Non-200 health status in GET (for key not found):", w4.Code)
	}

	raw2, _ := ioutil.ReadAll(w2.Body)
	if string(raw2) != "" {
		t.Error("Incorrect response:", string(raw))
	}
}

func TestVote(t *testing.T) {
	log.Println("~~~ TestVote")
	router, node := setupServer(t)

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
		t.Error("Vote should be granted for a new term. Got response:", vote2)
	}

	// --- Part 3 ---
	// Todo:
	// After going back to Follower, check that node's election timer fires
	// again, becoming candidate, then failing vote because the other 2
	// known nodes do not respond (go into election cycling).
	//
	// Mock out Node.SetState to record transitions.

}

func CompareLogs(t *testing.T, got *pb.LogStore, expected *pb.LogStore) {
	// Note: reflect.DeepEqual failed to return true for `LogRecord`s with
	// identical contents, so have to do this instead... (DeepEqual probably
	// can't reliably traverse objects with arrays of pointers to objects)

	length1 := len(got.Entries)
	length2 := len(expected.Entries)
	if length1 != length2 {
		t.Error("Expected", length1, "log entries roundtrip but got", length2)
	} else {
		for idx, entry := range got.Entries {
			if entry.Term != expected.Entries[idx].Term {
				t.Error(
					"Expected term", expected.Entries[idx].Term,
					"got", entry.Term)
			}
			if entry.Key != expected.Entries[idx].Key {
				t.Error(
					"Expected Key", expected.Entries[idx].Key,
					"got", entry.Key)
			}
			if entry.Value != expected.Entries[idx].Value {
				t.Error(
					"Expected value", expected.Entries[idx].Value,
					"got", entry.Value)
			}
		}
	}

}

func TestPersistence(t *testing.T) {
	log.Println("~~~ TestPersistence")
	addr := "localhost:8080"

	testDir, _ := CreateTestDir()
	t.Cleanup(func() {
		RemoveTestDir(testDir)
	})

	config := NewNodeConfig(testDir, addr)

	termRecord := &pb.TermRecord{Term: 5, VotedFor: "localhost:8181"}
	WriteTerm(config.TermFile, termRecord)

	termData := ReadTerm(config.TermFile)
	if termData.Term != termRecord.Term {
		t.Error("Term data file roundtrip incorrect term:", termData.Term)
	}
	if termData.VotedFor != termRecord.VotedFor {
		t.Error("Term data file roundtrip incorrect vote:", termData.VotedFor)
	}

	logCache := &pb.LogStore{
		Entries: []*pb.LogRecord{
			{
				Term:   1,
				Action: pb.LogRecord_SET,
				Key:    "test",
				Value:  "run"},
			{
				Term:   2,
				Action: pb.LogRecord_SET,
				Key:    "other",
				Value:  "questions"},
			{
				Term:   3,
				Action: pb.LogRecord_SET,
				Key:    "stuff",
				Value:  "there"}}}

	err := WriteLogs(config.LogFile, logCache)
	if err != nil {
		t.Error("Log write failure:", err)
	}
	_, err2 := os.Stat(config.LogFile)
	if err2 != nil {
		t.Error("LogFile does not exist after write:", err)
	}
	roundtrip := ReadLogs(config.LogFile)

	CompareLogs(t, roundtrip, logCache)

	store := db.NewDatabase()
	node, _ := NewNode(config, store)

	if node.Term != 5 {
		t.Error("Term not loaded correctly. Found term: ", node.Term)
	}

	CompareLogs(t, node.Log, logCache)
}

func TestAppend(t *testing.T) {
	log.Println("~~~ TestAppend")
	addr := "localhost:8080"
	testDir, _ := CreateTestDir()
	t.Cleanup(func() {
		RemoveTestDir(testDir)
	})

	config := NewNodeConfig(testDir, addr)

	termRecord := &pb.TermRecord{Term: 5, VotedFor: "localhost:8181"}
	WriteTerm(config.TermFile, termRecord)

	logCache := &pb.LogStore{
		Entries: []*pb.LogRecord{
			{
				Term:   1,
				Action: pb.LogRecord_SET,
				Key:    "Harry",
				Value:  "present"},
			{
				Term:   2,
				Action: pb.LogRecord_SET,
				Key:    "Ron",
				Value:  "absent"},
			{
				Term:   5,
				Action: pb.LogRecord_SET,
				Key:    "Hermione",
				Value:  "present"}}}

	out, err := proto.Marshal(logCache)
	if err != nil {
		log.Fatalln("Failed to encode logs:", err)
	}
	if err := ioutil.WriteFile(config.LogFile, out, 0644); err != nil {
		log.Fatalln("Failed to write log file:", err)
	}

	store := db.NewDatabase()
	node, _ := NewNode(config, store)
	router := buildRouter(node)

	validLeaderId := "localhost:8181"
	invalidLeaderId := "localhost:12345"
	prevIdx := int64(len(logCache.Entries) - 1)
	prevTerm := logCache.Entries[prevIdx].Term

	// --- Part 1 ---
	// Construct an empty append request for an expired term
	body1 := pb.AppendRequest{
		Term:         node.Term - 1,
		LeaderId:     validLeaderId,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      make([]*pb.LogRecord, 0, 0),
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
	body2 := pb.AppendRequest{
		Term:         node.Term,
		LeaderId:     invalidLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      make([]*pb.LogRecord, 0, 0),
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
	body3 := pb.AppendRequest{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      make([]*pb.LogRecord, 0, 0),
		LeaderCommit: 0}

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
	record := &pb.LogRecord{
		Term:   node.Term,
		Action: pb.LogRecord_SET,
		Key:    "Ginny",
		Value:  "adventuring"}

	body4 := pb.AppendRequest{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      []*pb.LogRecord{record},
		LeaderCommit: 0}

	b4, _ := json.Marshal(body4)
	br4 := bytes.NewReader(b4)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/append", br4)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Error("Append response status for valid request should be 200")
	}

	expectedLog := &pb.LogStore{Entries: append(logCache.Entries, record)}

	CompareLogs(t, node.Log, expectedLog)

	// --- Part 5 ---
	// todo: construct valid append request that drops some uncommitted entries
}
