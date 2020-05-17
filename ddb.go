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

func (n *Node) handleWrite(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	data := new(DataBody)
	d.Decode(&data)
	fmt.Println("\tBody: ", *data)
	n.Value = data.Value
	fmt.Fprintf(w, "Ok")
}

func (n Node) handleRead(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", n.Value)
}

func (n *Node) dataHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[data] ", r.Method, r.URL.Path)
	if r.Method == "POST" {
		n.handleWrite(w, r)
	} else if r.Method == "GET" {
		n.handleRead(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Unsupported verb", r.Method)
	}
}

// Methods for handling Raft protocol interactions
func (n *Node) handleElectionRequest(w http.ResponseWriter, r *http.Request) {
	idx := strings.LastIndex(r.RemoteAddr, ":")
	addr := r.RemoteAddr[:idx]
	n.otherNodes[addr] = true
	fmt.Println("Added ", addr, " to known nodes")
	fmt.Println("Nodes: ", n.otherNodes)
	fmt.Fprintln(w, "Ok")
}

func (n *Node) raftHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[raft] ", r.Method, r.URL.Path)
		if r.Method == "POST" {
		n.handleElectionRequest(w, r)
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
	http.HandleFunc("/raft", node.raftHandler)
	http.HandleFunc("/stop", node.handleStop)
	http.HandleFunc("/", node.dataHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}