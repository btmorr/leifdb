// Unit tests on rpc functionality

package raftserver

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/btmorr/leifdb/internal/testutil"
	"github.com/btmorr/leifdb/internal/util"
	"github.com/golang/protobuf/proto"
)

// checkForeignNodeMock is used to skip membership checks during test, so that a
// Node will respond to RPC calls without creating a full multi-node configuration
func checkForeignNodeMock(addr string, known map[string]*node.ForeignNode) bool {
	return true
}

// setupServer configurs a Database and a Node, mocks cluster membership check,
// and creates a test directory that is automatically cleaned up after each test
func setupServer(t *testing.T) *node.Node {
	addr := "localhost:16990"

	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	store := db.NewDatabase()

	config := node.NewNodeConfig(testDir, addr, make([]string, 0, 0))
	n, _ := node.NewNode(config, store)
	n.CheckForeignNode = checkForeignNodeMock
	return n
}

func TestPersistence(t *testing.T) {
	log.Println("~~~ TestPersistence")

	addr := "localhost:8080"

	testDir, _ := util.CreateTmpDir(".tmp-leifdb")
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	config := node.NewNodeConfig(testDir, addr, make([]string, 0, 0))

	termRecord := &raft.TermRecord{Term: 5, VotedFor: "localhost:8181"}
	node.WriteTerm(config.TermFile, termRecord)

	termData := node.ReadTerm(config.TermFile)
	if termData.Term != termRecord.Term {
		t.Error("Term data file roundtrip incorrect term:", termData.Term)
	}
	if termData.VotedFor != termRecord.VotedFor {
		t.Error("Term data file roundtrip incorrect vote:", termData.VotedFor)
	}

	logCache := &raft.LogStore{
		Entries: []*raft.LogRecord{
			{
				Term:   1,
				Action: raft.LogRecord_SET,
				Key:    "test",
				Value:  "run"},
			{
				Term:   2,
				Action: raft.LogRecord_SET,
				Key:    "other",
				Value:  "questions"},
			{
				Term:   3,
				Action: raft.LogRecord_SET,
				Key:    "stuff",
				Value:  "there"}}}

	err := node.WriteLogs(config.LogFile, logCache)
	if err != nil {
		t.Error("Log write failure:", err)
	}
	_, err2 := os.Stat(config.LogFile)
	if err2 != nil {
		t.Error("LogFile does not exist after write:", err)
	}
	roundtrip := node.ReadLogs(config.LogFile)

	testutil.CompareLogs(t, "Roundtrip", roundtrip, logCache)

	store := db.NewDatabase()
	n, _ := node.NewNode(config, store)

	n.Halt()

	if n.Term != 5 {
		t.Error("Term not loaded correctly. Found term: ", n.Term)
	}

	testutil.CompareLogs(t, "Node load", n.Log, logCache)
}

type appendTestCase struct {
	name            string
	sendTerm        int64
	sendId          string
	sendPrevIdx     int64
	sendPrevTerm    int64
	sendCommit      int64
	sendRecords     []*raft.LogRecord
	expectedSuccess bool
	expectedStore   *raft.LogStore
	expectedDb      map[string]string
}

