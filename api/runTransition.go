package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

type RunTransitionState int

const (
	TransitionIdle     RunTransitionState = iota
	TransitionStarting RunTransitionState = iota
	TransitionStopping RunTransitionState = iota
	TransitionError    RunTransitionState = iota
)

type RunTransition struct {
	mu     sync.Mutex
	state  RunTransitionState
	errMsg string
}

func (rt *RunTransition) tryBeginStart() (ok bool, msg string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	switch rt.state {
	case TransitionStarting:
		return false, "start already in progress"
	case TransitionStopping:
		return false, "stop in progress, cannot start"
	default:
		rt.state = TransitionStarting
		rt.errMsg = ""
		return true, ""
	}
}

func (rt *RunTransition) tryBeginStop() (ok bool, msg string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	switch rt.state {
	case TransitionStopping:
		return false, "stop already in progress"
	case TransitionStarting:
		return false, "start in progress, cannot stop"
	default:
		rt.state = TransitionStopping
		rt.errMsg = ""
		return true, ""
	}
}

func (rt *RunTransition) setDone() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.state = TransitionIdle
	rt.errMsg = ""
}

func (rt *RunTransition) setError(msg string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.state = TransitionError
	rt.errMsg = msg
}

func (rt *RunTransition) Status() (RunTransitionState, string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.state, rt.errMsg
}

// WaitForIdle polls until the state is Idle or Error, or the timeout elapses.
// Returns true if the state reached Idle or Error within the timeout.
func (rt *RunTransition) WaitForIdle(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, _ := rt.Status()
		if state == TransitionIdle || state == TransitionError {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func stateString(s RunTransitionState) string {
	switch s {
	case TransitionStarting:
		return "starting"
	case TransitionStopping:
		return "stopping"
	case TransitionError:
		return "error"
	default:
		return "idle"
	}
}

func recoverToError(rt *RunTransition) {
	if recovered := recover(); recovered != nil {
		handleRunTransitionPanic(rt, nil, "run-transition", recovered)
	}
}

func recoverToErrorWithHub(rt *RunTransition, hub *sentry.Hub, operation string) {
	if recovered := recover(); recovered != nil {
		handleRunTransitionPanic(rt, hub, operation, recovered)
	}
}

func handleRunTransitionPanic(rt *RunTransition, hub *sentry.Hub, operation string, recovered any) {
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("operation", operation)
		hub.Recover(recovered)
	})
	rt.setError(fmt.Sprintf("panic: %v", recovered))
}
