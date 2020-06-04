package node

import (
	"context"
	"log"
	"os"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/btmorr/leifdb/internal/testutil"
	"github.com/btmorr/leifdb/internal/util"
	"google.golang.org/grpc"
)

// checkForeignNodeMock is used to skip membership checks during test, so that a
// Node will perform raft operations without creating a full multi-node config
func checkForeignNodeMock(addr string, known map[string]*ForeignNode) bool {
	return true
}

type MockRaftClient struct {
	cc raft.RaftClient
}

func (m *MockRaftClient) RequestVote(ctx context.Context, in *raft.VoteRequest, opts ...grpc.CallOption) (*raft.VoteReply, error) {
	return &raft.VoteReply{Term: in.Term, VoteGranted: true}, nil
}
func (m *MockRaftClient) AppendLogs(ctx context.Context, in *raft.AppendRequest, opts ...grpc.CallOption) (*raft.AppendReply, error) {
	return &raft.AppendReply{Term: in.Term, Success: true}, nil
}

// setupNode configures a Database and a Node for test, and creates
// a test directory that is automatically cleaned up after each test
func setupNode(t *testing.T) *Node {
	addr := "localhost:8080"

	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	store := db.NewDatabase()

	config := NewNodeConfig(testDir, addr, make([]string, 0, 0))
	n, _ := NewNode(config, store)
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

	config := NewNodeConfig(testDir, addr, make([]string, 0, 0))

	termRecord := &raft.TermRecord{Term: 5, VotedFor: "localhost:8181"}
	WriteTerm(config.TermFile, termRecord)

	termData := ReadTerm(config.TermFile)
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

	err := WriteLogs(config.LogFile, logCache)
	if err != nil {
		t.Error("Log write failure:", err)
	}
	_, err2 := os.Stat(config.LogFile)
	if err2 != nil {
		t.Error("LogFile does not exist after write:", err)
	}
	roundtrip := ReadLogs(config.LogFile)

	testutil.CompareLogs(t, "Roundtrip", roundtrip, logCache)

	store := db.NewDatabase()
	n, _ := NewNode(config, store)

	n.Halt()

	if n.Term != 5 {
		t.Error("Term not loaded correctly. Found term: ", n.Term)
	}

	testutil.CompareLogs(t, "Node load", n.Log, logCache)
}

type ReconcileTestCase struct {
	Name     string
	Store    *raft.LogStore
	Request  *raft.AppendRequest
	Expected *raft.LogStore
}

func TestReconcileLogs(t *testing.T) {
	emptyLog := &raft.LogStore{
		Entries: make([]*raft.LogRecord, 0, 0)}

	firstThree := []*raft.LogRecord{
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
			Term:   3,
			Action: raft.LogRecord_SET,
			Key:    "Hermione",
			Value:  "present"}}

	starterLog := &raft.LogStore{
		Entries: firstThree}

	nextTwo := []*raft.LogRecord{
		{
			Term:   5,
			Action: raft.LogRecord_DEL,
			Key:    "Harry"},
		{
			Term:   6,
			Action: raft.LogRecord_DEL,
			Key:    "Ron"}}

	appendLog := &raft.LogStore{
		Entries: append(firstThree, nextTwo...)}

	overlapLog := &raft.LogStore{
		Entries: append(firstThree[:2], nextTwo...)}

	testCases := []ReconcileTestCase{
		{
			Name:  "Empty mind, empty body",
			Store: emptyLog,
			Request: &raft.AppendRequest{
				Term:         0,
				LeaderId:     "localhost:8181",
				PrevLogIndex: -1,
				PrevLogTerm:  -1,
				LeaderCommit: -1,
				Entries:      []*raft.LogRecord{}},
			Expected: emptyLog},
		{
			Name:  "Empty mind, full body",
			Store: emptyLog,
			Request: &raft.AppendRequest{
				Term:         5,
				LeaderId:     "localhost:8181",
				PrevLogIndex: -1,
				PrevLogTerm:  -1,
				LeaderCommit: -1,
				Entries:      firstThree},
			Expected: starterLog},
		{
			Name:  "Full mind, full body",
			Store: starterLog,
			Request: &raft.AppendRequest{
				Term:         6,
				LeaderId:     "localhost:8181",
				PrevLogIndex: 2,
				PrevLogTerm:  3,
				LeaderCommit: -1,
				Entries:      nextTwo},
			Expected: appendLog},
		{
			Name:  "Incongruous mind",
			Store: starterLog,
			Request: &raft.AppendRequest{
				Term:         6,
				LeaderId:     "localhost:8181",
				PrevLogIndex: 1,
				PrevLogTerm:  2,
				LeaderCommit: -1,
				Entries:      nextTwo},
			Expected: overlapLog}}

	for _, tc := range testCases {
		result := reconcileLogs(tc.Store, tc.Request)
		testutil.CompareLogs(t, tc.Name, result, tc.Expected)
	}
}

// func TestAddNewLog(t *testing.T) {
// 	n := setupNode(t)
// 	// Simulating node in leader position, rather than adding a time.Sleep
// 	n.DoElection()

// }
