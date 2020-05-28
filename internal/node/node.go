package node

import (
	
)

// A Role is one of Leader, Candidate, or Follower
type Role int

const (
	Follower Role = iota
	Candidate
	Leader
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
	electionTimeout time.Duration
	electionTimer   *time.Timer
	appendTimeout   time.Duration
	appendTicker    *time.Ticker
	State           Role
	haltAppend      chan bool
	Term            int64
	votedFor        string
	otherNodes      map[string]bool
	nextIndex       map[string]int
	matchIndex      map[string]int
	commitIndex     int
	lastApplied     int
	log             []*raft.EventLog
	config          NodeConfig
}

// Client methods for managing raft state

// Non-volatile state functions
// `Term`, `votedFor`, and `log` must persist through application restart, so
// any request that changes these values must be written to disk before
// responding to the request.

// SetTerm records term and vote in non-volatile state
func (n *Node) SetTerm(newTerm int64, votedFor string) error {
	n.Term = newTerm
	n.votedFor = votedFor
	vote := fmt.Sprintf("%d %s\n", newTerm, votedFor)
	return fileutils.Write(n.config.TermFile, vote)
}

// SetLog records new log contents in non-volatile memory and updates state
func (n *Node) SetLog(newLog []*raft.EventLog]) error {
	out, err := proto.Marshal(&newLog)
	if err != nil {
		log.Fatalln("Failed to encode logs:", err)
	}
	if err := ioutil.WriteFile(n.config.LogFile, out, 0644); err != nil {
		log.Fatalln("Failed to write log file:", err)
	}
	// If marshal and write succeeds, update node state
	n.log = newLog
	return err
}

// Volatile state functions
// Other state should not be written to disk, and should be re-initialized on
// restart, but may have other side-effects that happen on state change

// SetState designates the Node as one of the roles in the Role enumeration,
// and handles any side-effects that should happen specifically on state
// transition
func (n *Node) SetState(newState Role) {
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
func NewNode(config NodeConfig) (*Node, error) {
	lowerBound := 150
	upperBound := 300
	ms := (rand.Int() % lowerBound) + (upperBound - lowerBound)
	electionTimeout := time.Duration(ms) * time.Millisecond

	appendTimeout := time.Duration(10) * time.Millisecond

	var term int64
	var votedFor string
	_, err := os.Stat(config.TermFile)
	if err != nil {
		term = 0
		votedFor = ""
		log.Println("No term data found, starting at term 0")
	} else {
		raw, _ := fileutils.Read(config.TermFile)
		dat := strings.Split(strings.TrimSpace(raw), " ")
		term, _ = strconv.ParseInt(dat[0], 10, 64)
		votedFor = dat[1]
		log.Println("Term data found. Current term:", term)
	}

	// replace next couple of blocks with protobuf persistence
	logFile, err2 := ioutil.ReadFile(config.LogFile)
	if err2 != nil {
		if os.IsNotExist(err) {
			fmt.Println("Log file not found--starting new log file")
		} else {
			log.Fatalln("Error reading log file:", err)
		}
	}

	logs := []*raft.EventLog{}
	if err3 := proto.Unmarshal(logFile, logs); err3 != nil {
		log.Fatalln("Failed to unmarshal log file:", err)
	}

	n := Node{
		Value:           make(map[string]string),
		NodeId:          config.Id,
		electionTimeout: electionTimeout,
		electionTimer:   time.NewTimer(electionTimeout),
		appendTimeout:   appendTimeout,
		appendTicker:    time.NewTicker(appendTimeout),
		State:           Follower,
		haltAppend:      make(chan bool),
		Term:            term,
		votedFor:        votedFor,
		otherNodes:      make(map[string]bool),
		nextIndex:       make(map[string]int),
		matchIndex:      make(map[string]int),
		commitIndex:     0,
		lastApplied:     0,
		log:             *logs,
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
func (n *Node) handleWrite(c *gin.Context) {
	// todo: revise to log-append / commit protocol once elections work
	var data WriteBody
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if n.State != Leader {
		c.JSON(http.StatusForbidden, gin.H{"error": "Writes must be made to leader"})
		return
	}

	newRecord := &raft.LogRecord{
		Term:   n.Term,
		Action: raft.LogRecord_SET,
		Key: data.Key,
		Value: data.Value}
	newLog := []*raft.EventLog{Records: append(n.log.Records, newRecord)}
	err := n.SetLog(newLog)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	} else {
		c.JSON(http.StatusOK, gin.H{"value": n.Value})
	}

}

// Handler for GETs to the data endpoint (client read)
func (n *Node) handleRead(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"value": n.Value})
}

