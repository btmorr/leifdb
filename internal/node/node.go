package node

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/mgmt"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
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
	// without successfully appending to a majorit of nodes
	ErrAppendFailed = errors.New("Failed to append logs to a majority of nodes")

	// ErrCommitFailed indicates that the leader's commit index after append
	// is less than the index of the record being added
	ErrCommitFailed = errors.New("Failed to commit record")

	// ErrAppendRangeMet indicates that reverse-iteration has reached the
	// beginning of the log and still not gotten a response--aborting
	ErrAppendRangeMet = errors.New("Append range reached, not trying again")
)

// A ForeignNode is another member of the cluster, with connections needed
// to manage gRPC interaction with that node
type ForeignNode struct {
	Connection *grpc.ClientConn
	Client     raft.RaftClient
	NextIndex  int64
	MatchIndex int64
}

// NewForeignNode constructs a ForeignNode from an address ("host:port")
func NewForeignNode(address string) (*ForeignNode, error) {
	conn, err := grpc.Dial(
		address,
		grpc.WithInsecure(),
		grpc.WithTimeout(time.Second))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to %s", address)
		return nil, err
	}

	client := raft.NewRaftClient(conn)

	return &ForeignNode{
		Connection: conn,
		Client:     client,
		NextIndex:  0,
		MatchIndex: -1}, err
}

// Close cleans up the gRPC connection with the foreign node
func (f *ForeignNode) Close() {
	f.Connection.Close()
}

// NodeConfig contains configurable properties for a node
type NodeConfig struct {
	Id       string
	DataDir  string
	TermFile string
	LogFile  string
	NodeIds  []string
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
	NodeId           string
	State            mgmt.Role
	Term             int64
	votedFor         string
	Reset            chan bool
	otherNodes       map[string]*ForeignNode
	CheckForeignNode ForeignNodeChecker
	commitIndex      int64
	lastApplied      int64
	Log              *raft.LogStore
	config           NodeConfig
	Store            *db.Database
}

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

