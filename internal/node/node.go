package node

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/raft"
)

// Role is either Leader or Follower
type Role string

// Follower is a read-only member of a cluster
// Leader is a read/write member of a cluster
const (
	Leader   Role = "Leader"
	Follower      = "Follower"
)

var (
	// ErrNotLeaderRecv indicates that a client attempted to make a write to a
	// node that is not currently the leader of the cluster
	ErrNotLeaderRecv = errors.New("Cannot accept writes if not leader")

	// ErrNotLeaderSend indicates that a server attempted to send an append
	// request while it is not the leader of the cluster
	ErrNotLeaderSend = errors.New("Cannot send log append request if not leader")

	// ErrExpiredTerm indicates that an append request was generated for a past
	// term, so it should not be sent
	ErrExpiredTerm = errors.New("Do not send append requests for expired terms")

	// ErrAppendFailed indicates that an append job ran out of retry attempts
	// without successfully appending to a majority of nodes
	ErrAppendFailed = errors.New("Failed to append logs to a majority of nodes")

	// ErrCommitFailed indicates that the leader's commit index after append
	// is less than the index of the record being added
	ErrCommitFailed = errors.New("Failed to commit record")

	// ErrAppendRangeMet indicates that reverse-iteration has reached the
	// beginning of the log and still not gotten a response--aborting
	ErrAppendRangeMet = errors.New("Append range reached, not trying again")
)

// A ForeignNode is another member of the cluster, with connections needed
// to manage gRPC interaction with that node and track recent availability
type ForeignNode struct {
	Connection *grpc.ClientConn
	Client     raft.RaftClient
	NextIndex  int64
	MatchIndex int64
	Available  bool
}

// NewForeignNode constructs a ForeignNode from an address ("host:port")
func NewForeignNode(address string) (*ForeignNode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithInsecure())
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to %s", address)
		return nil, err
	}

	client := raft.NewRaftClient(conn)

	return &ForeignNode{
		Connection: conn,
		Client:     client,
		NextIndex:  0,
		MatchIndex: -1,
		Available:  true,
	}, err
}

// Close cleans up the gRPC connection with the foreign node
func (f *ForeignNode) Close() {
	f.Connection.Close()
}

// NodeConfig contains configurable properties for a node
type NodeConfig struct {
	Id         string
	ClientAddr string
	DataDir    string
	TermFile   string
	LogFile    string
	NodeIds    []string
}

// ForeignNodeChecker functions are used to determine if a request comes from
// a valid participant in a cluster. It should generally check against a
// configuration file or other canonical record of membership, but can also
// be mocked out for test to cause a Node to respond to RPC requests without
// creating a full multi-node deployment.
type ForeignNodeChecker func(string, map[string]*ForeignNode) bool

// A Node is one member of a Raft cluster, with all state needed to operate the
// algorithm's state machine. At any time, its role may be Leader, Candidate,
// or Follower, and have different responsibilities depending on its role (note
// that Candidate is a virtual role--a Candidate does not behave differently
// from a Follower w.r.t. incoming messages, so the node will remain in the
// Follower state while an election is in progress)
type Node struct {
	RaftNode         *raft.Node
	State            Role
	Term             int64
	votedFor         *raft.Node
	Reset            chan bool
	otherNodes       map[string]*ForeignNode
	CheckForeignNode ForeignNodeChecker
	AllowVote        bool
	CommitIndex      int64
	lastApplied      int64
	Log              *raft.LogStore
	config           NodeConfig
	Store            *db.Database
	sync.Mutex
}

// Non-volatile state functions
// `Term`, `votedFor`, and `Log` must persist through application restart, so
// any request that changes these values must be written to disk before
// responding to the request.

