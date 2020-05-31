package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

type ForeignNode struct {
	Connection *grpc.ClientConn
	Client     raft.RaftClient
}

func NewForeignNode(address string) (*ForeignNode, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Println("Failed to connect to", address, "-", err)
		return nil, err
	}
	client := raft.NewRaftClient(conn)

	return &ForeignNode{Connection: conn, Client: client}, err
}

func (f *ForeignNode) Close() {
	f.Connection.Close()
}

type role int

// A role is one of Leader, Candidate, or Follower
const (
	Follower  role = iota // A Follower is a read-only member of a cluster
	Candidate             // A Candidate solicits votes to become a Leader
	Leader                // A Leader is a read/write member of a cluster
)

// NodeConfig contains configurable properties for a node
type NodeConfig struct {
	Id       string
	DataDir  string
	TermFile string
	LogFile  string
}

// A Node is one member of a Raft cluster, with all state needed to operate the
// algorithm's state machine. At any one time, its role may be Leader, Candidate,
// or Follower, and have different responsibilities depending on its role
type Node struct {
	Value           map[string]string
	NodeId          string
	ElectionTimeout time.Duration
	electionTimer   *time.Timer
	appendTimeout   time.Duration
	appendTicker    *time.Ticker
	State           role
	haltAppend      chan bool
	Term            int64
	votedFor        string
	otherNodes      map[string]*ForeignNode
	nextIndex       map[string]int64
	matchIndex      map[string]int64
	commitIndex     int64
	lastApplied     int64
	Log             *raft.LogStore
	config          NodeConfig
	Store           *db.Database
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
		log.Fatalln("Failed to marshal term record:", err)
		return err
	}
	_, err = os.Stat(filepath.Dir(filename))
	if err != nil {
		log.Fatalln("Failed stat:", err)
		return err
	}
	if err = ioutil.WriteFile(filename, out, 0644); err != nil {
		log.Fatalln("Failed to write term file:", err)
	}
	return err
}

