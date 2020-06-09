package mgmt

import (
	"errors"
	"time"
)

var (
	ErrIncorrectState = errors.New("Incorrect state for action")
)

type role string

const (
	Leader   role = "Leader"
	Follower      = "Follower"
)

type State interface {
	stop()
	restart()
	stateType() role
}

type LeaderState struct {
	done chan bool
	job  func()
}

func (l *LeaderState) stop() {
	l.done <- true
	return
}

func (l *LeaderState) restart() {}

func (l LeaderState) stateType() role {
	return Leader
}

func NewLeaderState(job func(), timeout time.Duration) *LeaderState {
	l := &LeaderState{done: make(chan bool), job: job}
	t := time.NewTicker(timeout)
	go func() {
		for {
			select {
			case <-l.done:
				return
			case <-t.C:
				l.job()
			}
		}
	}()
	return l
}

type FollowerState struct {
	timer    *time.Timer
	timeout  time.Duration
	election chan bool
}

func (f *FollowerState) stop() {
	if !f.timer.Stop() {
		select {
		case <-f.timer.C:
		default:
		}
	}
	return
}

func (f *FollowerState) restart() {
	f.stop()
	f.timer.Reset(f.timeout)
	return
}

func (f FollowerState) stateType() role {
	return Follower
}

func NewFollowerState(electionFlag chan bool, timeout time.Duration) *FollowerState {
	t := time.NewTimer(timeout)
	f := &FollowerState{timer: t, timeout: timeout, election: electionFlag}
	go func() {
		<-t.C
		f.election <- true
	}()
	return f
}

type StateManager struct {
	state           State
	electionFlag    chan bool
	electionTimeout time.Duration
	appendTimeout   time.Duration
	appendJob       func()
}

func (s *StateManager) changeState(newState role) {
	s.state.stop()
	if newState == Leader {
		s.state = NewLeaderState(s.appendJob, s.appendTimeout)
	} else {
		s.state = NewFollowerState(s.electionFlag, s.electionTimeout)
	}
}

func (s *StateManager) resetTimer() {
	s.state.restart()
}

func NewStateManager(
	electionTimeout time.Duration,
	electionJob func() bool,
	appendTimeout time.Duration,
	appendJob func()) *StateManager {

	c := make(chan bool)
	s := &StateManager{
		state:           NewFollowerState(c, electionTimeout),
		electionFlag:    c,
		electionTimeout: electionTimeout,
		appendTimeout:   appendTimeout,
		appendJob:       appendJob}

	go func() {
		for {
			switch {
			case <-s.electionFlag:
				s.resetTimer()
				// Note that "Candidate" state is a virtual state--in between this
				// `resetTimer` call and the return of the following `electionJob`, the
				// state of the node corresponds to the Candidate state, but the
				// behavior is not meaningfully different from during a Follower
				// period (from the perspective of the StateManager). The `electionJob`
				// function should perform any side-effects that are unique to the
				// Candidate state.
				if electionJob() {
					s.changeState(Leader)
				} else {
					s.changeState(Follower)
				}
			default:
			}
		}
	}()

	return s
}
