package raftserver

import (
	"context"
	"log"
	"net"

	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raft"
	"google.golang.org/grpc"
)

type server struct {
	raft.UnimplementedRaftServer
	Node *node.Node
}

// RequestVote handles RPC vote requests from other nodes
func (s *server) RequestVote(ctx context.Context, v *raft.VoteRequest) (*raft.VoteReply, error) {
	log.Println("Received vote request:", v)
	return s.Node.HandleVote(v), nil
}

// RequestVote handles RPC log-append requests from other nodes
func (s *server) AppendLogs(ctx context.Context, a *raft.AppendRequest) (*raft.AppendReply, error) {
	log.Println("Received append request:", a)
	return s.Node.HandleAppend(a), nil
}

// StartRaftServer constructs and starts a gRPC server for Raft protocol routes
// Note: `port` must be in the form ":12345"
func StartRaftServer(port string, n *node.Node) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalln("Cluster interface failed to bind:", err)
	}
	s := grpc.NewServer()
	raft.RegisterRaftServer(s, &server{Node: n})
	if err := s.Serve(lis); err != nil {
		log.Fatalln("Failed to serve:", err)
	}
}
