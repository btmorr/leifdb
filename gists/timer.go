// +build ignore

// demonstration of a more coherent state machine manager

package main

import (
	"errors"
	"fmt"
	"time"
)

type role string

const (
	Leader    role = "Leader"
	Candidate      = "Candidate"
	Follower       = "Follower"
)

type State interface {
	stop()
	restart()
	stateType() role
}

type LeaderState struct {
	done chan bool
}

func (l *LeaderState) stop() {
	l.done <- true
	return
}

func (l *LeaderState) restart() {
	return
}

func (l *LeaderState) stateType() role {
	return Leader
}

func NewLeaderState() *LeaderState {
	l := &LeaderState{done: make(chan bool)}
	t := time.NewTicker(time.Millisecond * 300)
	go func() {
		for {
			select {
			case <-l.done:
				fmt.Println("Halting leader job")
				return
			case ts := <-t.C:
				fmt.Println("Leader job at", ts)
			}
		}
	}()
	return l
}

type FollowerState struct {
	timer    *time.Timer
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
	f.timer.Reset(time.Second)
	return
}

func (f *FollowerState) stateType() role {
	return Follower
}

func NewFollowerState(electionFlag chan bool) *FollowerState {
	t := time.NewTimer(time.Second)
	f := &FollowerState{timer: t, election: electionFlag}
	go func() {
		<-t.C
		fmt.Println("Follower job")
		f.election <- true
	}()
	return f
}

type StateManager struct {
	watchedState State
	electionFlag chan bool
}

func (s *StateManager) changeState(newState role) {
	s.watchedState.stop()
	fmt.Println("Switching to", newState)
	if newState == Leader {
		s.watchedState = NewLeaderState()
	} else if newState == Follower {
		s.watchedState = NewFollowerState(s.electionFlag)
	}
	return
}

var (
	ErrIncorrectState = errors.New("Incorrect state for action")
)

func (s *StateManager) resetTimer() error {
	if s.watchedState.stateType() == Leader {
		return ErrIncorrectState
	}
	s.watchedState.restart()
	return nil
}

func NewStateManager() *StateManager {
	c := make(chan bool)
	s := &StateManager{
		watchedState: NewFollowerState(c),
		electionFlag: c}

	go func() {
		for {
			switch {
			case <-s.electionFlag:
				s.changeState(Leader)
			default:
			}
		}
	}()

	return s
}

func main() {
	w := NewStateManager()
	fmt.Println("T00.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T00.25")
	time.Sleep(time.Millisecond * 250)
	w.resetTimer()
	fmt.Println("T00.50")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T00.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T01.00")
	time.Sleep(time.Millisecond * 250)
	w.resetTimer()
	fmt.Println("T01.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T01.50")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T01.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T02.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("Expect follower job")
	fmt.Println("T02.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T02.50")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T02.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T03.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T03.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T03.50")
	time.Sleep(time.Millisecond * 250)
	w.changeState(Follower)
	fmt.Println("T03.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T04.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T04.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T04.50")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("Expect follower job")
	fmt.Println("T04.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T05.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T05.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T05.50")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T05.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T06.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T06.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T06.50")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T06.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T07.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T07.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T07.50")
	time.Sleep(time.Millisecond * 250)
	w.changeState(Follower)
	fmt.Println("T07.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T08.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T08.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T08.50")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("Expect follower job")
	fmt.Println("T08.75")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T09.00")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T09.25")
	time.Sleep(time.Millisecond * 250)
	fmt.Println("T09.50")
}