// RedirectLeader provides the leader which we want to redirect requests to if
// we are not the leader at present
func (n *Node) RedirectLeader() string {
	if n.votedFor == nil {
		return ""
	}
	return n.votedFor.ClientAddr
}

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
	record := &raft.TermRecord{Term: 0, VotedFor: nil}
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
func (n *Node) SetTerm(newTerm int64, votedFor *raft.Node) error {
	n.Term = newTerm
	n.votedFor = votedFor
	vote := &raft.TermRecord{
		Term:     newTerm,
		VotedFor: votedFor}
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

// resetElectionTimer ensures that the node's state is Follower, and sends a
// signal to the reset channel (read by the StateManager, which controls the
// timers used for elections)
func (n *Node) resetElectionTimer() {
	n.State = Follower
	go func() {
		n.Reset <- true
	}()
}

// setLog records new log contents in non-volatile state, and returns the index
// of the record in the log, or an error
func (n *Node) setLog(newLogs []*raft.LogRecord) (int64, error) {
	record := &raft.LogStore{Entries: newLogs}
	idx := int64(len(record.Entries) - 1)
	err := WriteLogs(n.config.LogFile, record)
	if err == nil {
		n.Log = record
	}
	return idx, err
}

// applyRecord adds a new record to the log, then sends an append-logs request
// to other nodes in the cluster. This method does not return until either the
// log is successfully committed to a majority of nodes, or a majority of
// nodes fail via explicit rejection or timeout (which should generally result
// in an election)
func (n *Node) applyRecord(record *raft.LogRecord) error {
	if n.State != Leader {
		return ErrNotLeaderRecv
	}
	newEntries := append(n.Log.Entries, record)
	idx, err := n.setLog(newEntries)
	if err != nil {
		log.Error().Err(err).Msg("applyRecord: Error setting log")
		return err
	}
	// Try appending logs to other nodes, with 3 retries
	currentTerm := n.Term
	err = n.SendAppend(3, currentTerm)
	if err != nil {
		log.Error().Err(err).Msg("applyRecord: Error shipping log")
		return err
	}
	// verify that n.CommitIndex >= idx
	if n.CommitIndex < idx {
		log.Error().Err(ErrCommitFailed).
			Int64("recordIndex", idx).
			Int64("CommitIndex", n.CommitIndex).
			Msg("Commit index failed to update after append")
		return ErrCommitFailed
	}
	// return once entry is applied to state machine or error
	return err
}

// Client methods for managing raft state

// Set appends a write entry to the log record, and returns once the update is
// applied to the state machine or an error is generated
func (n *Node) Set(key string, value string) error {
	log.Info().
		Str("key", key).
		Str("value", value).
		Msg("Set")
	record := &raft.LogRecord{
		Term:   n.Term,
		Action: raft.LogRecord_SET,
		Key:    key,
		Value:  value}
	n.Lock()
	defer n.Unlock()
	return n.applyRecord(record)
}

// Delete appends a delete entry to the log record, and returns once the update
// is applied to the state machine or an error is generated
func (n *Node) Delete(key string) error {
	log.Info().
		Str("key", key).
		Msg("Delete")
	record := &raft.LogRecord{
		Term:   n.Term,
		Action: raft.LogRecord_DEL,
		Key:    key}
	n.Lock()
	defer n.Unlock()
	return n.applyRecord(record)
}

// requestVote sends a request for vote to a single other node (see DoElection)
func (n *Node) requestVote(host string) (*raft.VoteReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*4)
	defer cancel()

	lastLogIndex := int64(len(n.Log.Entries)) - 1
	var lastLogTerm int64
	if lastLogIndex >= 0 {
		lastLogTerm = n.Log.Entries[lastLogIndex].Term
	} else {
		lastLogTerm = 0
	}
	voteRequest := &raft.VoteRequest{
		Term:         n.Term,
		Candidate:    n.RaftNode,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm}

	vote, err := n.otherNodes[host].Client.RequestVote(ctx, voteRequest)
	if err != nil {
		log.Warn().Err(err).Msgf("Error requesting vote from %s", host)
		n.otherNodes[host].Available = false
	} else {
		n.otherNodes[host].Available = true
	}

	return vote, err
}

