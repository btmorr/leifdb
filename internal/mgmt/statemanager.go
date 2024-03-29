package mgmt

// current design of StateManager is meant to decouple managers from Node,
// but the behavioral needs are too closely coupled anyway. This module should
// be simplified by allowing StateManager to have a reference to a Node.
// Node should still not have any more awareness of StateManager, if for no
// other reason than to prevent circular dependencies.
// [todo: add a ref to Node and simplify StateManager logic]
import (
	"time"

	"github.com/btmorr/leifdb/internal/node"
)

// There is also a virtual role of Candidate when an election is in progress,
// but for the StateManager this is not different from Follower

type state interface {
	stop()
	restart()
	stateType() node.Role
}

type leaderState struct {
	done chan bool
	job  func()
}

func (l *leaderState) stop() {
	l.done <- true
	return
}

func (l *leaderState) restart() {}

func (l leaderState) stateType() node.Role {
	return node.Leader
}

func newLeaderState(job func(), interval time.Duration, grace time.Duration, graceJob func()) *leaderState {
	l := &leaderState{done: make(chan bool), job: job}
	l.job()
	graceCountdown := time.NewTimer(grace)
	go func() {
		<-graceCountdown.C
		graceJob()
	}()
	appendTicker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-l.done:
				return
			case <-appendTicker.C:
				l.job()
			default:
			}
		}
	}()
	return l
}

type followerState struct {
	timer    *time.Timer
	timeout  time.Duration
	election chan bool
}

func (f *followerState) stop() {
	if !f.timer.Stop() {
		select {
		case <-f.timer.C:
		default:
		}
	}
	return
}

func (f *followerState) restart() {
	f.stop()
	f.timer.Reset(f.timeout)
	return
}

func (f followerState) stateType() node.Role {
	return node.Follower
}

func newFollowerState(signal chan bool, timeout time.Duration) *followerState {
	t := time.NewTimer(timeout)
	f := &followerState{timer: t, timeout: timeout, election: signal}
	go func() {
		<-t.C
		f.election <- true
	}()
	return f
}

// StateManager handles the aspects of the Raft protocol that require timing
type StateManager struct {
	state           state
	electionFlag    chan bool
	electionTimeout time.Duration
	graceWindow     time.Duration
	graceEndJob     func()
	appendInterval  time.Duration
	appendJob       func()
}

func (s *StateManager) changeState(newState node.Role) {
	s.state.stop()
	if newState == node.Leader {
		s.state = newLeaderState(s.appendJob, s.appendInterval, s.graceWindow, s.graceEndJob)
	} else {
		s.state = newFollowerState(s.electionFlag, s.electionTimeout)
	}
}

// ResetTimer restarts the countdown on the election timer if the current state
// is Follower (does nothing if the current state is Leader)
func (s *StateManager) ResetTimer() {
	s.state.restart()
}

// BecomeFollower explicitly changes the state to Follower
func (s *StateManager) BecomeFollower() {
	s.changeState(node.Follower)
	s.graceEndJob()
}

// NewStateManager creates a StateManager with state initialized to Follower
// followFlag is a channel that indicates the node should reset the election
// timer, including becoming a Follower if the current state is Leader
//
// electionTimeout is the duration a node should wait before starting an
// election. Events that delay an election should call `ResetTimer`. If the
// timer expires, the electionJob function is called
//
// electionJob is a function that is called when the election timer expires,
// which should return a boolean designating whether the node should become
// a Leader (on true), or remain a Follower (on false)
//
// appendInterval is the period between append requests when a node is a Leader
//
// # The ticker ticks on this period, and calls appendJob
//
// appendJob is the task that a Leader should perform after each appendInterval
func NewStateManager(
	resetFlag chan bool,
	electionTimeout time.Duration,
	electionJob func() bool,
	graceWindow time.Duration,
	graceEndJob func(),
	appendInterval time.Duration,
	appendJob func()) *StateManager {

	c := make(chan bool)
	s := &StateManager{
		state:           newFollowerState(c, electionTimeout),
		electionFlag:    c,
		electionTimeout: electionTimeout,
		graceWindow:     graceWindow,
		graceEndJob:     graceEndJob,
		appendInterval:  appendInterval,
		appendJob:       appendJob}

	go func() {
		for {
			select {
			case <-s.electionFlag:
				s.ResetTimer()
				// Note that "Candidate" state is a virtual state--in between this
				// `ResetTimer` call and the return of the following `electionJob`, the
				// state of the node corresponds to the Candidate state, but the
				// behavior is not meaningfully different from during a Follower
				// period (from the perspective of the StateManager). The `electionJob`
				// function should perform any side-effects that are unique to the
				// Candidate state.
				if electionJob() {
					s.changeState(node.Leader)
				} else {
					s.changeState(node.Follower)
				}
			case <-resetFlag:
				s.BecomeFollower()
			default:
			}
		}
	}()

	return s
}
