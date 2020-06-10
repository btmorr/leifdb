package mgmt

import (
	// "fmt"
	"testing"
	"time"
)

func TestManager(t *testing.T) {
	electionTimeout := time.Second
	appendTimeout := time.Millisecond * 50

	electionCounter := 0
	appendCounter := 0
	electionShouldSucceed := true

	var currentState Role
	currentState = Follower
	followFlag := make(chan bool)

	mgmt := NewStateManager(
		followFlag,                        // reset channel
		func(r Role) { currentState = r }, // state change callback
		electionTimeout,                   // election timeout
		func() bool { // election job
			electionCounter++
			return electionShouldSucceed
		},
		appendTimeout, // append timeout
		func() { // append job
			appendCounter++
			return
		})

	t.Skip("This test is flaky because of dependence on timing windows")
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to be Follower but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*20)
	if electionCounter != 1 {
		t.Errorf("Expected %d elections, got %d\n", 1, electionCounter)
	}
	if mgmt.state.stateType() != Leader {
		t.Errorf("Expected state to be Leader after election, but got %s\n", mgmt.state.stateType())
	}
	if currentState != Leader {
		t.Error("Expected StateManager to call update hook")
	}
	// Is there a better way to test ticker behavior? I'd prefer not to have
	// a time.Sleep in a test, but not sure how else to deal with this. Made
	// the timeouts as short as possible, but any shorter and the job func
	// throws the timing.
	n := 5
	time.Sleep(appendTimeout * time.Duration(n+1))
	if appendCounter < n {
		t.Errorf("Expected at least %d appends, got %d\n", n, appendCounter)
	}
	electionShouldSucceed = false
	go func() {
		followFlag <- true
	}()
	time.Sleep(time.Microsecond * 500)
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to be Follower, but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*20)
	if electionCounter != 2 {
		t.Errorf("Expected %d elections, got %d\n", 2, electionCounter)
	}
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to be Follower after failed election, but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout / time.Duration(2))
	go func() {
		followFlag <- true
	}()
	time.Sleep(electionTimeout / time.Duration(2))
	go func() {
		followFlag <- true
	}()
	time.Sleep(electionTimeout / time.Duration(2))
	go func() {
		followFlag <- true
	}()
	if electionCounter != 2 {
		t.Errorf("Expected %d elections, got %d\n", 2, electionCounter)
	}
	if mgmt.state.stateType() != Follower {
		t.Errorf("Expected state to still be Follower after resets, but got %s\n", mgmt.state.stateType())
	}
	time.Sleep(electionTimeout + time.Millisecond*20)
	if electionCounter != 3 {
		t.Errorf("Expected %d elections, got %d\n", 3, electionCounter)
	}
}
