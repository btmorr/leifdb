package types

// A LogRecord is a Raft log object, shipped to other servers to propagate writes
// The Record field is a string representing a database op, like "set thisKey someValue"
// or "del anotherKey"
type LogRecord struct {
	Term   int    `json:"term"`
	Record string `json:"record"`
}

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
	Term         int    `json:"term"`
	CandidateId  string `json:"candidateId"`
	LastLogIndex int    `json:"lastLogIndex"`
	LastLogTerm  int    `json:"lastLogTerm"`
}

// A VoteResponse is a response body template for the request-vote route
type VoteResponse struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"voteGranted"`
}

// An AppendBody is a request body template for the log-append route
type AppendBody struct {
	Term         int         `json:"term"`
	LeaderId     string      `json:"leaderId"`
	PrevLogIndex int         `json:"prevLogIndex"`
	PrevLogTerm  int         `json:"prevLogTerm"`
	Entries      []LogRecord `json:"entries"`
	LeaderCommit int         `json:"leaderCommit"`
}

// An AppendResponse is a response body template for the log-append route
type AppendResponse struct {
	Term    int  `json:"term"`
	Success bool `json:"success"`
}

// A HealthResponse is a response body template for the health route [note: the
//health endpoint takes a GET request, so there is no corresponding Body type]
type HealthResponse struct {
	Status string `json:"status"`
}