// setTerm records term and vote in non-volatile state
func (n *Node) setTerm(newTerm int64, votedFor string) error {
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

func (n *Node) resetElectionTimer() {
	n.State = mgmt.Follower
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
	if n.State != mgmt.Leader {
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
	// verify that n.commitIndex >= idx
	if n.commitIndex < idx {
		log.Error().Err(ErrCommitFailed).
			Int64("recordIndex", idx).
			Int64("commitIndex", n.commitIndex).
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
	return n.applyRecord(record)
}

// requestVote sends a request for vote to a single other node (see DoElection)
func (n *Node) requestVote(host string) (*raft.VoteReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*4)
	defer cancel()

	voteRequest := &raft.VoteRequest{
		Term:         n.Term,
		CandidateId:  n.NodeId,
		LastLogIndex: 0,
		LastLogTerm:  0}

	vote, err := n.otherNodes[host].Client.RequestVote(ctx, voteRequest)
	if err != nil {
		log.Warn().Err(err).Msgf("Error requesting vote from %s", host)
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
	n.setTerm(n.Term+1, n.NodeId)

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
	for k := range n.otherNodes {
		// put this in a goroutine and use a sync.WaitGroup to collect? if needed
		// for performance, figure out how to collect the term responses in a
		// thread-safe way
		vote, err := n.requestVote(k)
		log.Trace().Msg("got a vote")
		if err != nil {
			continue
		}
		if vote.VoteGranted {
			log.Trace().Msg("it's a 'yay'")
			numVotes = numVotes + 1
		} else {
			if vote.Term > maxTermSeen {
				maxTermSeen = vote.Term
				maxTermSeenSource = k
			}
		}
	}
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
				Str("other node", maxTermSeenSource).
				Msg("Updating term to max seen")
			n.setTerm(maxTermSeen, maxTermSeenSource)
		}
	} else {
		voteLog.
			Bool("success", true).
			Int64("term", n.Term).
			Msg("Election succeeded")
		n.State = mgmt.Leader
		success = true

		for k := range n.otherNodes {
			n.otherNodes[k].MatchIndex = -1
			n.otherNodes[k].NextIndex = int64(len(n.Log.Entries))
		}
	}
	return success
}

// commitRecords iterates backward from last index of log entries, and finds
// latest index that has been appended to a majority of nodes, and updates
// the database and node commitIndex
func (n *Node) commitRecords() {
	log.Trace().Msg("commitRecords")

	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1
	log.Debug().Msgf("Need to apply message to %d nodes", majority)

	// todo: find a more computationally efficient way to compute this
	lastIdx := int64(len(n.Log.Entries) - 1)
	log.Debug().
		Int64("lastIndex", lastIdx).
		Int64("commitIndex", n.commitIndex).
		Msgf("Checking for update to commit index")
	for lastIdx > n.commitIndex {
		count := 1
		for k := range n.otherNodes {
			if n.otherNodes[k].MatchIndex >= lastIdx {
				count++
			}
		}
		log.Debug().Msgf("Applied to %d nodes", count)
		if count >= majority {
			log.Info().
				Int64("prevCommitIndex", n.commitIndex).
				Int64("newCommitIndex", lastIdx).
				Msgf("Updated commit index")
			n.commitIndex = lastIdx
			break
		}
		lastIdx--
	}
	// if any records were committed, apply them to the database
	log.Debug().
		Int64("lastApplied", n.lastApplied).
		Msg("Applying records to database")
	for n.lastApplied < n.commitIndex {
		n.lastApplied++
		action := n.Log.Entries[n.lastApplied].Action
		key := n.Log.Entries[n.lastApplied].Key
		if action == raft.LogRecord_SET {
			value := n.Log.Entries[n.lastApplied].Value
			log.Debug().
				Str("key", key).
				Str("value", value).
				Msg("Db set")
			n.Store.Set(key, value)
		} else if action == raft.LogRecord_DEL {
			log.Debug().
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
		LeaderId:     n.NodeId,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      newEntries,
		LeaderCommit: n.commitIndex}

	if n.State != mgmt.Leader {
		// escape hatch in case this node stepped down in between the call to
		// `SendAppend` and this point
		log.Debug().Msg("requestAppend not leader, returning")
		return ErrNotLeaderSend
	}
	if term != n.Term {
		log.Debug().
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
		} else {
			if prevLogIndex > 0 {
				n.otherNodes[host].MatchIndex--
				return n.requestAppend(host, term)
			} else {
				return ErrAppendRangeMet
			}
			// todo: would it be viable for AppendReply to include the other
			// node's log index, so this could fast-forward to the correct
			// index, rather than recursing possibly down the whole list?
			// This implementation will blow the stack fast with any kind of
			// realistic history when you add a fresh node
		}
	}
	return err
}

// SendAppend sends out append-logs requests to each other node in the cluster,
// and updates database state on majority success
func (n *Node) SendAppend(retriesRemaining int, term int64) error {
	log.Trace().Msgf("SendAppend(r%d)", retriesRemaining)
	if n.State != mgmt.Leader {
		log.Debug().Msg("SendAppend but not leader, returning")
		return ErrNotLeaderSend
	}

	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1

	log.Debug().Msgf("Number needed for append: %d", majority)

	numAppended := 1
	// Send append out to all other nodes with new record(s)
	var wg sync.WaitGroup
	for k := range n.otherNodes {
		// append new entries
		// update indicies
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := n.requestAppend(k, term)
			if err != nil {
				log.Debug().Err(err).Msgf(
					"Error requesting append from %s for term %d", k, term)
			} else {
				// is this threadsafe?
				numAppended++
			}
		}()
	}
	wg.Wait()

	log.Debug().Msgf("Appended to %d nodes", numAppended)
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
		} else {
			return ErrAppendFailed
		}
	}
	return nil
}

