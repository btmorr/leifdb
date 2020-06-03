package node

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// A ForeignNode is another member of the cluster, with connections needed
// to manage gRPC interaction with that node
type ForeignNode struct {
	Connection *grpc.ClientConn
	Client     raft.RaftClient
}

// NewForeignNode constructs a ForeignNode from an address ("host:port")
func NewForeignNode(address string) (*ForeignNode, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to %s", address)
		return nil, err
	}
	client := raft.NewRaftClient(conn)

	return &ForeignNode{Connection: conn, Client: client}, err
}

// Close cleans up the gRPC connection with the foreign node
func (f *ForeignNode) Close() {
	f.Connection.Close()
}

// A Role is one of Leader, Candidate, or Follower
type Role string

// A Follower is a read-only member of a cluster
// A Candidate solicits votes to become a Leader
// A Leader is a read/write member of a cluster
const (
	Follower  Role = "Follower"
	Candidate      = "Candidate"
	Leader         = "Leader"
)

// NodeConfig contains configurable properties for a node
type NodeConfig struct {
	Id       string
	DataDir  string
	TermFile string
	LogFile  string
}

// ForeignNodeChecker functions are used to determine if a request comes from
// a valid participant in a cluster. It should generally check against a
// configuration file or other canonical record of membership, but can also
// be mocked out for test to cause a Node to respond to RPC requests without
// creating a full multi-node deployment.
type ForeignNodeChecker func(string, map[string]*ForeignNode) bool

// A Node is one member of a Raft cluster, with all state needed to operate the
// algorithm's state machine. At any one time, its role may be Leader, Candidate,
// or Follower, and have different responsibilities depending on its role
type Node struct {
	Value            map[string]string
	NodeId           string
	ElectionTimeout  time.Duration
	electionTimer    *time.Timer
	appendTimeout    time.Duration
	appendTicker     *time.Ticker
	State            Role
	haltAppend       chan bool
	Term             int64
	votedFor         string
	otherNodes       map[string]*ForeignNode
	CheckForeignNode ForeignNodeChecker
	nextIndex        map[string]int64
	matchIndex       map[string]int64
	commitIndex      int64
	lastApplied      int64
	Log              *raft.LogStore
	config           NodeConfig
	Store            *db.Database
}

// Client methods for managing raft state

// Non-volatile state functions
// `Term`, `votedFor`, and `Log` must persist through application restart, so
// any request that changes these values must be written to disk before
// responding to the request.

// WriteTerm persists the node's most recent term and vote
func WriteTerm(filename string, termRecord *raft.TermRecord) error {
	out, err := proto.Marshal(termRecord)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to marshal term record")
		return err
	}
	_, err = os.Stat(filepath.Dir(filename))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed stat")
		return err
	}
	if err = ioutil.WriteFile(filename, out, 0644); err != nil {
		log.Fatal().Err(err).Msg("Failed to write term file")
	}
	return err
}

// ReadTerm attempts to unmarshal and return a TermRecord from the specified
// file, and if unable to do so returns an initialized TermRecord
func ReadTerm(filename string) *raft.TermRecord {
	record := &raft.TermRecord{Term: 0, VotedFor: ""}
	_, err := os.Stat(filename)
	if err == nil {
		termFile, _ := ioutil.ReadFile(filename)
		if err = proto.Unmarshal(termFile, record); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal term file")
		}
	}
	return record
}

// SetTerm records term and vote in non-volatile state
func (n *Node) SetTerm(newTerm int64, votedFor string) error {
	n.Term = newTerm
	n.votedFor = votedFor
	vote := &raft.TermRecord{
		Term:     n.Term,
		VotedFor: n.votedFor}
	return WriteTerm(n.config.TermFile, vote)
}

// WriteLogs persists the node's log
func WriteLogs(filename string, logStore *raft.LogStore) error {
	out, err := proto.Marshal(logStore)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to marshal logs")
	}
	if err = ioutil.WriteFile(filename, out, 0644); err != nil {
		log.Fatal().Err(err).Msg("Failed to write log file")
	}
	return err
}

