//go:build mgmttest

// This test is timing-dependent, and fails if run along with the rest of the
// test suite. Until we have a better way of testing the timer logic, this test
// is run separately on CI

package mgmt

import (
	"testing"
	"time"

	"github.com/btmorr/leifdb/internal/node"
)

func TestManager(t *testing.T) {
	electionTimeout := time.Second / 4
	minimumTimeout := electionTimeout / 2
	appendInterval := time.Millisecond * 20

	electionCounter := 0
	appendCounter := 0
	electionShouldSucceed := true

	resetFlag := make(chan bool)

	mgmt := NewStateManager(
		resetFlag,
		electionTimeout,
		func() bool { // election job
			electionCounter++
			return electionShouldSucceed
		},
		minimumTimeout,
		func() {}, // grace window job (not checked here)
		appendInterval,
		func() { // append job
			appendCounter++
			return
		})

	if mgmt.state.stateType() != node.Follower {
		t.Errorf(
			"Expected state to be Follower but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*2)
	time.Sleep(appendInterval)
	if electionCounter != 1 {
		t.Errorf("Expected %d elections, got %d\n", 1, electionCounter)
	}
	if mgmt.state.stateType() != node.Leader {
		t.Errorf(
			"Expected state to be Leader after election, but got %s\n",
			mgmt.state.stateType())
	}
	// Is there a better way to test ticker behavior? I'd prefer not to have
	// a time.Sleep in a test, but not sure how else to deal with this. Made
	// the timeouts as short as possible, but any shorter and the job func
	// throws the timing.
	n := 5
	for i := 0; i <= n; i++ {
		time.Sleep(appendInterval)
	}
	if appendCounter < n {
		t.Errorf("Expected at least %d appends, got %d\n", n, appendCounter)
	}
	electionShouldSucceed = false
	go func() {
		resetFlag <- true
	}()
	time.Sleep(time.Millisecond * 2)
	if mgmt.state.stateType() != node.Follower {
		t.Errorf(
			"Expected state to be Follower, but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*2)
	time.Sleep(appendInterval)
	if electionCounter != 2 {
		t.Errorf("Expected %d elections, got %d\n", 2, electionCounter)
	}
	if mgmt.state.stateType() != node.Follower {
		t.Errorf(
			"Expected state to be Follower after failed election, but got %s\n",
			mgmt.state.stateType())
	}
	time.Sleep(electionTimeout / time.Duration(2))
	go func() {
		resetFlag <- true
	}()
	time.Sleep(electionTimeout / time.Duration(2))
	go func() {
		resetFlag <- true
	}()
	time.Sleep(electionTimeout / time.Duration(2))
	go func() {
		resetFlag <- true
	}()
	if electionCounter != 2 {
		t.Errorf("Expected %d elections, got %d\n", 2, electionCounter)
	}
	if mgmt.state.stateType() != node.Follower {
		t.Errorf(
			"Expected state to still be Follower after resets, but got %s\n",
			mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*2)
	time.Sleep(appendInterval)
	if electionCounter != 3 {
		t.Errorf("Expected %d elections, got %d\n", 3, electionCounter)
	}
}

func TestGraceWindow(t *testing.T) {
	electionTimeout := time.Second / 4
	minimumTimeout := electionTimeout / 2
	appendInterval := time.Millisecond * 20

	electionCounter := 0
	appendCounter := 0
	electionShouldSucceed := true
	allowVote := true

	resetFlag := make(chan bool)

	NewStateManager(
		resetFlag,
		electionTimeout,
		func() bool { // election job
			electionCounter++
			if electionShouldSucceed {
				allowVote = false
			}
			return electionShouldSucceed
		},
		minimumTimeout,
		func() {
			allowVote = true
		},
		appendInterval,
		func() { // append job
			appendCounter++
			return
		})

	if !allowVote {
		t.Error("AllowVote should be true for follower")
	}
	time.Sleep(electionTimeout + appendInterval)
	if allowVote {
		t.Error("AllowVote should be false immediately after becoming leader")
	}
	time.Sleep(minimumTimeout + appendInterval)
	if !allowVote {
		t.Error("AllowVote should be true after grace window expires")
	}
}
