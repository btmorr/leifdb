package main

import (
	"fmt"
	"log"
	"time"
	"strings"
	"math/rand"
	"encoding/json"
	"net/http"
)

type LogRecord struct {
	Value string
	Timestamp int
}

type Role int

const (
	FOLLOWER Role = iota
	CANDIDATE
	LEADER
)

type Node struct {
	Value string
	electionTimeout time.Duration
	electionTimer *time.Timer
	State Role
	Term int
	otherNodes map[string]bool
}

func (n *Node) doElection() bool {
	fmt.Println("Starting Election")
	n.Term = n.Term + 1
	fmt.Println("\tNew Term: ", n.Term)
	fmt.Println("\tN other nodes: ", len(n.otherNodes))
	fmt.Println("\tVotes needed: ", )
	return true
}

func NewNode() *Node {
	timeout := time.Duration((rand.Int() % 150) + 150) * time.Millisecond
	n := Node{
		Value: "",
		electionTimeout: timeout,
		electionTimer: time.NewTimer(timeout),
		State: FOLLOWER,
		Term: 0,
		otherNodes: make(map[string]bool)}
	go func() {
		<-n.electionTimer.C
		n.doElection()
	}()
	return &n
}

// Methods for handling data read/write

type DataBody struct {
	Value string
}

func (n *Node) handleDataWrite(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	data := new(DataBody)
	d.Decode(&data)
	fmt.Println("\tBody: ", *data)
	n.Value = data.Value
	fmt.Fprintf(w, "Ok")
}

func (n Node) handleDataRead(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", n.Value)
}

func (n *Node) handleData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[data] ", r.Method, r.URL.Path)
	if r.Method == http.MethodPost {
		n.handleDataWrite(w, r)
	} else if r.Method == http.MethodGet {
		n.handleDataRead(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Unsupported verb", r.Method)
	}
}

// Methods for handling Raft protocol interactions

type VoteBody struct {
	Term int
}

func (n *Node) handleVoteRequest(w http.ResponseWriter, r *http.Request) {
	idx := strings.LastIndex(r.RemoteAddr, ":")
	addr := r.RemoteAddr[:idx]
	n.otherNodes[addr] = true
	fmt.Println("Added ", addr, " to known nodes")
	fmt.Println("Nodes: ", n.otherNodes)

	d := json.NewDecoder(r.Body)
	body := new(VoteBody)
	d.Decode(&body)
	fmt.Println("\tTerm: ", *body)

	w.Header().Set("Content-Type", "application/json")
	
	if body.Term <= n.Term {
		// Use 409 Conflict to represent invalid term
		n.Term = n.Term + 1
		fmt.Println("Expired term vote received. New term: ", n.Term)
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintln(w, "Conflict: invalid term")
	} else {
		fmt.Println("Voting for ", addr, " for term ", body.Term)
		n.Term = body.Term
		fmt.Fprintln(w, "Accepted")
	}
}

func (n *Node) handleAppendLogsRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Ok")
}

func (n *Node) handleVote(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[vote] ", r.Method, r.URL.Path)
	 if r.Method == http.MethodPost {
		// This is a request for Vote
		n.handleVoteRequest(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Unsupported verb", r.Method)
	}
}

func(n *Node) handleStop(w http.ResponseWriter, r *http.Request) {
	// this is a debug fn--rip out this and the endpoint
	n.electionTimer.Stop()
	fmt.Fprintln(w, "Ok")
}

// Other stuff

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Ok")
}

func main() {
	port := "8080"

	node := NewNode()
	fmt.Println("Election timeout: ", node.electionTimeout.String())

	// t1.Stop()

	fmt.Println(fmt.Sprintf("Server listening on port %s", port))
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/vote", node.handleVote)
	http.HandleFunc("/stop", node.handleStop)
	http.HandleFunc("/", node.handleData)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}