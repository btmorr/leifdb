// +build unit

package raftserver

import (
	"context"
	"io/ioutil"
	"log"
	"testing"
	"time"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/mgmt"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/btmorr/leifdb/internal/testutil"
	"github.com/btmorr/leifdb/internal/util"
	"github.com/golang/protobuf/proto"
)

// checkMock is used to skip membership checks during test, so that a Node will
// respond to RPC calls without creating a full multi-node configuration
func checkMock(addr string, known map[string]*node.ForeignNode) bool {
	return true
}

// setupServer configurs a Database and a Node, mocks cluster membership check,
// and creates a test directory that is cleaned up after each test
func setupServer(t *testing.T) *node.Node {
	addr := "localhost:16990"
	clientAddr := "localhost:8080"

	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	store := db.NewDatabase()

	config := node.NewNodeConfig(testDir, addr, clientAddr, make([]string, 0, 0))
	n, _ := node.NewNode(config, store)
	n.CheckForeignNode = checkMock
	return n
}

type appendTestCase struct {
	name            string
	sendTerm        int64
	sendId          *raft.Node
	sendPrevIdx     int64
	sendPrevTerm    int64
	sendCommit      int64
	sendRecords     []*raft.LogRecord
	expectedSuccess bool
	expectedStore   *raft.LogStore
	expectedDb      map[string]string
}

func TestAppend(t *testing.T) {
	addr := "localhost:16990"
	clientAddr := "localhost:8080"

	testDir, _ := util.CreateTmpDir(".tmp-leifdb")
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	config := node.NewNodeConfig(testDir, addr, clientAddr, make([]string, 0, 0))

	validLeader := &raft.Node{
		Id:         "localhost:16991",
		ClientAddr: "localhost:8081",
	}

	invalidLeader := &raft.Node{
		Id:         "localhost:12345",
		ClientAddr: "localhost:9999",
	}

	termRecord := &raft.TermRecord{Term: 5, VotedFor: validLeader}
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
			sendId:          validLeader,
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
			sendId:          invalidLeader,
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
			sendId:          validLeader,
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
			sendId:          validLeader,
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
			sendId:          validLeader,
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
			sendId:          validLeader,
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
			Leader:       tc.sendId,
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
}

type voteTestCase struct {
	name            string
	request         *raft.VoteRequest
	expectTerm      int64
	expectVote      bool
	expectNodeState mgmt.Role
}

func TestVote(t *testing.T) {
	n := setupServer(t)

	testRaftNode := &raft.Node{
		Id:         "localhost:12345",
		ClientAddr: "localhost:9999",
	}

	s := server{Node: n}
	// Simulating node in leader position, rather than adding a time.Sleep
	n.State = mgmt.Leader
	n.DoElection()
	// mock behavior of StateManager
	go func() {
		for {
			select {
			case <-n.Reset:
				n.State = mgmt.Follower
			default:
			}
		}
	}()

	testCases := []voteTestCase{
		{
			name: "Vote request expired term",
			request: &raft.VoteRequest{
				Term:         1,
				Candidate:    testRaftNode,
				LastLogIndex: -1,
				LastLogTerm:  0},
			expectTerm:      2,
			expectVote:      false,
			expectNodeState: mgmt.Leader},
		{
			name: "Vote request valid",
			request: &raft.VoteRequest{
				Term:         3,
				Candidate:    testRaftNode,
				LastLogIndex: -1,
				LastLogTerm:  0},
			expectTerm:      3,
			expectVote:      true,
			expectNodeState: mgmt.Follower}}

	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		reply, err := s.RequestVote(ctx, tc.request)
		time.Sleep(time.Microsecond * 300)
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
}