// DoElection sends out requests for votes to each other node in the Raft
// cluster. When a Raft node's role is "candidate", it should send start an
// election. If it is granted votes from a majority of nodes, its role changes
// to "leader". If it receives an append-logs message during the election from
// a node with a term higher than this node's current term, its role changes to
// "follower". If it does not receive a majority of votes and also does not
// receive an append-logs from a valid leader, it increments the term and
// starts another election (repeat until a leader is elected).
func (n *Node) DoElection() bool {
	log.Trace().Msg("Starting Election")
	n.SetTerm(n.Term+1, n.RaftNode)

	numNodes := len(n.otherNodes) + 1
	majority := (numNodes / 2) + 1
	var success bool

	log.Info().
		Int64("Term", n.Term).
		Int("clusterSize", numNodes).
		Int("needed", majority).
		Msg("Becoming candidate")

	numVotes := 1
	maxTermSeen := n.Term
	maxTermSeenSource := n.votedFor

	var wg sync.WaitGroup
	wg.Add(len(n.otherNodes))

	for k := range n.otherNodes {
		// if needed for performance, figure out how to collect the term responses in a thread-safe way
		go func(k string) {
			defer wg.Done()

			vote, err := n.requestVote(k)
			log.Trace().Msg("got a vote")
			if err != nil {
				return
			}
			if vote.VoteGranted {
				log.Trace().Msg("it's a 'yay'")
				numVotes++
			} else {
				if vote.Term > maxTermSeen {
					maxTermSeen = vote.Term
					maxTermSeenSource = vote.Node
				}
			}
		}(k)
	}

	wg.Wait()

	voteLog := log.Info().
		Int("needed", majority).
		Int("got", numVotes)

	if numVotes < majority {
		voteLog.
			Bool("success", false).
			Int64("term", n.Term).
			Msg("Election failed")
		success = false
		if maxTermSeen > n.Term {
			log.Info().
				Int64("max response term", maxTermSeen).
				Str("other node", maxTermSeenSource.Id).
				Msg("Updating term to max seen")
			n.SetTerm(maxTermSeen, maxTermSeenSource)
		}
	} else {
		voteLog.
			Bool("success", true).
			Int64("term", n.Term).
			Msg("Election succeeded")
		n.State = Leader
		success = true
		// StateManager grace window job sets this back to true
		n.AllowVote = false

		for k := range n.otherNodes {
			n.otherNodes[k].MatchIndex = -1
			n.otherNodes[k].NextIndex = int64(len(n.Log.Entries))
		}
	}
	return success
}

// commitRecords iterates backward from last index of log entries, and finds
// latest index that has been appended to a majority of nodes, and updates
// the database and node CommitIndex
func (n *Node) commitRecords() {
	log.Trace().Msg("commitRecords")

	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1
	log.Trace().Msgf("Need to apply message to %d nodes", majority)

	lastIdx := int64(len(n.Log.Entries) - 1)
	log.Trace().
		Int64("lastIndex", lastIdx).
		Int64("CommitIndex", n.CommitIndex).
		Msgf("Checking for update to commit index")
	for lastIdx > n.CommitIndex {
		count := 1
		for k := range n.otherNodes {
			if n.otherNodes[k].MatchIndex >= lastIdx {
				count++
			}
		}
		log.Trace().Msgf("Applied to %d nodes", count)
		if count >= majority {
			log.Info().
				Int64("prevCommitIndex", n.CommitIndex).
				Int64("newCommitIndex", lastIdx).
				Msgf("Updated commit index")
			n.CommitIndex = lastIdx
			break
		}
		lastIdx--
	}
	// if any records were committed, apply them to the database
	log.Trace().
		Int64("lastApplied", n.lastApplied).
		Msg("Applying records to database")
	for n.lastApplied < n.CommitIndex {
		n.lastApplied++
		action := n.Log.Entries[n.lastApplied].Action
		key := n.Log.Entries[n.lastApplied].Key
		if action == raft.LogRecord_SET {
			value := n.Log.Entries[n.lastApplied].Value
			log.Trace().
				Str("key", key).
				Str("value", value).
				Msg("Db set")
			n.Store.Set(key, value)
		} else if action == raft.LogRecord_DEL {
			log.Trace().
				Str("key", key).
				Msg("Db del")
			n.Store.Delete(key)
		}
	}
}

