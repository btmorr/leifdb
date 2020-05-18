package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
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
	electionTimeout time.Duration
	electionTimer   *time.Timer
	State           Role
	Term            int
	otherNodes      map[string]bool
}

func (n *Node) doElection() bool {
	fmt.Println("Starting Election")
	n.Term = n.Term + 1
	fmt.Println("\tNew Term: ", n.Term)
	fmt.Println("\tN other nodes: ", len(n.otherNodes))
	fmt.Println("\tVotes needed: ")
	return true
}

func NewNode() *Node {
	lowerBound := 150
	upperBound := 300
	timeout := time.Duration((rand.Int()%lowerBound)+(upperBound-lowerBound)) * time.Millisecond

	n := Node{
		Value:           "",
		electionTimeout: timeout,
		electionTimer:   time.NewTimer(timeout),
		State:           FOLLOWER,
		Term:            0,
		otherNodes:      make(map[string]bool)}

	go func() {
		<-n.electionTimer.C
		n.doElection()
	}()
	
	return &n
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Methods for handling data read/write

type DataBody struct {
	Value string `json:"value"`
}

type WriteResponse struct {
	Value string `json:"value"`
}

func (n *Node) handleDataWrite(w http.ResponseWriter, r *http.Request) {
	raw, err1 := ioutil.ReadAll(r.Body)
	if err1 != nil {
		w.WriteHeader(http.StatusBadRequest)
		error := ErrorResponse{Error: "Body required"}
		b, _ := json.Marshal(error)
		fmt.Fprintln(w, string(b))
		return
	}

	var data DataBody
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

// Methods for handling Raft protocol interactions

type VoteBody struct {
	Term int `json:"term"`
}

type VoteResponse struct {
	Term int `json:"term"`
}

func (n *Node) handleVoteRequest(w http.ResponseWriter, r *http.Request) {
	idx := strings.LastIndex(r.RemoteAddr, ":")
	addr := r.RemoteAddr[:idx]
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

type AppendResponse struct {
	Status string `json:"status"`
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

type HealthResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[health] ", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	res := HealthResponse{Status: "Ok"}
	b, _ := json.Marshal(res)
	fmt.Fprintln(w, string(b))
}

func main() {
	port := "8080"

	node := NewNode()
	fmt.Println("Election timeout: ", node.electionTimeout.String())

	// t1.Stop()

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/vote", node.handleVote)
	http.HandleFunc("/stop", node.handleStop)
	http.HandleFunc("/", node.handleData)

	fmt.Println("Server listening on port " + port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
