package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"

	// "strings"
	"time"
)

type LogRecord struct {
	Value     string
	Timestamp int
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
	State           Role
	Term            int
	votedFor        string
	otherNodes      map[string]bool
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
}

type AppendResponse struct {
	Status string `json:"status"`
}

// /health is a GET endpoint, so no Body type
type HealthResponse struct {
	Status string `json:"status"`
}

// Client methods for managing raft state

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
	timeout := time.Duration(ms) * time.Millisecond

	n := Node{
		Value:           "",
		NodeId:          port,
		electionTimeout: timeout,
		electionTimer:   time.NewTimer(timeout),
		State:           FOLLOWER,
		Term:            0,
		votedFor:        "",
		otherNodes:      make(map[string]bool)}

	go func() {
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
	res := AppendResponse{Status: "Ok"}
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

func (n *Node) handleStop(w http.ResponseWriter, r *http.Request) {
	// this is a debug fn--rip out this and the endpoint
	fmt.Println("[stop] ", r.Method, r.URL.Path)
	n.electionTimer.Stop()
	fmt.Fprintln(w, "Ok")
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
	port := "8080"

	node := NewNode(port)
	fmt.Println("Election timeout: ", node.electionTimeout.String())

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/vote", node.handleVote)
	http.HandleFunc("/stop", node.handleStop)
	http.HandleFunc("/", node.handleData)

	fmt.Println("Server listening on port " + port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