// requestAppend sends append to one other node with new record(s) and updates
// match index for that node if successful
func (n *Node) requestAppend(host string, term int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*12)
	defer cancel()

	prevLogIndex := n.otherNodes[host].MatchIndex
	// make a slice of all entries the other node has not seen (right after
	// election, this will be all records--would it be better to query for
	// number of entries in other node's log and start there? or is it better
	// to deal with this via reasonable log-compaction limits? (need to figure
	// out the relationship between log size and message size and make a
	// reasonable speculation about desired max message size)
	idx := int64(len(n.Log.Entries))
	newEntries := n.Log.Entries[prevLogIndex+1 : idx]
	var prevLogTerm int64
	if prevLogIndex >= 0 {
		prevLogTerm = n.Log.Entries[prevLogIndex].Term
	} else {
		prevLogTerm = 0
	}

	req := &raft.AppendRequest{
		Term:         term,
		Leader:       n.RaftNode,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      newEntries,
		LeaderCommit: n.CommitIndex}

	if n.State != Leader {
		// escape hatch in case this node stepped down in between the call to
		// `SendAppend` and this point
		log.Trace().Msg("requestAppend not leader, returning")
		return ErrNotLeaderSend
	}
	if term != n.Term {
		log.Trace().
			Int64("req term", term).
			Int64("node term", n.Term).
			Str("state", string(n.State)).
			Msg("past escape hatch")
		return ErrExpiredTerm
	}
	reply, err := n.otherNodes[host].Client.AppendLogs(ctx, req)
	if err == nil {
		if reply.Success {
			n.otherNodes[host].MatchIndex = idx - 1
			n.otherNodes[host].NextIndex = idx
			n.otherNodes[host].Available = true
			return nil
		} else {
			if prevLogIndex > 0 {
				n.otherNodes[host].MatchIndex--
				return n.requestAppend(host, term)
			}
			n.otherNodes[host].Available = false
			return ErrAppendRangeMet
			// todo: would it be viable for AppendReply to include the other
			// node's log index, so this could fast-forward to the correct
			// index, rather than recursing possibly down the whole list?
			// This implementation will blow the stack fast with any kind of
			// realistic history when you add a fresh node
		}
	}
	n.otherNodes[host].Available = false
	return err
}

// SendAppend sends out append-logs requests to each other node in the cluster,
// and updates database state on majority success
func (n *Node) SendAppend(retriesRemaining int, term int64) error {
	log.Trace().Msgf("SendAppend(r%d)", retriesRemaining)
	if n.State != Leader {
		log.Trace().Msg("SendAppend but not leader, returning")
		return ErrNotLeaderSend
	}

	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1

	log.Trace().Msgf("Number needed for append: %d", majority)

	var m sync.Mutex
	numAppended := 1
	// Send append out to all other nodes with new record(s)
	var wg sync.WaitGroup
	for k := range n.otherNodes {
		// append new entries
		// update indices
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			err := n.requestAppend(k, term)
			if err != nil {
				log.Debug().Err(err).Msgf(
					"Error requesting append from %s for term %d", k, term)
			} else {
				m.Lock()
				numAppended++
				m.Unlock()
			}
		}(k)
	}
	wg.Wait()

	log.Trace().Msgf("Appended to %d nodes", numAppended)
	if numAppended >= majority {
		log.Trace().Msg("majority")
		// update commit index on this node and apply newly committed records
		// to the database (next automatic append will commit on other nodes)
		n.commitRecords()
	} else {
		log.Trace().Msg("minority")
		// did not get a majority
		if retriesRemaining > 0 {
			return n.SendAppend(retriesRemaining-1, term)
		}
		return ErrAppendFailed
	}
	return nil
}

// NewNodeConfig creates a config for a Node
func NewNodeConfig(dataDir string, addr, clientAddr string, nodeIds []string) NodeConfig {
	return NodeConfig{
		Id:         addr,
		ClientAddr: clientAddr,
		DataDir:    dataDir,
		TermFile:   filepath.Join(dataDir, "term"),
		LogFile:    filepath.Join(dataDir, "raftlog"),
		NodeIds:    nodeIds}
}

// checkForeignNode verifies that a node is a known member of the cluster (this
// is the expected checker for a Node, but is extracted so it can be mocked)
func checkForeignNode(addr string, known map[string]*ForeignNode) bool {
	_, ok := known[addr]
	return ok
}

// NewNode initializes a Node with a randomized election timeout
func NewNode(config NodeConfig, store *db.Database) (*Node, error) {
	// Load persistent Node state
	termRecord := ReadTerm(config.TermFile)
	logStore := ReadLogs(config.LogFile)

	// channels used by Node to communicate with StateManager
	resetChannel := make(chan bool)
	// haltChannel := make(chan bool)

	votedForId := "<undetermined>"
	if termRecord.VotedFor != nil {
		votedForId = termRecord.VotedFor.Id
	}

	log.Info().
		Int64("Term", termRecord.Term).
		Str("Vote", votedForId).
		Int("nLogs", len(logStore.Entries)).
		Msg("On load")

	n := Node{
		RaftNode: &raft.Node{
			Id:         config.Id,
			ClientAddr: config.ClientAddr,
		},
		State:            Follower,
		Term:             termRecord.Term,
		votedFor:         termRecord.VotedFor,
		Reset:            resetChannel,
		otherNodes:       make(map[string]*ForeignNode),
		CheckForeignNode: checkForeignNode,
		AllowVote:        true,
		CommitIndex:      -1,
		lastApplied:      -1,
		Log:              logStore,
		config:           config,
		Store:            store}

	for _, addr := range config.NodeIds {
		n.AddForeignNode(addr)
	}
	return &n, nil
}

