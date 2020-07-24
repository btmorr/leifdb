// +build unit

package node

import (
	"log"
	"os"
	"testing"

	"github.com/rs/zerolog"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/btmorr/leifdb/internal/testutil"
	"github.com/btmorr/leifdb/internal/util"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

// checkForeignNodeMock is used to skip membership checks during test, so that
// a Node will perform raft operations without a full multi-node config
func checkForeignNodeMock(addr string, known map[string]*ForeignNode) bool {
	return true
}

// setupNode configures a Database and a Node for test, and creates
// a test directory that is automatically cleaned up after each test
func setupNode(t *testing.T) *Node {
	addr := "localhost:8080"
	clientAddr := "localhost:16990"

	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	store := db.NewDatabase()

	config := NewNodeConfig(testDir, addr, clientAddr, make([]string, 0, 0))
	n, _ := NewNode(config, store)
	n.CheckForeignNode = checkForeignNodeMock
	return n
}

func TestNewForeignNode(t *testing.T) {
	fn, err := NewForeignNode("localhost:12345")
	if err != nil {
		t.Errorf("Connection failed with: %v\n", err)
	}
	fn.Close()
}

func TestAvailabilityReport(t *testing.T) {
	n := setupNode(t)
	n.State = Leader

	host1 := "localhost:12345"
	host2 := "localhost:23456"
	avail, total := n.availability()
	if avail != 1 {
		t.Errorf("Initial availability should be 1 (self), got %d\n", avail)
	}
	if total != 1 {
		t.Errorf("Initial total should be 1 (self), got %d\n", total)
	}

	// add 1, no RPC functions have been called, availability should be true
	n.AddForeignNode(host1)
	avail, total = n.availability()
	if avail != 2 {
		t.Errorf("Availability with 1 other should be 2, got %d\n", avail)
	}
	if total != 2 {
		t.Errorf("Total with 1 other should be 2, got %d\n", total)
	}

	// add another and make an RPC call to one of them, since RPC is not mocked,
	// it will fail, so one will report unavailable
	n.AddForeignNode(host2)
	n.requestAppend(host2, n.Term)
	avail, total = n.availability()
	if avail != 2 {
		t.Errorf("Availability with 1 other up, 1 other down should be 2, got %d\n", avail)
	}
	if total != 3 {
		t.Errorf("Total with 2 others should be 3, got %d\n", total)
	}

	// make an RPC call to the other, now both others will be unavailable
	n.requestVote(host1)
	avail, total = n.availability()
	if avail != 1 {
		t.Errorf("Availability with 2 other down should be 1, got %d\n", avail)
	}
	if total != 3 {
		t.Errorf("Total with 2 others should be 3, got %d\n", total)
	}
	// todo: mock RPC response and verify that availability goes back to true
}

func TestPersistence(t *testing.T) {
	addr := "localhost:8080"
	clientAddr := "localhost:16990"

	testDir, _ := util.CreateTmpDir(".tmp-leifdb")
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	config := NewNodeConfig(testDir, addr, clientAddr, make([]string, 0, 0))

	termRecord := &raft.TermRecord{Term: 5, VotedFor: &raft.Node{
		Id:         "localhost:8181",
		ClientAddr: "localhost:80",
	}}
	WriteTerm(config.TermFile, termRecord)

	termData := ReadTerm(config.TermFile)
	if termData.Term != termRecord.Term {
		t.Error("Term data file roundtrip incorrect term:", termData.Term)
	}
	if termData.VotedFor.Id != termRecord.VotedFor.Id {
		t.Error("Term data file roundtrip incorrect vote:", termData.VotedFor.Id)
	}
	if termData.VotedFor.ClientAddr != termRecord.VotedFor.ClientAddr {
		t.Error("Term data file roundtrip incorrect Client Addr:", termData.VotedFor.ClientAddr)
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

	if n.Term != 5 {
		t.Error("Term not loaded correctly. Found term: ", n.Term)
	}

	testutil.CompareLogs(t, "Node load", n.Log, logCache)
}

type VoteTestCase struct {
	name       string
	request    *raft.VoteRequest
	expectTerm int64
	expectVote bool
}

func TestVote(t *testing.T) {
	n := setupNode(t)

	// set up node as if it is Leader with two logs committed
	n.State = Leader
	n.SetTerm(2, n.RaftNode)
	n.Log = &raft.LogStore{
		Entries: []*raft.LogRecord{
			{
				Term:   1,
				Action: raft.LogRecord_SET,
				Key:    "testing",
				Value:  "[1, 2, 3]",
			},
			{
				Term:   2,
				Action: raft.LogRecord_SET,
				Key:    "what'cha say?",
				Value:  "ah said...",
			},
		},
	}
	n.CommitIndex = 1

	testRaftNode := &raft.Node{Id: "localhost:16999", ClientAddr: "localhost:8089"}

	testCases := []VoteTestCase{
		{
			name: "Vote request expired term",
			request: &raft.VoteRequest{
				Term:         1,
				Candidate:    testRaftNode,
				LastLogIndex: 1,
				LastLogTerm:  2},
			expectTerm: 2,
			expectVote: false},
		{
			name: "Vote request same term",
			request: &raft.VoteRequest{
				Term:         2,
				Candidate:    testRaftNode,
				LastLogIndex: 1,
				LastLogTerm:  2},
			expectTerm: 3,
			expectVote: false},
		{
			name: "Vote request log behind",
			request: &raft.VoteRequest{
				Term:         4,
				Candidate:    testRaftNode,
				LastLogIndex: 0,
				LastLogTerm:  1},
			expectTerm: 3,
			expectVote: false},
		{
			name: "Vote request log incorrect (shouldn't happen)",
			request: &raft.VoteRequest{
				Term:         4,
				Candidate:    testRaftNode,
				LastLogIndex: 1,
				LastLogTerm:  1},
			expectTerm: 3,
			expectVote: false},
		{
			name: "Vote request valid, candidate equal",
			request: &raft.VoteRequest{
				Term:         4,
				Candidate:    testRaftNode,
				LastLogIndex: 1,
				LastLogTerm:  2},
			expectTerm: 4,
			expectVote: true},
		{
			name: "Vote request valid, candidate ahead",
			request: &raft.VoteRequest{
				Term:         6,
				Candidate:    testRaftNode,
				LastLogIndex: 7,
				LastLogTerm:  5},
			expectTerm: 6,
			expectVote: true,
		},
	}

	for _, tc := range testCases {
		reply := n.HandleVote(tc.request)
		if reply.Term != tc.expectTerm {
			t.Errorf("[%s] Expected term %d but got %d\n", tc.name, tc.expectTerm, reply.Term)
		}
		if reply.VoteGranted != tc.expectVote {
			t.Errorf("[%s] Expected vote %t but got %t\n", tc.name, tc.expectVote, reply.VoteGranted)
		}
	}
	// After test cases, node should have voted for `testRaftNode` and redirect to it
	redirectNode := n.RedirectLeader()
	if redirectNode != testRaftNode.ClientAddr {
		t.Errorf("Expected redirect to %s, but got %s\n", testRaftNode.ClientAddr, redirectNode)
	}
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

	raftNode := &raft.Node{
		Id:         "localhost:16991",
		ClientAddr: "localhost:8181",
	}

	testCases := []ReconcileTestCase{
		{
			Name:  "Empty log and request",
			Store: emptyLog,
			Request: &raft.AppendRequest{
				Term:         0,
				Leader:       raftNode,
				PrevLogIndex: -1,
				PrevLogTerm:  -1,
				LeaderCommit: -1,
				Entries:      []*raft.LogRecord{}},
			Expected: emptyLog},
		{
			Name:  "Empty log, populated request",
			Store: emptyLog,
			Request: &raft.AppendRequest{
				Term:         3,
				Leader:       raftNode,
				PrevLogIndex: -1,
				PrevLogTerm:  -1,
				LeaderCommit: -1,
				Entries:      firstThree},
			Expected: starterLog},
		{
			Name:  "Populated log and request",
			Store: starterLog,
			Request: &raft.AppendRequest{
				Term:         6,
				Leader:       raftNode,
				PrevLogIndex: 2,
				PrevLogTerm:  3,
				LeaderCommit: -1,
				Entries:      nextTwo},
			Expected: appendLog},
		{
			Name:  "Match but truncate",
			Store: appendLog,
			Request: &raft.AppendRequest{
				Term:         6,
				Leader:       raftNode,
				PrevLogIndex: 2,
				PrevLogTerm:  3,
				LeaderCommit: -1,
				Entries:      []*raft.LogRecord{nextTwo[0]}},
			Expected: &raft.LogStore{Entries: appendLog.Entries[:4]}},
		{
			Name:  "Mismatch and add",
			Store: starterLog,
			Request: &raft.AppendRequest{
				Term:         6,
				Leader:       raftNode,
				PrevLogIndex: 1,
				PrevLogTerm:  2,
				LeaderCommit: -1,
				Entries:      nextTwo},
			Expected: overlapLog}}

	for _, tc := range testCases {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Recovered panic in %s: %v", tc.Name, r)
			}
		}()
		result := reconcileLogs(tc.Store, tc.Request)
		testutil.CompareLogs(t, tc.Name, result, tc.Expected)
	}
}