// ReadLogs attempts to unmarshal and return a LogStore from the specified
// file, and if unable to do so returns an empty LogStore
func ReadLogs(filename string) *raft.LogStore {
	logStore := &raft.LogStore{Entries: make([]*raft.LogRecord, 0, 0)}
	_, err := os.Stat(filename)
	if err != nil {
	} else {
		logFile, _ := ioutil.ReadFile(filename)
		if err = proto.Unmarshal(logFile, logStore); err != nil {
			log.Error().
				Err(err).
				Msg("Failed to unmarshal log file, creating empty log store")
		}
	}
	return logStore
}

// SetLog records new log contents in non-volatile state
func (n *Node) SetLog(newLog []*raft.LogRecord) error {
	record := &raft.LogStore{Entries: newLog}
	err := WriteLogs(n.config.LogFile, record)
	if err == nil {
		n.Log = record
	}
	return err
}

// Volatile state functions
// Other state should not be written to disk, and should be re-initialized on
// restart, but may have other side-effects that happen on state change

// SetState designates the Node as one of the roles in the role enumeration,
// and handles any side-effects that should happen specifically on state
// transition
func (n *Node) SetState(newState Role) {
	n.State = newState
	// todo: move starting/stopping election timer and append ticker here
}

// When a Raft node's role is "leader", startAppendTicker periodically send out
// an append-logs request to each other node on a period shorter than any node's
// election timeout
func (n *Node) startAppendTicker() {
	go func() {
		for {
			select {
			case <-n.haltAppend:
				log.Debug().Msg("No longer leader, halting log append")
				// defer?
				n.resetElectionTimer()
				return
			case <-n.appendTicker.C:
				// placeholder for generating append requests
				fmt.Print(".")
				continue
			}
		}
	}()
}

// requestVote sends a request for vote to a single other node (see `DoElection`)
func (n *Node) requestVote(host string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	voteRequest := &raft.VoteRequest{
		Term:         n.Term,
		CandidateId:  n.NodeId,
		LastLogIndex: 0,
		LastLogTerm:  0}

	vote, err := n.otherNodes[host].Client.RequestVote(ctx, voteRequest)
	if err != nil {
		log.Error().Err(err).Msgf("Error requesting vote from %sv", host)
	}

	return vote.VoteGranted, err
}

// DoElection sends out requests for votes to each other node in the Raft cluster.
// When a Raft node's role is "candidate", it should send start an election. If it
// is granted votes from a majority of nodes, its role changes to "leader". If it
// receives an append-logs message during the election from a node with a term higher
// than this node's current term, its role changes to "follower". If it does not
// receive a majority of votes and also does not receive an append-logs from a valid
// leader, it increments the term and starts another election (repeat until a leader
// is elected).
func (n *Node) DoElection() {
	log.Trace().Msg("Starting Election")
	n.SetState(Candidate)

	n.SetTerm(n.Term+1, n.NodeId)

	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1

	log.Info().
		Int64("Term", n.Term).
		Int("clusterSize", len(n.otherNodes)).
		Int("needed", majority).
		Msg("Becoming candidate")

	n.resetElectionTimer()
	numVotes := 1
	for k := range n.otherNodes {
		_, err := n.requestVote(k)
		log.Trace().Msg("got a vote")
		if err == nil {
			log.Trace().Msg("it's a 'yay'")
			numVotes = numVotes + 1
		}
	}
	voteLog := log.Info().
		Int("needed", majority).
		Int("got", numVotes)
	if numVotes >= majority {
		voteLog.Bool("success", true).Msg("Election succeeded")
		n.SetState(Leader)
		log.Trace().Msg("Becoming leader")

		n.electionTimer.Stop()
		select {
		case <-n.electionTimer.C:
		default:
		}
		log.Debug().Msg("Stopping election timer, starting append ticker")
		n.startAppendTicker()
	} else {
		voteLog.Bool("success", false).Msg("Election failed")
		n.SetState(Follower)
		log.Trace().Msg("Becoming follower")
	}
}

// NewNodeConfig creates a config for a Node
func NewNodeConfig(dataDir string, addr string) NodeConfig {
	// todo: check dataDir. if it is empty, initialize Node with default
	// values for non-volatile state. Otherwise, read values.
	return NodeConfig{
		Id:       addr,
		DataDir:  dataDir,
		TermFile: filepath.Join(dataDir, "term"),
		LogFile:  filepath.Join(dataDir, "raftlog")}
}