// AddForeignNode updates the list of known other members of the raft cluster
func (n *Node) AddForeignNode(addr string) {
	log.Trace().Msgf("AddForeignNode: %s", addr)
	n.otherNodes[addr], _ = NewForeignNode(addr)
	log.Info().Msgf("Added %s to known nodes", addr)
}

// availability returns the number of nodes believed to be currently available
// and the number of total nodes in the current cluster configuration
func (n *Node) availability() (int, int) {
	// initialize to 1 to account for the self
	available := 1
	total := 1
	for _, foreignNode := range n.otherNodes {
		total++
		if foreignNode.Available {
			available++
		}
	}
	return available, total
}

// candidateLogUpToDate checks if a candidate's log index is at least as high as
// the node's commit index (e.g.: candidate has all known committed entries), and
// that the
func (n *Node) candidateLogUpToDate(cLogIndex int64, cLogTerm int64) bool {
	indexGreater := cLogIndex > n.CommitIndex
	indexEqual := cLogIndex == n.CommitIndex
	bothEmpty := cLogIndex == -1 && n.CommitIndex == -1
	indexPresent := cLogIndex < int64(len(n.Log.Entries))

	upToDate := indexGreater ||
		bothEmpty ||
		(indexEqual && cLogTerm == n.Log.Entries[cLogIndex].Term)

	if !upToDate {
		failLog := log.Debug().
			Int64("CLogIdx", cLogIndex).
			Int64("CommitIdx", n.CommitIndex).
			Int64("CLogTerm", cLogTerm)
		if indexPresent {
			failLog.Int64("LogTerm", n.Log.Entries[cLogIndex].Term)
		}
		failLog.Msg("candidate log not up to date")
	}

	return upToDate
}

// HandleVote responds to vote requests from candidate nodes
func (n *Node) HandleVote(req *raft.VoteRequest) *raft.VoteReply {
	log.Info().Msgf("%s proposed term: %d", req.Candidate.Id, req.Term)
	var vote bool
	var msg string
	if req.Term < n.Term {
		vote = false
		msg = "Past term vote received"
	} else if req.Term == n.Term {
		vote = false
		msg = "Current term vote received"
		// If this node is the leader, and a vote request is received for the
		// current term, the current term should be increased because the other
		// node will have voted for itself and therefore not accept appends from
		// this leader (this happens when a previous leader restarts and then
		// comes back online and restarts an election for the current term, having
		// not received an append request from this leader in its initial election
		// window--shouldn't happen often, but if it does it would otherwise result
		// in an otherwise unnecessary election)
		if n.State == Leader {
			msg = msg + ", incrementing term"
			n.SetTerm(n.Term+1, n.RaftNode)
		}
	} else if !n.CheckForeignNode(req.Candidate.Id, n.otherNodes) {
		vote = false
		msg = "Unknown foreign node: " + req.Candidate.Id
	} else if !n.candidateLogUpToDate(req.LastLogIndex, req.LastLogTerm) {
		vote = false
		msg = "Candidate log not up to date"
	} else if !n.AllowVote {
		vote = false
		msg = "Leader still in grace period"
	} else {
		msg = "Voting yay"
		vote = true
		n.resetElectionTimer()
		n.SetTerm(req.Term, req.Candidate)
	}
	log.Info().
		Int64("Term", n.Term).
		Bool("Granted", vote).
		Msg(msg)
	return &raft.VoteReply{
		Term:        n.Term,
		VoteGranted: vote,
		Node:        n.RaftNode}
}

