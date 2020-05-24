package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/btmorr/go-raft/internal/fileutils"
)

func CreateTestDir() (string, error) {
	tmpDir := os.TempDir()
	dataDir := filepath.Join(tmpDir, ".tmp-go-raft")
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

func TestVoteOldTerm(t *testing.T) {
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

	testTerm := "5 localhost:8181\n"
	fileutils.Write(config.TermFile, testTerm)

	testLog := "1 test\n2 other\n"
	fileutils.Write(config.LogFile, testLog)

	termData, e1 := fileutils.Read(config.TermFile)
	if e1 != nil {
		t.Error(e1)
	}
	if termData != testTerm {
		t.Error("Term data file roundtrip failed")
	}

	logData, e2 := fileutils.Read(config.LogFile)
	if e2 != nil {
		t.Error(e2)
	}
	if logData != testLog {
		t.Error("Term data file roundtrip failed")
	}

	node, _ := NewNode(config)

	if node.Term != 5 {
		t.Error("Term not loaded correctly. Found term: ", node.Term)
	}

	if len(node.log) != 2 {
		t.Error("Incorrect number of logs loaded. Number found: ", len(node.log))
	}
}