type CommitTestCase struct {
	Name     string
	Store    *raft.LogStore
	Request  *raft.AppendRequest
	Expected map[string]string
}

func TestCommitLogs(t *testing.T) {
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

	currentTerm := int64(6)
	currentLead := &raft.Node{
		Id:         "localhost:8181",
		ClientAddr: "localhost:80",
	}

	testCases := []CommitTestCase{
		{
			Name:  "Append no commit",
			Store: emptyLog,
			Request: &raft.AppendRequest{
				Term:         currentTerm,
				Leader:       currentLead,
				PrevLogIndex: -1,
				PrevLogTerm:  -1,
				LeaderCommit: -1,
				Entries:      firstThree},
			Expected: map[string]string{
				"Harry":    "",
				"Ron":      "",
				"Hermione": ""}},
		{
			Name:  "Commit some, none new",
			Store: starterLog,
			Request: &raft.AppendRequest{
				Term:         currentTerm,
				Leader:       currentLead,
				PrevLogIndex: 2,
				PrevLogTerm:  3,
				LeaderCommit: 1,
				Entries:      []*raft.LogRecord{}},
			Expected: map[string]string{
				"Harry":    "present",
				"Ron":      "absent",
				"Hermione": ""}},
		{
			Name:  "Commit some, some new",
			Store: starterLog,
			Request: &raft.AppendRequest{
				Term:         currentTerm,
				Leader:       currentLead,
				PrevLogIndex: 2,
				PrevLogTerm:  3,
				LeaderCommit: 2,
				Entries:      nextTwo},
			Expected: map[string]string{
				"Harry":    "present",
				"Ron":      "absent",
				"Hermione": "present"}},
		{
			Name:  "Commit all",
			Store: starterLog,
			Request: &raft.AppendRequest{
				Term:         currentTerm,
				Leader:       currentLead,
				PrevLogIndex: 4,
				PrevLogTerm:  6,
				LeaderCommit: 4,
				Entries:      nextTwo},
			Expected: map[string]string{
				"Harry":    "",
				"Ron":      "",
				"Hermione": "present"}}}

	n := setupNode(t)
	n.SetTerm(currentTerm, currentLead)

	for _, tc := range testCases {
		n.HandleAppend(tc.Request)
		for k := range tc.Expected {
			v := n.Store.Get(k)
			if v != tc.Expected[k] {
				t.Errorf("[%s] Expected %s=%s got %s", tc.Name, k, tc.Expected[k], v)
			}
		}
	}
}

