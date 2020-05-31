// Unit tests on rpc functionality

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	. "github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/golang/protobuf/proto"
)

func TestAppend(t *testing.T) {
	log.Println("~~~ TestAppend")
	t.Skip()
	addr := "localhost:8080"
	testDir, _ := CreateTestDir()
	t.Cleanup(func() {
		RemoveTestDir(testDir)
	})

	config := NewNodeConfig(testDir, addr)

	termRecord := &raft.TermRecord{Term: 5, VotedFor: "localhost:8181"}
	WriteTerm(config.TermFile, termRecord)

	logCache := &raft.LogStore{
		Entries: []*raft.LogRecord{
			{
				Term:   1,
				Action: raft.LogRecord_SET,
				Key:    "Harry",
				Value:  "present"},
			{
				Term:   2,
				Action: raft.LogRecord_SET,
				Key:    "Ron",
				Value:  "absent"},
			{
				Term:   5,
				Action: raft.LogRecord_SET,
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
	body1 := raft.AppendRequest{
		Term:         node.Term - 1,
		LeaderId:     validLeaderId,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      make([]*raft.LogRecord, 0, 0),
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
	body2 := raft.AppendRequest{
		Term:         node.Term,
		LeaderId:     invalidLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      make([]*raft.LogRecord, 0, 0),
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
	body3 := raft.AppendRequest{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      make([]*raft.LogRecord, 0, 0),
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
	record := &raft.LogRecord{
		Term:   node.Term,
		Action: raft.LogRecord_SET,
		Key:    "Ginny",
		Value:  "adventuring"}

	body4 := raft.AppendRequest{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      []*raft.LogRecord{record},
		LeaderCommit: 0}

	b4, _ := json.Marshal(body4)
	br4 := bytes.NewReader(b4)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/append", br4)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Error("Append response status for valid request should be 200")
	}

	expectedLog := &raft.LogStore{Entries: append(logCache.Entries, record)}

	CompareLogs(t, node.Log, expectedLog)

	// --- Part 5 ---
	// construct  append request that commits some logs (the commit index is
	// 1-origin, so a LeaderCommit of 2 will commit the first 2 records)
	body5 := raft.AppendRequest{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      []*raft.LogRecord{},
		LeaderCommit: 2}

	b5, _ := json.Marshal(body5)
	br5 := bytes.NewReader(b5)

	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("POST", "/append", br5)
	router.ServeHTTP(w5, req5)

	if w5.Code != http.StatusOK {
		t.Error("Append response status for valid request should be 200")
	}
	expected := map[string]string{
		"Harry":    "present",
		"Ron":      "absent",
		"Hermione": "present",
		"Ginny":    "adventuring"}
	for k := range expected {
		if store.Get(k) != expected[k] {
			t.Errorf("Expected %s to be %s", k, expected[k])
		}
	}
	unexpected := map[string]string{
		"Hermione": "",
		"Ginny":    ""}
	for k := range unexpected {
		if store.Get(k) != unexpected[k] {
			t.Errorf("Did not expect a status for %s yet\n", k)
		}
	}

	// --- Part 6 ---
	// todo: construct append request that drops some uncommitted entries

	// --- Part 7 ---
	// construct append request that commits all logs
	body7 := raft.AppendRequest{
		Term:         node.Term,
		LeaderId:     validLeaderId,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      []*raft.LogRecord{},
		LeaderCommit: int64(len(logCache.Entries) + 1)}

	b7, _ := json.Marshal(body7)
	br7 := bytes.NewReader(b7)

	w7 := httptest.NewRecorder()
	req7, _ := http.NewRequest("POST", "/append", br7)
	router.ServeHTTP(w7, req7)

	if w7.Code != http.StatusOK {
		t.Error("Append response status for valid request should be 200")
	}
	expected2 := map[string]string{
		"Harry":    "present",
		"Ron":      "absent",
		"Hermione": "present",
		"Ginny":    "adventuring"}
	for k := range expected2 {
		if store.Get(k) != expected2[k] {
			t.Errorf("Expected %s to be %s", k, expected2[k])
		}
	}

	// --- Part 8 ---
	// todo: construct append request including a delete action
}
