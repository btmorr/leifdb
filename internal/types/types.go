package types

import pb "github.com/btmorr/leifdb/internal/persistence"

// A Role is one of Leader, Candidate, or Follower
type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

// Data types for [un]marshalling JSON

// A WriteBody is a request body template for write route
type WriteBody struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// A DeleteBody is a request body template for delete route
type DeleteBody struct {
	Key string `json:"key"`
}

// A VoteBody is a request body template for the request-vote route
type VoteBody struct {
	Term         int64    `json:"term"`
	CandidateId  string `json:"candidateId"`
	LastLogIndex int    `json:"lastLogIndex"`
	LastLogTerm  int64    `json:"lastLogTerm"`
}

// A VoteResponse is a response body template for the request-vote route
type VoteResponse struct {
	Term        int64  `json:"term"`
	VoteGranted bool `json:"voteGranted"`
}

// An AppendBody is a request body template for the log-append route
type AppendBody struct {
	Term         int64         `json:"term"`
	LeaderId     string      `json:"leaderId"`
	PrevLogIndex int         `json:"prevLogIndex"`
	PrevLogTerm  int64         `json:"prevLogTerm"`
	Entries      []pb.LogRecord `json:"entries"`
	LeaderCommit int         `json:"leaderCommit"`
}

// An AppendResponse is a response body template for the log-append route
type AppendResponse struct {
	Term    int64  `json:"term"`
	Success bool `json:"success"`
}

// A HealthResponse is a response body template for the health route [note: the
//health endpoint takes a GET request, so there is no corresponding Body type]
type HealthResponse struct {
	Status string `json:"status"`
}