// checkForeignNode verifies that a node is a known member of the cluster (this
// is the expected checker for a Node, but is extracted so it can be mocked)
func checkForeignNode(addr string, known map[string]*ForeignNode) bool {
	_, ok := known[addr]
	return ok
}

// NewNode initializes a Node with a randomized election timeout between
// 150-300ms, and starts the election timer
func NewNode(config NodeConfig, store *db.Database) (*Node, error) {
	lowerBound := 150
	upperBound := 300
	ms := (rand.Int() % lowerBound) + (upperBound - lowerBound)
	electionTimeout := time.Duration(ms) * time.Millisecond

	appendTimeout := time.Duration(10) * time.Millisecond

	termRecord := ReadTerm(config.TermFile)
	logStore := ReadLogs(config.LogFile)

	log.Info().
		Int64("Term", termRecord.Term).
		Str("Vote", termRecord.VotedFor).
		Int("nLogs", len(logStore.Entries)).
		Msg("On load")

	n := Node{
		NodeId:           config.Id,
		ElectionTimeout:  electionTimeout,
		electionTimer:    time.NewTimer(electionTimeout),
		appendTimeout:    appendTimeout,
		appendTicker:     time.NewTicker(appendTimeout),
		State:            Follower,
		haltAppend:       make(chan bool),
		Term:             termRecord.Term,
		votedFor:         termRecord.VotedFor,
		otherNodes:       make(map[string]*ForeignNode),
		CheckForeignNode: checkForeignNode,
		nextIndex:        make(map[string]int64),
		matchIndex:       make(map[string]int64),
		commitIndex:      0,
		lastApplied:      0,
		Log:              logStore,
		config:           config,
		Store:            store}

	go func() {
		log.Trace().Msg("First election timer")
		<-n.electionTimer.C
		n.DoElection()
	}()

	return &n, nil
}

// Halt is used to stop all timers (primarily in test)
func (n *Node) Halt() {
	log.Trace().Msg("Halting")
	if !n.electionTimer.Stop() {
		select {
		case <-n.electionTimer.C:
		default:
		}
	}

	n.appendTicker.Stop()
	select {
	case <-n.appendTicker.C:
	default:
	}
}

// resetElectionTimer restarts the countdown to election when a Raft node is a
// follower or candidate and receives a message from a valid leader
func (n *Node) resetElectionTimer() {
	if !n.electionTimer.Stop() {
		select {
		case <-n.electionTimer.C:
		default:
		}
	}
	n.electionTimer.Reset(n.ElectionTimeout)

	go func() {
		<-n.electionTimer.C
		n.DoElection()
	}()
}

// AddForeignNode updates the list of known other members of the raft cluster
func (n *Node) AddForeignNode(addr string) {
	n.otherNodes[addr], _ = NewForeignNode(addr)
	log.Info().Msgf("Added %s to known nodes", addr)
}

// HandleVote responds to vote requests from candidate nodes
func (n *Node) HandleVote(req *raft.VoteRequest) *raft.VoteReply {
	log.Info().Msgf("%s proposed term: %d", req.CandidateId, req.Term)
	var vote bool
	var msg string
	if req.Term <= n.Term {
		// Increment term, vote for same node as previous term (is this correct?)
		n.SetTerm(n.Term+1, n.votedFor)
		vote = false
		msg = "Expired term vote received, incrementing term"
	} else if !n.CheckForeignNode(req.CandidateId, n.otherNodes) {
		vote = false
		msg = "Unknown foreign node: " + req.CandidateId
	} else {
		msg = "Voting"
		// todo: check candidate's log details
		vote = true

		n.SetTerm(req.Term, req.CandidateId)
		if n.State == Leader {
			n.haltAppend <- true
		}
		n.SetState(Follower)
		// defer?
		n.resetElectionTimer()
	}
	log.Info().
		Int64("Term", n.Term).
		Bool("Granted", vote).
		Msg(msg)
	return &raft.VoteReply{Term: n.Term, VoteGranted: vote}
}