func TestUpdateTermViaAppend(t *testing.T) {
	n := setupNode(t)

	startTerm := int64(3)
	otherNode := &raft.Node{
		Id:         "localhost:8181",
		ClientAddr: "localhost:80",
	}
	n.SetTerm(startTerm, otherNode)

	newTerm := startTerm + 1
	req := &raft.AppendRequest{
		Term:         newTerm,
		Leader:       otherNode,
		PrevLogIndex: -1,
		PrevLogTerm:  -1,
		Entries:      make([]*raft.LogRecord, 0, 0),
		LeaderCommit: -1}
	reply := n.HandleAppend(req)
	if !reply.Success {
		t.Error("Expected append success")
	}
	if n.Term != newTerm {
		t.Errorf("Expected term %d but got %d", newTerm, n.Term)
	}
	if n.votedFor.Id != otherNode.Id {
		t.Errorf("Expected voted for %s but got %s", otherNode.Id, n.votedFor.Id)
	}
}

func TestCompactLogs(t *testing.T) {
	n := setupNode(t)

	// set up node as if it is Leader with two logs committed, one uncommitted
	n.State = Leader
	n.SetTerm(2, n.RaftNode)
	n.Log = &raft.LogStore{
		Entries: []*raft.LogRecord{
			{
				Term:   1,
				Action: raft.LogRecord_SET,
				Key:    "ah",
				Value:  "one",
			},
			{
				Term:   2,
				Action: raft.LogRecord_SET,
				Key:    "and ah",
				Value:  "two",
			},
			{
				Term:   2,
				Action: raft.LogRecord_SET,
				Key:    "and ah",
				Value:  "ONE TWO THREE FOUR!",
			},
		},
	}
	n.CommitIndex = 1

	snapshotIndex := n.CommitIndex
	snapshotTerm := n.Log.Entries[n.CommitIndex].Term
	appliedIndex := n.lastApplied

	err := n.CompactLogs(snapshotIndex, snapshotTerm)
	if err != nil {
		t.Error(err)
	}

	if n.IndexOffset != snapshotIndex+1 {
		t.Errorf("Expected index offset of %d, got %d\n", snapshotIndex+1, n.IndexOffset)
	}
	if n.LastSnapshotTerm != snapshotTerm {
		t.Errorf("Expected term of %d, got %d\n", snapshotTerm, n.LastSnapshotTerm)
	}
	remaining := len(n.Log.Entries)
	if remaining != 1 {
		t.Errorf("Expected compacted log to have 1 entry, found %d\n", remaining)
	}
	if n.CommitIndex != -1 {
		t.Errorf("Expected commit index of -1, got %d\n", n.CommitIndex)
	}
	sum := n.CommitIndex + n.IndexOffset
	if sum != snapshotIndex {
		t.Errorf("Sum of commit index and offset (%d) should match previous commit index (%d)\n", sum, snapshotIndex)
	}
	appliedSum := n.lastApplied + n.IndexOffset
	if appliedSum != appliedIndex {
		t.Errorf("Sum of last applied index and offset (%d) should match previous last applied index (%d)\n", appliedSum, appliedIndex)
	}
}
