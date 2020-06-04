package node

import (
	"context"
	"log"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/raft"
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

func TestAddNewLog(t *testing.T) {
	n := setupNode(t)
	// Simulating node in leader position, rather than adding a time.Sleep
	n.DoElection()

}