// validateAppend performs all checks for valid append request
func (n *Node) validateAppend(term int64, leaderId string) bool {
	var success bool
	success = true
	// reply false if req term < current term
	if term < n.Term {
		success = false
	} else if term == n.Term && leaderId != n.votedFor.Id {
		log.Error().
			Int64("term", n.Term).
			Str("got", leaderId).
			Str("expected", n.votedFor.Id).
			Msgf("Append request leader mismatch")
		success = false
	}
	if success {
		n.resetElectionTimer()
	}
	return success
}

// If an existing entry conflicts with a new one (same idx diff term),
// reconcileLogs deletes the existing entry and any that follow
func reconcileLogs(
	logStore *raft.LogStore, body *raft.AppendRequest) *raft.LogStore {
	// note: don't memoize length of Entries, it changes multiple times
	// during this method--safer to recalculate, and memoizing would
	// only save a maximum of one pass so it's not worth it
	var mismatchIdx int64
	mismatchIdx = -1
	if body.PrevLogIndex < int64(len(logStore.Entries)-1) {
		overlappingEntries := logStore.Entries[body.PrevLogIndex+1:]
		for i, rec := range overlappingEntries {
			if i >= len(body.Entries) {
				mismatchIdx = body.PrevLogIndex + int64(i)
				break
			}
			if rec.Term != body.Entries[i].Term {
				mismatchIdx = body.PrevLogIndex + 1 + int64(i)
				break
			}
		}
	}
	if mismatchIdx >= 0 {
		log.Debug().Msgf("Mismatch index: %d - rewinding log", mismatchIdx)
		logStore.Entries = logStore.Entries[:mismatchIdx]
	}
	// append any entries not already in log
	offset := int64(len(logStore.Entries)-1) - body.PrevLogIndex
	newLogs := body.Entries[offset:]
	log.Info().Msgf("Appending %d entries from %s", len(newLogs), body.Leader.Id)
	return &raft.LogStore{Entries: append(logStore.Entries, newLogs...)}
}

// applyCommittedLogs updates the database with actions that have not yet been
// applied, up to the new commit index
func (n *Node) applyCommittedLogs(commitIdx int64) {
	log.Debug().
		Int64("current", n.CommitIndex).
		Int64("leader", commitIdx).
		Msg("apply commits")
	if commitIdx > n.CommitIndex {
		// ensure we don't run over the end of the log
		lastIndex := int64(len(n.Log.Entries))
		if commitIdx > lastIndex {
			commitIdx = lastIndex
		}

		// apply all entries up to new commit index to store
		for n.CommitIndex < commitIdx {
			n.CommitIndex++
			action := n.Log.Entries[n.CommitIndex].Action
			key := n.Log.Entries[n.CommitIndex].Key
			if action == raft.LogRecord_SET {
				value := n.Log.Entries[n.CommitIndex].Value
				n.Store.Set(key, value)
			} else if action == raft.LogRecord_DEL {
				n.Store.Delete(key)
			}
		}

		log.Info().
			Int64("commit", n.CommitIndex).
			Msg("Commit updated")
	}
}

// checkPrevious returns true if Node.logs contains an entry at the specified
// index with the specified term, otherwise false
func (n *Node) checkPrevious(prevIndex int64, prevTerm int64) bool {
	if prevIndex < 0 {
		return true
	}
	inRange := prevIndex < int64(len(n.Log.Entries))
	matches := n.Log.Entries[prevIndex].Term == prevTerm
	return inRange && matches
}

// HandleAppend responds to append-log messages from leader nodes
func (n *Node) HandleAppend(req *raft.AppendRequest) *raft.AppendReply {
	var success bool

	valid := n.validateAppend(req.Term, req.Leader.Id)
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
			n.Log = reconcileLogs(n.Log, req)
			n.setLog(n.Log.Entries)
		}
		n.applyCommittedLogs(req.LeaderCommit)
		success = true
	}
	if valid {
		// update term if necessary
		if req.Term > n.Term {
			log.Info().
				Int64("newTerm", req.Term).
				Str("votedFor", req.Leader.Id).
				Msg("Got more recent append, updating term record")
			n.SetTerm(req.Term, req.Leader)
		}
		// reset the election timer on append from a valid leader (even if
		// not matched)--this duplicates the reset in `validateAppend`, in order to
		// ensure that the time it takes to do all of the operations in this
		// handler effectively happened "instantaneously" from the perspective of
		// the election timeout
		n.resetElectionTimer()
	}
	// finally
	return &raft.AppendReply{Term: n.Term, Success: success}
}