// ReadTerm attempts to unmarshal and return a TermRecord from the specified
// file, and if unable to do so returns an initialized TermRecord
func ReadTerm(filename string) *raft.TermRecord {
	record := &raft.TermRecord{Term: 0, VotedFor: ""}
	_, err := os.Stat(filename)
	if err != nil {
	} else {
		termFile, _ := ioutil.ReadFile(filename)
		if err = proto.Unmarshal(termFile, record); err != nil {
			log.Println("Failed to unmarshal term file:", err)
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
		log.Fatalln("Failed to marshal logs:", err)
	}
	if err = ioutil.WriteFile(filename, out, 0644); err != nil {
		log.Fatalln("Failed to write log file:", err)
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
			log.Println("Failed to unmarshal log file, creating empty log store:", err)
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
func (n *Node) SetState(newState role) {
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
				log.Println("No longer leader. Halting log append...")
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

// requestVote sends a request for vote to a single other node (see `doElection`)
func (n Node) requestVote(host string, term int64) (bool, error) {
	uri := "http://" + host + "/vote"
	log.Println("Requesting vote from ", host)

	body := raft.VoteRequest{
		Term:         term,
		CandidateId:  n.NodeId,
		LastLogIndex: 0,
		LastLogTerm:  0}

	b, err0 := json.Marshal(body)
	if err0 != nil {
		return false, err0
	}
	br := bytes.NewReader(b)

	resp, err1 := http.Post(uri, "application/json", br)
	if err1 != nil {
		return false, err1
	}

	raw, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		return false, err2
	}

	var vote raft.VoteReply
	err3 := json.Unmarshal(raw, &vote)
	if err3 != nil {
		return false, err3
	}

	return vote.VoteGranted, err3
}

// doElection sends out requests for votes to each other node in the Raft cluster.
// When a Raft node's role is "candidate", it should send start an election. If it
// is granted votes from a majority of nodes, its role changes to "leader". If it
// receives an append-logs message during the election from a node with a term higher
// than this node's current term, its role changes to "follower". If it does not
// receive a majority of votes and also does not receive an append-logs from a valid
// leader, it increments the term and starts another election (repeat until a leader
// is elected).
func (n *Node) doElection() {
	log.Println("Starting Election")
	n.SetState(Candidate)
	log.Println("Becoming candidate")

	n.SetTerm(n.Term+1, n.NodeId)

	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1

	log.Println("\tNew Term: ", n.Term)
	log.Println("\tN other nodes: ", len(n.otherNodes))
	log.Println("\tVotes needed: ", majority)

	n.resetElectionTimer()
	numVotes := 1
	for k := range n.otherNodes {
		_, err := n.requestVote(k, n.Term)
		log.Println("got a vote")
		if err == nil {
			log.Println("it's a 'yay'")
			numVotes = numVotes + 1
		}
	}
	if numVotes >= majority {
		log.Println(
			"Election succeeded [",
			numVotes, "out of", majority,
			"needed]")
		n.SetState(Leader)
		log.Println("Becoming leader")

		n.electionTimer.Stop()
		select {
		case <-n.electionTimer.C:
		default:
		}
		log.Println("Stopping election timer, starting append ticker")
		n.startAppendTicker()
	} else {
		log.Println(
			"Election failed [",
			numVotes, "out of", majority,
			"needed]")
		n.SetState(Follower)
		log.Println("Becoming follower")
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

// NewNode initializes a Node with a randomized election timeout between
// 150-300ms, and starts the election timer
func NewNode(config NodeConfig, store *db.Database) (*Node, error) {
	lowerBound := 150
	upperBound := 300
	ms := (rand.Int() % lowerBound) + (upperBound - lowerBound)
	electionTimeout := time.Duration(ms) * time.Millisecond

	appendTimeout := time.Duration(10) * time.Millisecond

	termRecord := ReadTerm(config.TermFile)
	log.Println("Current term:", termRecord.Term, "- Voted for:", termRecord.VotedFor)
	logStore := ReadLogs(config.LogFile)
	log.Println("Loaded", len(logStore.Entries), "logs")

	n := Node{
		NodeId:          config.Id,
		ElectionTimeout: electionTimeout,
		electionTimer:   time.NewTimer(electionTimeout),
		appendTimeout:   appendTimeout,
		appendTicker:    time.NewTicker(appendTimeout),
		State:           Follower,
		haltAppend:      make(chan bool),
		Term:            termRecord.Term,
		votedFor:        termRecord.VotedFor,
		otherNodes:      make(map[string]*ForeignNode),
		nextIndex:       make(map[string]int64),
		matchIndex:      make(map[string]int64),
		commitIndex:     0,
		lastApplied:     0,
		Log:             logStore,
		config:          config,
		Store:           store}

	go func() {
		// log.Println("First election timer")
		<-n.electionTimer.C
		n.doElection()
	}()

	return &n, nil
}

// When a Raft node is a follower or candidate and receives a message from a
// valid leader, it should reset its election countdown timer
func (n *Node) resetElectionTimer() {
	log.Println("Restarting election timer")

	if !n.electionTimer.Stop() {
		select {
		case <-n.electionTimer.C:
		default:
		}
	}
	n.electionTimer.Reset(n.ElectionTimeout)

	go func() {
		<-n.electionTimer.C
		n.doElection()
	}()
}

// AddForeignNode updates the list of known other members of the raft cluster
func (n *Node) AddForeignNode(addr string) {
	n.otherNodes[addr], _ = NewForeignNode(addr)
	log.Println("Added", addr, "to known nodes")
	log.Println("Nodes:", n.otherNodes)
}

// HandleVote responds to vote requests from candidate nodes
func (n *Node) HandleVote(req *raft.VoteRequest) *raft.VoteReply {
	log.Println(req.CandidateId, " proposed term: ", req.Term)
	var vote bool
	if req.Term <= n.Term {
		// Increment term, vote for same node as previous term (is this correct?)
		log.Println("Expired term vote received. New term:", n.Term)
		n.SetTerm(n.Term+1, n.votedFor)
		vote = false
	} else if _, ok := n.otherNodes[req.CandidateId]; !ok {
		log.Println("Unknown foreign node:", req.CandidateId)
		vote = false
	} else {
		log.Println("Voting for", req.CandidateId, "for term", req.Term)
		n.SetTerm(req.Term, req.CandidateId)
		if n.State == Leader {
			n.haltAppend <- true
		}
		n.SetState(Follower)
		n.resetElectionTimer()

		// todo: check candidate's log details
		vote = true
	}
	// log.Println("Returning vote: [", status, " ]", vote)
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
		byzMsg1 := "Append request from LeaderId mismatch for this term. "
		byzMsg2 := "Possible byzantine actor: " + leaderId
		byzMsg3 := " (voted for: " + n.votedFor + ")"
		log.Println(byzMsg1 + byzMsg2 + byzMsg3)
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
		log.Println("Mismatch index:", mismatchIdx, "- rewinding log")
		n.Log.Entries = n.Log.Entries[:mismatchIdx]
	}
	// append any entries not already in log
	offset := int64(len(n.Log.Entries)) - body.PrevLogIndex - 1
	newLogs := body.Entries[offset:]
	log.Println("Appending", len(newLogs), "entries")
	n.Log.Entries = append(n.Log.Entries, newLogs...)
}

func (n *Node) applyCommittedLogs(commitIdx int64) {
	log.Println("Current commit:", n.commitIndex, "- Leader commit:", commitIdx)
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
		log.Println("New commit:", n.commitIndex)
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
		n.resetElectionTimer()
	}
	// finally
	return &raft.AppendReply{Term: n.Term, Success: success}
}
