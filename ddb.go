package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type LogRecord struct {
	Term  int    `json:"term"`
	Value string `json:"value"`
}

type Role int

const (
	FOLLOWER Role = iota
	CANDIDATE
	LEADER
)

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

// Common error type for all endpoints
type ErrorResponse struct {
	Error string `json:"error"`
}

type WriteBody struct {
	Value string `json:"value"`
}

type WriteResponse struct {
	Value string `json:"value"`
}

type VoteBody struct {
	Term         int    `json:"term"`
	CandidateId  string `json:"candidateId"`
	LastLogIndex int    `json:"lastLogIndex"`
	LastLogTerm  int    `json:"lastLogTerm"`
}

type VoteResponse struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"voteGranted"`
}

type AppendBody struct {
	Term         int         `json:"term"`
	LeaderId     string      `json:"leaderId"`
	PrevLogIndex int         `json:"prevLogIndex"`
	PrevLogTerm  int         `json:"prevLogTerm"`
	Entries      []LogRecord `json:"entries"`
	LeaderCommit int         `json:"leaderCommit"`
}

type AppendResponse struct {
	Term    int  `json:"term"`
	Success bool `json:"success"`
}

// /health is a GET endpoint, so no Body type
type HealthResponse struct {
	Status string `json:"status"`
}

// Client methods for managing raft state

func (n *Node) startAppendTicker() {
	go func() {
		for {
			select {
			case <-n.haltAppend:
				fmt.Println("No longer leader. Halting log append...")
				return
			case <-n.appendTicker.C:
				// placeholder for generating append requests
				// fmt.Print(".")
				continue
			}
		}
	}()
}

func (n Node) requestVote(host string, term int) (bool, error) {
	uri := "http://" + host + "/vote"
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

func (n *Node) doElection() {
	fmt.Println("Starting Election")
	n.State = CANDIDATE
	n.Term = n.Term + 1
	numNodes := len(n.otherNodes)
	majority := (numNodes / 2) + 1

	fmt.Println("\tNew Term: ", n.Term)
	fmt.Println("\tN other nodes: ", len(n.otherNodes))
	fmt.Println("\tVotes needed: ", majority)

	n.resetElectionTimer()
	numVotes := 1
	for k, _ := range n.otherNodes {
		vote, _ := n.requestVote(k, n.Term)
		if vote {
			numVotes = numVotes + 1
		}
	}
	if numVotes >= majority {
		fmt.Println(
			"Election succeeded [",
			numVotes, " out of ", majority,
			"]")
		n.State = LEADER

		n.electionTimer.Stop()
		n.startAppendTicker()
	} else {
		fmt.Println(
			"Election failed [",
			numVotes, " out of ", majority,
			"]")
		n.State = FOLLOWER
	}
}

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

func (n *Node) handleDataWrite(w http.ResponseWriter, r *http.Request) {
	// revise to log-append / commit protocol once elections work
	raw, err1 := ioutil.ReadAll(r.Body)
	if err1 != nil {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Body required"}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
		return
	}

	var data WriteBody
	err2 := json.Unmarshal(raw, &data)
	if err2 != nil {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Invalid JSON body"}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
		return
	}

	fmt.Println("New value: ", data.Value)
	n.Value = data.Value
	res := WriteResponse{Value: n.Value}
	b, _ := json.Marshal(res)
	fmt.Fprintln(w, string(b))
}

func (n Node) handleDataRead(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", n.Value)
}

func (n *Node) handleData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[data] ", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		n.handleDataWrite(w, r)
	} else if r.Method == http.MethodGet {
		n.handleDataRead(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)

		fmt.Fprintln(w, "Unsupported verb", r.Method)
	}
}

// Server methods for handling Raft state

func (n *Node) resetElectionTimer() {
	fmt.Println("Restarting election timer")
	n.electionTimer.Stop()
	n.electionTimer = time.NewTimer(n.electionTimeout)
	go func() {
		<-n.electionTimer.C
		n.doElection()
	}()
}

func (n *Node) handleVoteRequest(w http.ResponseWriter, r *http.Request) {
	addr := r.RemoteAddr

	n.otherNodes[addr] = true
	fmt.Println("Added ", addr, " to known nodes")
	fmt.Println("Nodes: ", n.otherNodes)

	raw, err1 := ioutil.ReadAll(r.Body)
	if err1 != nil {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Body required"}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
		return
	}

	var body VoteBody
	err2 := json.Unmarshal(raw, &body)
	if err2 != nil {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Invalid JSON body"}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
		return
	}
	fmt.Println("\tProposed term: ", body.Term)

	if body.Term <= n.Term {
		// Use 409 Conflict to represent invalid term
		n.Term = n.Term + 1
		fmt.Println("Expired term vote received. New term: ", n.Term)
		w.WriteHeader(http.StatusConflict)
		vote := VoteResponse{Term: n.Term}
		b, _ := json.Marshal(vote)
		fmt.Fprintln(w, string(b))
	} else {
		fmt.Println("Voting for ", addr, " for term ", body.Term)
		n.Term = body.Term
		vote := VoteResponse{Term: n.Term}
		b, _ := json.Marshal(vote)
		fmt.Fprintln(w, string(b))
	}
}

func (n *Node) handleAppend(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[logs] ", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")

	raw, err1 := ioutil.ReadAll(r.Body)
	if err1 != nil {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Body required"}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
		return
	}

	var body AppendBody
	err2 := json.Unmarshal(raw, &body)
	if err2 != nil {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Invalid JSON body"}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
		return
	}

	var success bool
	success = true
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

		if n.State == CANDIDATE {
			n.State = FOLLOWER
			n.resetElectionTimer()
		}
	}
	// finally
	res := AppendResponse{Term: n.Term, Success: success}
	b, _ := json.Marshal(res)
	fmt.Fprintln(w, string(b))
}

func (n *Node) handleVote(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[vote] ", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		// This is a request for Vote
		n.handleVoteRequest(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Unsupported HTTP verb " + r.Method}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
	}
}

// Other stuff

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[health] ", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	res := HealthResponse{Status: "Ok"}
	b, _ := json.Marshal(res)
	fmt.Fprintln(w, string(b))
}

func main() {
	rand.Seed(time.Now().UnixNano())
	port := "8080"

	node := NewNode(port)
	fmt.Println("Election timeout: ", node.electionTimeout.String())

	// Todo: use https://pkg.go.dev/github.com/gin-gonic/gin?tab=overview
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/vote", node.handleVote)
	http.HandleFunc("/append", node.handleAppend)
	http.HandleFunc("/", node.handleData)

	fmt.Println("Server listening on port " + port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