// Handler for DELETEs to the data endpoint (client delete)
func (n *Node) handleDelete(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
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
		// Increment term, vote for same node as previous term (is this correct?)
		n.SetTerm(n.Term+1, n.votedFor)
		log.Println("Expired term vote received. New term: ", n.Term)
		status = http.StatusConflict
		vote = gin.H{"term": n.Term, "voteGranted": false}
	} else {
		log.Println("Voting for ", body.CandidateId, " for term ", body.Term)
		n.SetTerm(body.Term, body.CandidateId)
		if n.State == Leader {
			n.haltAppend <- true
		}
		n.SetState(Follower)
		n.resetElectionTimer()

		// todo: check candidate's log details
		status = http.StatusOK
		vote = gin.H{"term": n.Term, "voteGranted": true}
		log.Println("Returning vote: ", vote)
	}
	log.Println("Returning vote: [", status, " ]", vote)

	c.JSON(status, vote)
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
		byz_msg1 := "Append request from LeaderId mismatch for this term. "
		byz_msg2 := "Possible byzantine actor: " + leaderId + " != " + n.votedFor
		log.Println(byz_msg1 + byz_msg2)
		success = false
	}
	return success
}

// If an existing entry conflicts with a new one (same idx diff term),
// reconcileLogs deletes the existing entry and any that follow
func (n *Node) reconcileLogs(body AppendBody) {
	mismatchIdx := -1
	if body.PrevLogIndex < len(n.log.Records) {
		overlappingEntries := n.log.Records[body.PrevLogIndex:]
		for i, rec := range overlappingEntries {
			if rec.Term != body.Entries[i].Term {
				mismatchIdx = body.PrevLogIndex + i
				break
			}
		}
	}
	if mismatchIdx >= 0 {
		n.log.Records = n.log.Records[:mismatchIdx]
	}
	// append any entries not already in log
	offset := len(n.log.Records) - body.PrevLogIndex
	n.log = []*raft.EventLog{Records: append(n.log.Records, body.Entries[offset:]...)}
	// update commit idx
	if body.LeaderCommit > n.commitIndex {
		if body.LeaderCommit < len(n.log.Records) {
			n.commitIndex = body.LeaderCommit
		} else {
			n.commitIndex = len(n.log.Records)
		}
	}
}

// checkPrevious returns true if Node.logs contains an entry at the specified
// index with the specified term, otherwise false
func (n *Node) checkPrevious(prevIndex int, prevTerm int64) bool {
	return prevIndex < len(n.log.Records) && n.log.Records[prevIndex].Term == prevTerm
}

// Handler for append-log messages from leader nodes
func (n *Node) handleAppend(c *gin.Context) {
	var body AppendBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// if none of the failure conditions fired...
	var status int
	var success bool

	valid := n.validateAppend(body.Term, body.LeaderId)
	matched := n.checkPrevious(body.PrevLogIndex, body.PrevLogTerm)
	if !valid {
		// Invalid request
		status = http.StatusConflict
		success = false
	} else if !matched {
		// Valid request, but earlier entries needed
		status = http.StatusOK
		success = false
	} else {
		// Valid request, and all required logs present
		if len(body.Entries) > 0 {
			n.reconcileLogs(body)
		}

		status = http.StatusOK
		success = true
	}
	if valid {
		// For any valid append received during an election, cancel election
		if n.State == Candidate {
			n.SetState(Follower)
		}

		// only reset the election timer on append from a valid leader
		n.resetElectionTimer()
	}
	// finally
	c.JSON(status, gin.H{"term": n.Term, "success": success})
}