// validateAppend performs all checks for valid append request
func (n *Node) validateAppend(term int64, leaderId string) bool {
	var success bool
	success = true
	// reply false if req term < current term
	if term < n.Term {
		success = false
	}
	if leaderId != n.votedFor {
		msg1 := "Append request from LeaderId mismatch for this term. "
		msg2 := "Got: " + leaderId + " (voted for: " + n.votedFor + "). "
		msg3 := "Has the configuration changed?"
		log.Error().Msg(msg1 + msg2 + msg3)
		success = false
	}
	return success
}

// If an existing entry conflicts with a new one (same idx diff term),
// reconcileLogs deletes the existing entry and any that follow
func (n *Node) reconcileLogs(body *raft.AppendRequest) {
	// note: don't memoize length of Entries, it changes multiple times
	// during this method--safer to recalculate, and memoizing would
	// only save a maximum of one pass so it's not worth it
	var mismatchIdx int64
	mismatchIdx = -1
	if body.PrevLogIndex < int64(len(n.Log.Entries)) {
		overlappingEntries := n.Log.Entries[body.PrevLogIndex:]
		for i, rec := range overlappingEntries {
			if rec.Term != body.Entries[i].Term {
				mismatchIdx = body.PrevLogIndex + int64(i)
				break
			}
		}
	}
	if mismatchIdx >= 0 {
		log.Debug().Msgf("Mismatch index: %d - rewinding log", mismatchIdx)
		n.Log.Entries = n.Log.Entries[:mismatchIdx]
	}
	// append any entries not already in log
	offset := int64(len(n.Log.Entries)) - body.PrevLogIndex - 1
	newLogs := body.Entries[offset:]
	log.Debug().Msgf("Appending %d entries", len(newLogs))
	n.Log.Entries = append(n.Log.Entries, newLogs...)
}

// applyCommittedLogs updates the database with actions that have not yet been
// applied, up to the new commit index
func (n *Node) applyCommittedLogs(commitIdx int64) {
	log.Debug().
		Int64("current", n.commitIndex).
		Int64("leader", commitIdx).
		Msg("apply commits")
	if commitIdx > n.commitIndex {
		// apply all entries up to new commit index to store
		for i := n.commitIndex; i < commitIdx; i++ {
			action := n.Log.Entries[i].Action
			key := n.Log.Entries[i].Key
			if action == raft.LogRecord_SET {
				value := n.Log.Entries[i].Value
				n.Store.Set(key, value)
			} else if action == raft.LogRecord_DEL {
				n.Store.Delete(key)
			}
		}
		lastIndex := int64(len(n.Log.Entries))
		if commitIdx > lastIndex {
			commitIdx = lastIndex
		}
		n.commitIndex = commitIdx
		log.Debug().
			Int64("commit", n.commitIndex).
			Msg("Commit updated")
	}
}

// checkPrevious returns true if Node.logs contains an entry at the specified
// index with the specified term, otherwise false
func (n *Node) checkPrevious(prevIndex int64, prevTerm int64) bool {
	return prevIndex < int64(len(n.Log.Entries)) && n.Log.Entries[prevIndex].Term == prevTerm
}

// HandleAppend responds to append-log messages from leader nodes
func (n *Node) HandleAppend(req *raft.AppendRequest) *raft.AppendReply {
	var success bool

	valid := n.validateAppend(req.Term, req.LeaderId)
	matched := n.checkPrevious(req.PrevLogIndex, req.PrevLogTerm)
	if !valid {
		// Invalid request
		success = false
	} else if !matched {
		// Valid request, but earlier entries needed
		success = false
	} else {
		// Valid request, and all required logs present
		if len(req.Entries) > 0 {
			n.reconcileLogs(req)
		}
		n.applyCommittedLogs(req.LeaderCommit)
		success = true
	}
	if valid {
		// For any valid append received during an election, cancel election
		if n.State != Follower {
			n.SetState(Follower)
		}
		// only reset the election timer on append from a valid leader
		// defer?
		n.resetElectionTimer()
	}
	// finally
	return &raft.AppendReply{Term: n.Term, Success: success}
}