func TestAppend(t *testing.T) {
	log.Println("~~~ TestAppend")

	addr := "localhost:8080"
	testDir, _ := util.CreateTmpDir(".tmp-leifdb")
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	config := node.NewNodeConfig(testDir, addr, make([]string, 0, 0))

	termRecord := &raft.TermRecord{Term: 5, VotedFor: "localhost:8181"}
	node.WriteTerm(config.TermFile, termRecord)

	starterLog := &raft.LogStore{
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

	out, err := proto.Marshal(starterLog)
	if err != nil {
		log.Fatalln("Failed to encode logs:", err)
	}
	if err := ioutil.WriteFile(config.LogFile, out, 0644); err != nil {
		log.Fatalln("Failed to write log file:", err)
	}

	validLeaderId := "localhost:8181"
	invalidLeaderId := "localhost:12345"
	prevIdx := int64(len(starterLog.Entries) - 1)
	prevTerm := starterLog.Entries[prevIdx].Term
	newRecord := &raft.LogRecord{
		Term:   termRecord.Term,
		Action: raft.LogRecord_SET,
		Key:    "Ginny",
		Value:  "adventuring"}
	updatedLog := &raft.LogStore{Entries: append(starterLog.Entries, newRecord)}

	// todo: construct test case that drops some uncommitted entries
	// todo: construct test case that includes a delete action
	testCases := []appendTestCase{
		{
			name:            "Expired term",
			sendTerm:        termRecord.Term - 1,
			sendId:          validLeaderId,
			sendPrevIdx:     0,
			sendPrevTerm:    0,
			sendCommit:      0,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: false,
			expectedStore:   starterLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "Invalid leader",
			sendTerm:        termRecord.Term,
			sendId:          invalidLeaderId,
			sendPrevIdx:     0,
			sendPrevTerm:    0,
			sendCommit:      2,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: false,
			expectedStore:   starterLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "Empty valid request",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      0,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: true,
			expectedStore:   starterLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "New record",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      0,
			sendRecords:     []*raft.LogRecord{newRecord},
			expectedSuccess: true,
			expectedStore:   updatedLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "Commit some logs",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      1,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: true,
			expectedStore:   updatedLog,
			expectedDb: map[string]string{
				"Harry":    "present",
				"Ron":      "absent",
				"Hermione": "",
				"Ginny":    ""}},
		{
			name:            "Commit all logs",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      int64(len(updatedLog.Entries) - 1),
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: true,
			expectedStore:   updatedLog,
			expectedDb: map[string]string{
				"Harry":    "present",
				"Ron":      "absent",
				"Hermione": "present",
				"Ginny":    "adventuring"}},
	}

	store := db.NewDatabase()
	n, _ := node.NewNode(config, store)
	s := server{Node: n}

	for _, tc := range testCases {
		req := &raft.AppendRequest{
			Term:         tc.sendTerm,
			LeaderId:     tc.sendId,
			PrevLogIndex: tc.sendPrevIdx,
			PrevLogTerm:  tc.sendPrevTerm,
			Entries:      tc.sendRecords,
			LeaderCommit: tc.sendCommit}
		reply, err := s.AppendLogs(context.Background(), req)
		if err != nil {
			t.Errorf("[%s] Unexpected error: %v", tc.name, err)
		}
		// Check for expected success
		if reply.Success != tc.expectedSuccess {
			t.Errorf("[%s] Expected success %t but got %t",
				tc.name,
				tc.expectedSuccess,
				reply.Success)
		}
		// Ensure node logs are as expected
		testutil.CompareLogs(t, tc.name, n.Log, tc.expectedStore)
		// Ensure database state is as expected
		for k := range tc.expectedDb {
			expect := tc.expectedDb[k]
			if store.Get(k) != expect {
				t.Errorf("[%s] Expected %s to be \"%s\"", tc.name, k, expect)
			}
		}
	}
	n.Halt()
}

type voteTestCase struct {
	name            string
	request         *raft.VoteRequest
	expectTerm      int64
	expectVote      bool
	expectNodeState node.Role
}

func TestVote(t *testing.T) {
	log.Println("~~~ TestVote")

	n := setupServer(t)
	testAddr := "localhost:12345"
	s := server{Node: n}
	// Simulating node in leader position, rather than adding a time.Sleep
	n.DoElection()

	testCases := []voteTestCase{
		{
			name: "Vote request expired term",
			request: &raft.VoteRequest{
				Term:         1,
				CandidateId:  testAddr,
				LastLogIndex: -1,
				LastLogTerm:  0},
			expectTerm:      2,
			expectVote:      false,
			expectNodeState: node.Leader},
		{
			name: "Vote request valid",
			request: &raft.VoteRequest{
				Term:         3,
				CandidateId:  testAddr,
				LastLogIndex: -1,
				LastLogTerm:  0},
			expectTerm:      3,
			expectVote:      true,
			expectNodeState: node.Follower}}

	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		reply, err := s.RequestVote(ctx, tc.request)
		fmt.Printf("[%s] Reply: %+v\n", tc.name, reply)
		if err != nil {
			t.Errorf("[%s] Unexpected error: %v", tc.name, err)
		}
		// Check for expected vote
		if reply.VoteGranted != tc.expectVote {
			t.Errorf("[%s] Expected voteGranted %t but got %t",
				tc.name,
				tc.expectVote,
				reply.VoteGranted)
		}
		// Check for expected term
		if reply.Term != tc.expectTerm {
			t.Errorf("[%s] Expected term %d but got %d",
				tc.name,
				tc.expectTerm,
				reply.Term)
		}
		// Ensure node logs are as expected
		if n.State != tc.expectNodeState {
			t.Errorf("[%s] Expected node to be a %v but it is a %v",
				tc.name,
				tc.expectNodeState,
				n.State)
		}
	}
	n.Halt()

	// --- Part 3 ---
	// Todo:
	// After going back to Follower, check that node's election timer fires
	// again, becoming candidate, then failing vote because the other 2
	// known nodes do not respond (go into election cycling).

}
