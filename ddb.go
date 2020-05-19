// Practice implementation of the Raft distributed-consensus algorithm
// See: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// A LogRecord is a Raft log object, shipped to other servers to propagate writes
type LogRecord struct {
	Term  int    `json:"term"`
	Value string `json:"value"`
}

// A Role is one of Leader, Candidate, or Follower
type Role int

const (
	FOLLOWER Role = iota
	CANDIDATE
	LEADER
)

// A Node is one member of a Raft cluster, with all state needed to operate the
// algorithm's state machine. At any one time, its role may be Leader, Candidate,
// or Follower, and have different responsibilities depending on its role
type Node struct {
	Value           string
	NodeId          string
	electionTimeout time.Duration
	electionTimer   *time.Timer
	appendTimeout   time.Duration
	appendTicker    *time.Ticker
	State           Role
	haltAppend      chan bool
	Term            int
	votedFor        string
	otherNodes      map[string]bool
	nextIndex       map[string]int
	matchIndex      map[string]int
	commitIndex     int
	lastApplied     int
	log             []LogRecord
}

// Data types for [un]marshalling JSON

// A WriteBody is a request body template for write route
type WriteBody struct {
	Value string `json:"value"`
}

// A WriteResponse is a response body template for the write route
type WriteResponse struct {
	Value string `json:"value"`
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

// Client methods for managing raft state

// When a Raft node's role is "leader", startAppendTicker periodically send out
// an append-logs request to each other node on a period shorter than any node's
// election timeout
func (n *Node) startAppendTicker() {
	go func() {
		for {
			select {
			case <-n.haltAppend:
				fmt.Println("No longer leader. Halting log append...")
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
func (n Node) requestVote(host string, term int) (bool, error) {
	uri := "http://" + host + "/vote"
	fmt.Println("Requesting vote from ", host)

	body := VoteBody{
		Term:         term,
		CandidateId:  n.NodeId,
		LastLogIndex: 0,
		LastLogTerm:  0}

	b, _ := json.Marshal(body)
	br := bytes.NewReader(b)

	resp, _ := http.Post(uri, "application/json", br)

	raw, err1 := ioutil.ReadAll(resp.Body)
	if err1 != nil {
		return false, err1
	}

	var vote VoteResponse
	err2 := json.Unmarshal(raw, &vote)

	return vote.VoteGranted, err2
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
	fmt.Println("Starting Election")
	n.State = CANDIDATE
	fmt.Println("Becoming candidate")
	n.Term = n.Term + 1
	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1

	fmt.Println("\tNew Term: ", n.Term)
	fmt.Println("\tN other nodes: ", len(n.otherNodes))
	fmt.Println("\tVotes needed: ", majority)

	n.resetElectionTimer()
	numVotes := 1
	for k := range n.otherNodes {
		_, err := n.requestVote(k, n.Term)
		fmt.Println("got a vote")
		if err == nil {
			numVotes = numVotes + 1
		}
	}
	if numVotes >= majority {
		fmt.Println(
			"Election succeeded [",
			numVotes, " out of ", majority,
			" needed]")
		n.State = LEADER
		fmt.Println("Becoming leader")

		n.electionTimer.Stop()
		fmt.Println("Stopping election timer, starting append ticker")
		n.startAppendTicker()
	} else {
		fmt.Println(
			"Election failed [",
			numVotes, " out of ", majority,
			" needed]")
		n.State = FOLLOWER
		fmt.Println("Becoming follower")
	}
}

// NewNode initializes a Node with a randomized election timeout between
// 150-300ms, and starts the election timer
func NewNode(port string) *Node {
	lowerBound := 150
	upperBound := 300
	ms := (rand.Int() % lowerBound) + (upperBound - lowerBound)
	electionTimeout := time.Duration(ms) * time.Millisecond

	appendTimeout := time.Duration(10) * time.Millisecond

	n := Node{
		Value:           "",
		NodeId:          port,
		electionTimeout: electionTimeout,
		electionTimer:   time.NewTimer(electionTimeout),
		appendTimeout:   appendTimeout,
		appendTicker:    time.NewTicker(appendTimeout),
		State:           FOLLOWER,
		haltAppend:      make(chan bool),
		Term:            0,
		votedFor:        "",
		otherNodes:      make(map[string]bool),
		nextIndex:       make(map[string]int),
		matchIndex:      make(map[string]int),
		commitIndex:     0,
		lastApplied:     0,
		log:             make([]LogRecord, 0, 0)}

	go func() {
		fmt.Println("First election timer")
		<-n.electionTimer.C
		n.doElection()
	}()

	return &n
}

// Server methods for handling data read/write

// Handler for POSTs to the data endpoint (client write)
func (n *Node) handleDataWrite(c *gin.Context) {
	// todo: revise to log-append / commit protocol once elections work
	var data WriteBody
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("New value: ", data.Value)
	n.Value = data.Value

	c.JSON(http.StatusOK, gin.H{"value": n.Value})
}

// Handler for GETs to the data endpoint (client read)
func (n *Node) handleDataRead(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"value": n.Value})
}

// When a Raft node is a follower or candidate and receives a message from a
// valid leader, it should reset its election countdown timer
func (n *Node) resetElectionTimer() {
	fmt.Println("Restarting election timer")
	n.electionTimer.Reset(n.electionTimeout)
}

// addNodeToKnown updates the list of known other members of the raft cluster
func (n *Node) addNodeToKnown(addr string) {
	n.otherNodes[addr] = true
	fmt.Println("Added ", addr, " to known nodes")
	fmt.Println("Nodes: ", n.otherNodes)
}

// Handler for vote requests from candidate nodes
func (n *Node) handleVote(c *gin.Context) {
	var body VoteBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n.addNodeToKnown(body.CandidateId)

	fmt.Println(body.CandidateId, " proposed term: ", body.Term)
	var status int
	var vote gin.H
	if body.Term <= n.Term {
		// Use 409 Conflict to represent invalid term
		n.Term = n.Term + 1
		fmt.Println("Expired term vote received. New term: ", n.Term)
		status = http.StatusConflict
		vote = gin.H{"term": n.Term, "voteGranted": false}
	} else {
		fmt.Println("Voting for ", body.CandidateId, " for term ", body.Term)
		n.Term = body.Term
		n.votedFor = body.CandidateId
		if n.State == LEADER {
			n.haltAppend<-true
		}
		n.State = FOLLOWER
		n.resetElectionTimer()

		// todo: check candidate's log details
		status = http.StatusOK
		vote = gin.H{"term": n.Term, "voteGranted": true}
	}	
	c.JSON(status, vote)
}

// Handler for append-log messages from leader nodes
func (n *Node) handleAppend(c *gin.Context) {
	var body AppendBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	success := true
	// reply false if req term < current term
	if body.Term < n.Term {
		success = false
	}
	// if none of the failure conditions fired...
	if success {
		// reply false if log does not contain entry at req idx matching req term
		if body.PrevLogIndex > len(n.log) {
			success = false
		} else {
			lastLog := n.log[body.PrevLogIndex]
			if lastLog.Term != body.PrevLogTerm {
				success = false
			}
		}
		// if an existing entry conflicts with a new one (same idx diff term),
		// delete the existing entry and any that follow
		mismatchIdx := -1
		if body.PrevLogIndex < len(n.log) {
			overlappingEntries := n.log[body.PrevLogIndex:]
			for i, rec := range overlappingEntries {
				if rec.Term != body.Entries[i].Term {
					mismatchIdx = body.PrevLogIndex + i
					break
				}
			}
		}
		if mismatchIdx >= 0 {
			n.log = n.log[:mismatchIdx]
		}
		// append any entries not already in log
		offset := len(n.log) - body.PrevLogIndex
		n.log = append(n.log, body.Entries[offset:]...)
		// update commit idx
		if body.LeaderCommit > n.commitIndex {
			if body.LeaderCommit < len(n.log) {
				n.commitIndex = body.LeaderCommit
			} else {
				n.commitIndex = len(n.log)
			}
		}

		// if a valid append is received during an election, cancel election
		if n.State == CANDIDATE {
			n.State = FOLLOWER
		}

		// only reset the election timer on append from a valid leader
		n.resetElectionTimer()
	}
	// finally
	c.JSON(http.StatusOK, gin.H{"term": n.Term, "success": success})
}

// Handler for the health endpoint--not required for Raft, but useful for infrastructure
// monitoring, such as determining when a node is available in blue-green deploy
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "Ok"})
}

func main() {
	rand.Seed(time.Now().UnixNano())
	port := "8080"

	node := NewNode(port)
	fmt.Println("Election timeout: ", node.electionTimeout.String())

	router := gin.Default()

	router.GET("/health", handleHealth)
	router.POST("/vote", node.handleVote)
	router.POST("/append", node.handleAppend)
	router.GET("/", node.handleDataRead)
	router.POST("/", node.handleDataWrite)

	router.Run()
}
