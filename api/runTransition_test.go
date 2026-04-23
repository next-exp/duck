package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunTransition_InitialStateIsIdle(t *testing.T) {
	var rt RunTransition
	state, msg := rt.Status()
	assert.Equal(t, TransitionIdle, state)
	assert.Empty(t, msg)
}

func TestRunTransition_StartAndDone(t *testing.T) {
	var rt RunTransition
	ok, msg := rt.tryBeginStart()
	assert.True(t, ok)
	assert.Empty(t, msg)

	state, _ := rt.Status()
	assert.Equal(t, TransitionStarting, state)

	rt.setDone()
	state, _ = rt.Status()
	assert.Equal(t, TransitionIdle, state)
}

func TestRunTransition_StopAndDone(t *testing.T) {
	var rt RunTransition
	ok, msg := rt.tryBeginStop()
	assert.True(t, ok)
	assert.Empty(t, msg)

	state, _ := rt.Status()
	assert.Equal(t, TransitionStopping, state)

	rt.setDone()
	state, _ = rt.Status()
	assert.Equal(t, TransitionIdle, state)
}

func TestRunTransition_DoubleStart(t *testing.T) {
	var rt RunTransition
	ok, _ := rt.tryBeginStart()
	assert.True(t, ok)

	ok, msg := rt.tryBeginStart()
	assert.False(t, ok)
	assert.Equal(t, "start already in progress", msg)
}

func TestRunTransition_DoubleStop(t *testing.T) {
	var rt RunTransition
	ok, _ := rt.tryBeginStop()
	assert.True(t, ok)

	ok, msg := rt.tryBeginStop()
	assert.False(t, ok)
	assert.Equal(t, "stop already in progress", msg)
}

func TestRunTransition_StartWhileStopping(t *testing.T) {
	var rt RunTransition
	ok, _ := rt.tryBeginStop()
	assert.True(t, ok)

	ok, msg := rt.tryBeginStart()
	assert.False(t, ok)
	assert.Equal(t, "stop in progress, cannot start", msg)
}

func TestRunTransition_StopWhileStarting(t *testing.T) {
	var rt RunTransition
	ok, _ := rt.tryBeginStart()
	assert.True(t, ok)

	ok, msg := rt.tryBeginStop()
	assert.False(t, ok)
	assert.Equal(t, "start in progress, cannot stop", msg)
}

func TestRunTransition_ErrorState(t *testing.T) {
	var rt RunTransition
	rt.setError("something went wrong")

	state, msg := rt.Status()
	assert.Equal(t, TransitionError, state)
	assert.Equal(t, "something went wrong", msg)
}

func TestRunTransition_ErrorCleared_OnStart(t *testing.T) {
	var rt RunTransition
	rt.setError("previous failure")

	ok, msg := rt.tryBeginStart()
	assert.True(t, ok)
	assert.Empty(t, msg)

	state, errMsg := rt.Status()
	assert.Equal(t, TransitionStarting, state)
	assert.Empty(t, errMsg)
}

func TestRunTransition_ErrorCleared_OnStop(t *testing.T) {
	var rt RunTransition
	rt.setError("previous failure")

	ok, msg := rt.tryBeginStop()
	assert.True(t, ok)
	assert.Empty(t, msg)

	state, _ := rt.Status()
	assert.Equal(t, TransitionStopping, state)
}

func TestRunTransition_WaitForIdle_ReturnsTrue(t *testing.T) {
	var rt RunTransition
	ok, _ := rt.tryBeginStart()
	assert.True(t, ok)

	go func() {
		time.Sleep(20 * time.Millisecond)
		rt.setDone()
	}()

	reached := rt.WaitForIdle(1 * time.Second)
	assert.True(t, reached)
}

func TestRunTransition_WaitForIdle_Timeout(t *testing.T) {
	var rt RunTransition
	ok, _ := rt.tryBeginStart()
	assert.True(t, ok)

	reached := rt.WaitForIdle(50 * time.Millisecond)
	assert.False(t, reached)
}

func TestStateString(t *testing.T) {
	assert.Equal(t, "idle", stateString(TransitionIdle))
	assert.Equal(t, "starting", stateString(TransitionStarting))
	assert.Equal(t, "stopping", stateString(TransitionStopping))
	assert.Equal(t, "error", stateString(TransitionError))
}