// NewNodeConfig creates a config for a Node
func NewNodeConfig(dataDir string, addr string, nodeIds []string) NodeConfig {
	// todo: check dataDir. if it is empty, initialize Node with default
	// values for non-volatile state. Otherwise, read values.
	return NodeConfig{
		Id:       addr,
		DataDir:  dataDir,
		TermFile: filepath.Join(dataDir, "term"),
		LogFile:  filepath.Join(dataDir, "raftlog"),
		NodeIds:  nodeIds}
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

	log.Info().
		Int64("Term", termRecord.Term).
		Str("Vote", termRecord.VotedFor).
		Int("nLogs", len(logStore.Entries)).
		Msg("On load")

	n := Node{
		NodeId:           config.Id,
		State:            mgmt.Follower,
		Term:             termRecord.Term,
		votedFor:         termRecord.VotedFor,
		Reset:            resetChannel,
		otherNodes:       make(map[string]*ForeignNode),
		CheckForeignNode: checkForeignNode,
		commitIndex:      -1,
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

// HandleVote responds to vote requests from candidate nodes
func (n *Node) HandleVote(req *raft.VoteRequest) *raft.VoteReply {
	log.Info().Msgf("%s proposed term: %d", req.CandidateId, req.Term)
	var vote bool
	var msg string
	// todo: check candidate's log details
	if req.Term <= n.Term {
		// Increment term, vote for same node as previous term (is this correct?)
		vote = false
		msg = "Expired term vote received"
		if n.State == mgmt.Leader {
			n.setTerm(n.Term+1, n.votedFor)
			msg = msg + ", incrementing term"
		}
	} else if !n.CheckForeignNode(req.CandidateId, n.otherNodes) {
		vote = false
		msg = "Unknown foreign node: " + req.CandidateId
	} else {
		msg = "Voting yay"
		vote = true
		n.resetElectionTimer()
		n.setTerm(req.Term, req.CandidateId)
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
	} else if term == n.Term && leaderId != n.votedFor {
		log.Error().
			Int64("term", n.Term).
			Str("got", leaderId).
			Str("expected", n.votedFor).
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
	log.Info().Msgf("Appending %d entries from %s", len(newLogs), body.LeaderId)
	return &raft.LogStore{Entries: append(logStore.Entries, newLogs...)}
}

// applyCommittedLogs updates the database with actions that have not yet been
// applied, up to the new commit index
func (n *Node) applyCommittedLogs(commitIdx int64) {
	log.Debug().
		Int64("current", n.commitIndex).
		Int64("leader", commitIdx).
		Msg("apply commits")
	if commitIdx > n.commitIndex {
		// ensure we don't run over the end of the log
		lastIndex := int64(len(n.Log.Entries))
		if commitIdx > lastIndex {
			commitIdx = lastIndex
		}

		// apply all entries up to new commit index to store
		for n.commitIndex < commitIdx {
			n.commitIndex++
			action := n.Log.Entries[n.commitIndex].Action
			key := n.Log.Entries[n.commitIndex].Key
			if action == raft.LogRecord_SET {
				value := n.Log.Entries[n.commitIndex].Value
				n.Store.Set(key, value)
			} else if action == raft.LogRecord_DEL {
				n.Store.Delete(key)
			}
		}

		log.Info().
			Int64("commit", n.commitIndex).
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
				Str("votedFor", req.LeaderId).
				Msg("Got more recent append, updating term record")
			n.setTerm(req.Term, req.LeaderId)
		}
		// reset the election timer on append from a valid leader (even if
		// not matched)--this duplicates the reset in `validateAppend`, in order to
		// ensure that the time it takes to do all of the operations in this
		// handler effectively happend "instantaneously" from the perspective of
		// the election timeout
		n.resetElectionTimer()
	}
	// finally
	return &raft.AppendReply{Term: n.Term, Success: success}
}
