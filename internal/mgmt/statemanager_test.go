package mgmt

import (
	"testing"
	"time"
)

func TestManager(t *testing.T) {
	electionTimeout := time.Millisecond * 8
	appendTimeout := time.Millisecond

	electionCounter := 0
	appendCounter := 0
	electionShouldSucceed := true

	mgmt := NewStateManager(
		electionTimeout,
		func() bool {
			electionCounter++
			return electionShouldSucceed
		},
		appendTimeout,
		func() {
			appendCounter++
			return
		})
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to be Follower but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*2)
	if electionCounter != 1 {
		t.Errorf("Expected %d elections, got %d\n", 1, electionCounter)
	}
	if mgmt.state.stateType() != Leader {
		t.Errorf("Expected state to be Leader after election, but got %s\n", mgmt.state.stateType())
	}
	// Is there a better way to test ticker behavior? I'd prefer not to have
	// a time.Sleep in a test, but not sure how else to deal with this. Made
	// the timeouts as short as possible, but any shorter and the job func
	// throws the timing.
	n := 5
	time.Sleep(appendTimeout * time.Duration(n))
	if appendCounter < n {
		t.Errorf("Expected at least %d appends, got %d\n", n, appendCounter)
	}
	electionShouldSucceed = false
	mgmt.changeState(Follower)
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to be Follower, but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*2)
	if electionCounter != 2 {
		t.Errorf("Expected %d elections, got %d\n", 2, electionCounter)
	}
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to be Follower after failed election, but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout / time.Duration(2))
	mgmt.resetTimer()
	time.Sleep(electionTimeout / time.Duration(2))
	mgmt.resetTimer()
	time.Sleep(electionTimeout / time.Duration(2))
	mgmt.resetTimer()
	if electionCounter != 2 {
		t.Errorf("Expected %d elections, got %d\n", 2, electionCounter)
	}
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to still be Follower after resets, but got %s\n", mgmt.state.stateType())
	}
}
