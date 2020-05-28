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

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/fileutils"
	. "github.com/btmorr/leifdb/internal/types"
	"github.com/gin-gonic/gin"
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

func setupServer() (*gin.Engine, *Node) {
	addr := "localhost:8080"

	testDir, _ := CreateTestDir()
	defer RemoveTestDir(testDir)

	store := db.NewDatabase()

	config := NewNodeConfig(testDir, addr)
	node, _ := NewNode(config, store)
	router := buildRouter(node)
	return router, node
}

func TestHealthRoute(t *testing.T) {
	router, _ := setupServer()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status")
	}
}

func TestReadAfterWrite(t *testing.T) {
	router, _ := setupServer()

	v := "testy"
	body1 := WriteBody{
		Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/db/stuff", br1)
	fmt.Println("------> POST")
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
	router, _ := setupServer()

	v := "testy"
	body1 := WriteBody{
		Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	uri := "/db/stuff"

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", uri, br1)
	fmt.Println("------> POST")
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
	router, node := setupServer()

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

	// Logfile persistence
	logs := []LogRecord{
		{Term: 1, Record: "set test run"},
		{Term: 2, Record: "set other questions"},
		{Term: 3, Record: "set stuff there"}}

	testLog := ""
	for _, l := range logs {
		logString := fmt.Sprintf("%d %s", l.Term, l.Record)
		testLog = testLog + logString + "\n"
	}
	fileutils.Write(config.LogFile, testLog)

	logData, e2 := fileutils.Read(config.LogFile)

	if e2 != nil {
		t.Error(e2)
	}
	if logData != testLog {
		t.Error("Log data file roundtrip failed")
	}

	store := db.NewDatabase()
	node, _ := NewNode(config, store)

	if node.Term != 5 {
		t.Error("Term not loaded correctly. Found term: ", node.Term)
	}

	for idx, l := range node.log {
		if l != logs[idx] {
			t.Error("Log mismatch:", l, logs[idx])
		}
	}
	if len(node.log) != 3 {
		t.Error("Incorrect number of logs loaded. Number found: ", len(node.log))
	}
}

func TestAppend(t *testing.T) {
	addr := "localhost:8080"
	testDir, _ := CreateTestDir()
	defer RemoveTestDir(testDir)
	config := NewNodeConfig(testDir, addr)

	testTerm := "5 localhost:8181\n"
	fileutils.Write(config.TermFile, testTerm)

	logs := []LogRecord{
		{Term: 1, Record: "set Harry present"},
		{Term: 2, Record: "set Ron absent"},
		{Term: 5, Record: "set Hermione present"}}

	testLog := ""
	for _, l := range logs {
		logString := fmt.Sprintf("%d %s", l.Term, l.Record)
		testLog = testLog + logString + "\n"
	}
	fileutils.Write(config.LogFile, testLog)

	store := db.NewDatabase()
	node, _ := NewNode(config, store)
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
		Entries:      make([]LogRecord, 0, 0),
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
		Entries:      make([]LogRecord, 0, 0),
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
		Entries:      make([]LogRecord, 0, 0),
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
	record := LogRecord{Term: node.Term, Record: "set Ginny adventuring"}
	body4 := AppendBody{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      []LogRecord{record},
		LeaderCommit: 0}

	b4, _ := json.Marshal(body4)
	br4 := bytes.NewReader(b4)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/append", br4)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Error("Append response status for valid request should be 200")
	}

	expectedLog := append(logs, record)
	for idx, l := range node.log {
		if l != expectedLog[idx] {
			t.Error("Log failed to update on valid append")
		}
	}
}
