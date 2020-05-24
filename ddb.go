// Practice implementation of the Raft distributed-consensus algorithm
// See: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/btmorr/go-raft/internal/fileutils"
)

// A LogRecord is a Raft log object, shipped to other servers to propagate writes
type LogRecord struct {
	Term  int    `json:"term"`
	Value string `json:"value"`
}

// A role is one of Leader, Candidate, or Follower
type role int

const (
	follower role = iota
	candidate
	leader
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
	Value           string
	NodeId          string
	electionTimeout time.Duration
	electionTimer   *time.Timer
	appendTimeout   time.Duration
	appendTicker    *time.Ticker
	State           role
	haltAppend      chan bool
	Term            int
	votedFor        string
	otherNodes      map[string]bool
	nextIndex       map[string]int
	matchIndex      map[string]int
	commitIndex     int
	lastApplied     int
	log             []LogRecord
	config          NodeConfig
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

// Non-volatile state functions
// `Term`, `votedFor`, and `log` must persist through application restart, so
// any request that changes these values must be written to disk before
// responding to the request.

// SetTerm records term and vote in non-volatile state
func (n *Node) SetTerm(newTerm int, votedFor string) error {
	n.Term = newTerm
	n.votedFor = votedFor
	vote := fmt.Sprintf("%d, %s\n", newTerm, votedFor)
	return fileutils.Write(n.config.TermFile, vote)
}

// SetLog records new log contents in non-volatile state
func (n *Node) SetLog(newLog []LogRecord) error {
	n.log = newLog
	logString := ""
	for _, l := range newLog {
		logString = logString + fmt.Sprintf("%d, %s\n", l.Term, l.Value)
	}
	return fileutils.Write(n.config.LogFile, logString)
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
				log.Print(".")
				continue
			}
		}
	}()
}

// requestVote sends a request for vote to a single other node (see `doElection`)
func (n Node) requestVote(host string, term int) (bool, error) {
	uri := "http://" + host + "/vote"
	log.Println("Requesting vote from ", host)

	body := VoteBody{
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

	var vote VoteResponse
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
	n.SetState(candidate)
	log.Println("Becoming candidate")
	n.Term = n.Term + 1
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
		n.SetState(leader)
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
		n.SetState(follower)
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
func NewNode(config NodeConfig) (*Node, error) {
	lowerBound := 150
	upperBound := 300
	ms := (rand.Int() % lowerBound) + (upperBound - lowerBound)
	electionTimeout := time.Duration(ms) * time.Millisecond

	appendTimeout := time.Duration(10) * time.Millisecond

	var term int
	var votedFor string
	_, err := os.Stat(config.TermFile)
	if err != nil {
		term = 0
		votedFor = ""
	} else {
		raw, _ := fileutils.Read(config.TermFile)
		dat := strings.Split(raw, " ")
		term, _ = strconv.Atoi(dat[0])
		votedFor = dat[1]
	}

	var logs []LogRecord
	_, err2 := os.Stat(config.LogFile)
	if err2 != nil {
		logs = make([]LogRecord, 0, 0)
	} else {
		raw, _ := fileutils.Read(config.LogFile)
		rows := strings.Split(raw, "\n")
		for _, row := range rows {
			dat := strings.Split(row, " ")
			if len(dat) == 2 {
				logTerm, _ := strconv.Atoi(dat[0])
				logs = append(logs, LogRecord{Term: logTerm, Value: dat[1]})
			}
		}
	}

	n := Node{
		Value:           "",
		NodeId:          config.Id,
		electionTimeout: electionTimeout,
		electionTimer:   time.NewTimer(electionTimeout),
		appendTimeout:   appendTimeout,
		appendTicker:    time.NewTicker(appendTimeout),
		State:           follower,
		haltAppend:      make(chan bool),
		Term:            term,
		votedFor:        votedFor,
		otherNodes:      make(map[string]bool),
		nextIndex:       make(map[string]int),
		matchIndex:      make(map[string]int),
		commitIndex:     0,
		lastApplied:     0,
		log:             logs,
		config:          config}

	go func() {
		log.Println("First election timer")
		<-n.electionTimer.C
		n.doElection()
	}()

	return &n, nil
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

	log.Println("New value: ", data.Value)
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
	log.Println("Restarting election timer")

	if !n.electionTimer.Stop() {
		select {
		case <-n.electionTimer.C:
		default:
		}
	}
	n.electionTimer.Reset(n.electionTimeout)

	go func() {
		<-n.electionTimer.C
		n.doElection()
	}()
}

// addNodeToKnown updates the list of known other members of the raft cluster
func (n *Node) addNodeToKnown(addr string) {
	n.otherNodes[addr] = true
	log.Println("Added ", addr, " to known nodes")
	log.Println("Nodes: ", n.otherNodes)
}

// Handler for vote requests from candidate nodes
func (n *Node) handleVote(c *gin.Context) {
	var body VoteBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n.addNodeToKnown(body.CandidateId)

	log.Println(body.CandidateId, " proposed term: ", body.Term)
	var status int
	var vote gin.H
	if body.Term <= n.Term {
		// Use 409 Conflict to represent invalid term
		n.Term = n.Term + 1
		log.Println("Expired term vote received. New term: ", n.Term)
		status = http.StatusConflict
		vote = gin.H{"term": n.Term, "voteGranted": false}
	} else {
		log.Println("Voting for ", body.CandidateId, " for term ", body.Term)
		n.Term = body.Term
		n.votedFor = body.CandidateId
		if n.State == leader {
			n.haltAppend <- true
		}
		n.SetState(follower)
		n.resetElectionTimer()

		// todo: check candidate's log details
		status = http.StatusOK
		vote = gin.H{"term": n.Term, "voteGranted": true}
		log.Println("Returning vote: ", vote)
	}
	log.Println("Returning vote: [", status, " ]", vote)

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
		if n.State == candidate {
			n.SetState(follower)
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

func buildRouter(node *Node) *gin.Engine {
	router := gin.Default()

	router.GET("/health", handleHealth)
	router.POST("/vote", node.handleVote)
	router.POST("/append", node.handleAppend)
	router.GET("/", node.handleDataRead)
	router.POST("/", node.handleDataWrite)

	return router
}

// GetOutboundIP returns ip of preferred interface this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

// EnsureDirectory creates the directory if it does not exist (fail if path
// exists and is not a directory)
func EnsureDirectory(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		var fileMode os.FileMode
		fileMode = os.ModeDir | 0775
		mdErr := os.MkdirAll(path, fileMode)
		return mdErr
	}
	if err != nil {
		return err
	}
	file, _ := os.Stat(path)
	if !file.IsDir() {
		return errors.New(path + " is not a directory")
	}
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// todo: determine port programatically
	port := "8080"
	addr := fmt.Sprintf("%s:%s", GetOutboundIP(), port)
	log.Println("Address: " + addr)

	hash := fnv.New32()
	hash.Write([]byte(addr))
	hashString := fmt.Sprintf("%x", hash.Sum(nil))

	// todo: make this configurable
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".go-raft", hashString)
	log.Println("Data dir: ", dataDir)
	err := EnsureDirectory(dataDir)
	if err != nil {
		panic(err)
	}

	config := NewNodeConfig(dataDir, addr)
	log.Println("Persistent data in " + config.DataDir)

	node, err := NewNode(config)
	if err != nil {
		log.Println("Failed to initialize node")
		panic(err)
	}
	log.Println("Election timeout: ", node.electionTimeout.String())

	router := buildRouter(node)
	router.Run()
}
