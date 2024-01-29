package main

import (
	"context"
	"testing"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test server
func createTestServerForFileCloser() *server {
	ctx, cancel := context.WithCancel(context.Background())
	return &server{
		ctx:            ctx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
}

func TestCreateFileCloserChannels(t *testing.T) {
	// Act
	channels := createFileCloserChannels(100)

	// Assert
	require.NotNil(t, channels.EvtReceived)
	require.NotNil(t, channels.EvtDecoded)
	require.NotNil(t, channels.EvtWritten)
	require.NotNil(t, channels.EvtWithError)
	require.NotNil(t, channels.DataEnd)
	require.NotNil(t, channels.CloseFiles)

	// Verify channel buffer sizes
	assert.Equal(t, 100, cap(channels.EvtReceived))
	assert.Equal(t, 100, cap(channels.EvtDecoded))
	assert.Equal(t, 100, cap(channels.EvtWritten))
	assert.Equal(t, 100, cap(channels.EvtWithError))
	assert.Equal(t, 100, cap(channels.DataEnd))
	assert.Equal(t, 100, cap(channels.CloseFiles))
}

func TestFileCloser_AllEventsProcessed_ClosesImmediately(t *testing.T) {
	// Setup
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 1 * time.Second // Not used in this test

	// Start fileCloser in goroutine
	done := make(chan bool)
	go func() {
		fileCloser(s, channels, subRun, timeout)
		done <- true
	}()

	// Act - send one event through the complete pipeline
	channels.EvtReceived <- 1
	channels.EvtDecoded <- 1
	channels.EvtWritten <- 1
	channels.DataEnd <- true

	// Assert - should close immediately
	select {
	case <-channels.CloseFiles:
		// Expected - file should close
	case <-time.After(1 * time.Second):
		t.Fatal("Expected file to close but timed out")
	}

	// Verify we got both close signals (one for each trigger)
	select {
	case <-channels.CloseFiles:
		// Expected - second close signal
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected second close signal but timed out")
	}

	// Wait for fileCloser to finish
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("fileCloser didn't finish")
	}
}

func TestFileCloser_MultipleEvents_WaitsForAll(t *testing.T) {
	// Setup
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 1 * time.Second

	// Start fileCloser in goroutine
	go fileCloser(s, channels, subRun, timeout)

	// Act - send multiple events
	channels.EvtReceived <- 1
	channels.EvtReceived <- 2
	channels.EvtReceived <- 3

	// Give it a moment for events to be processed
	time.Sleep(10 * time.Millisecond)

	// Send DataEnd but not all events are processed yet
	channels.DataEnd <- true

	// Give it a moment - should not close yet
	select {
	case <-channels.CloseFiles:
		t.Fatal("File should not close until all events are processed")
	case <-time.After(100 * time.Millisecond):
		// Expected - still waiting
	}

	// Now process events
	channels.EvtDecoded <- 1
	channels.EvtWritten <- 1
	channels.EvtDecoded <- 2
	channels.EvtWritten <- 2

	// Give it a moment to process
	time.Sleep(10 * time.Millisecond)

	// Still one event pending
	select {
	case <-channels.CloseFiles:
		t.Fatal("File should not close until all events are processed")
	case <-time.After(100 * time.Millisecond):
		// Expected - still waiting
	}

	// Process last event
	channels.EvtDecoded <- 3
	channels.EvtWritten <- 3

	// Assert - now should close
	select {
	case <-channels.CloseFiles:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected file to close after all events processed")
	}
}

func TestFileCloser_EventsWithErrors_CountedCorrectly(t *testing.T) {
	// Setup
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 1 * time.Second

	// Start fileCloser in goroutine
	go fileCloser(s, channels, subRun, timeout)

	// Act - send events with one error
	channels.EvtReceived <- 1
	channels.EvtReceived <- 2
	channels.EvtReceived <- 3
	channels.DataEnd <- true

	// Event 1: decoded and written successfully
	channels.EvtDecoded <- 1
	channels.EvtWritten <- 1

	// Event 2: has error (not decoded or written)
	channels.EvtWithError <- 2

	// Event 3: decoded and written successfully
	channels.EvtDecoded <- 3
	channels.EvtWritten <- 3

	// Assert - should close now (received=3, decoded=2, written=2, error=1)
	select {
	case <-channels.CloseFiles:
		// Expected - first close signal
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected file to close")
	}

	select {
	case <-channels.CloseFiles:
		// Expected - second close signal
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected second close signal")
	}
}

func TestFileCloser_AllErrorEvents_ClosesImmediately(t *testing.T) {
	// Setup
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 1 * time.Second

	// Start fileCloser in goroutine
	go fileCloser(s, channels, subRun, timeout)

	// Act - send events that all error
	channels.EvtReceived <- 1
	channels.EvtReceived <- 2
	channels.DataEnd <- true

	channels.EvtWithError <- 1
	channels.EvtWithError <- 2

	// Assert - should close immediately
	// received=2, decoded=0, written=0, error=2
	// (received == decoded + error) && (decoded == written) => (2 == 0+2) && (0 == 0) => true
	select {
	case <-channels.CloseFiles:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected file to close when all events have errors")
	}
}

func TestFileCloser_NoEvents_ClosesOnDataEnd(t *testing.T) {
	// Setup
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 1 * time.Second

	// Start fileCloser in goroutine
	go fileCloser(s, channels, subRun, timeout)

	// Act - send DataEnd with no events
	channels.DataEnd <- true

	// Assert - should close immediately (no events to wait for)
	select {
	case <-channels.CloseFiles:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected file to close immediately when no events")
	}
}

func TestFileCloser_EventsBeforeDataEnd_WaitsCorrectly(t *testing.T) {
	// Setup - test that events sent before DataEnd are tracked
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 1 * time.Second

	// Start fileCloser in goroutine
	go fileCloser(s, channels, subRun, timeout)

	// Act - process events before sending DataEnd
	channels.EvtReceived <- 1
	channels.EvtDecoded <- 1
	channels.EvtWritten <- 1

	channels.EvtReceived <- 2
	channels.EvtDecoded <- 2
	channels.EvtWritten <- 2

	// Now signal end
	channels.DataEnd <- true

	// Assert - should close immediately since all events already processed
	select {
	case <-channels.CloseFiles:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected file to close immediately when events already processed")
	}
}

func TestFileCloser_OutOfOrderProcessing_HandledCorrectly(t *testing.T) {
	// Setup
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 1 * time.Second

	// Start fileCloser in goroutine
	go fileCloser(s, channels, subRun, timeout)

	// Act - events arrive and process out of order
	channels.EvtReceived <- 1
	channels.EvtReceived <- 2
	channels.EvtReceived <- 3

	// Process in different order
	channels.EvtDecoded <- 3
	channels.EvtDecoded <- 1
	channels.EvtWritten <- 1
	channels.EvtWritten <- 3
	channels.EvtDecoded <- 2
	channels.EvtWritten <- 2

	channels.DataEnd <- true

	// Assert - should close (order doesn't matter, just counts)
	select {
	case <-channels.CloseFiles:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected file to close regardless of processing order")
	}
}

// TestFileCloser_RealTimeout_ForceCloses tests the actual timeout mechanism
func TestFileCloser_RealTimeout_ForceCloses(t *testing.T) {
	// Setup
	s := createTestServerForFileCloser()
	channels := createFileCloserChannels(10)
	subRun := 0
	timeout := 50 * time.Millisecond // Short timeout for testing

	// Start fileCloser in goroutine
	done := make(chan bool, 1)
	go func() {
		fileCloser(s, channels, subRun, timeout)
		done <- true
	}()

	// Act - send DataEnd with incomplete events (missing one event)
	channels.EvtReceived <- 1
	channels.EvtReceived <- 2
	channels.DataEnd <- true

	// Process only first event
	channels.EvtDecoded <- 1
	channels.EvtWritten <- 1

	// Second event never completes - should trigger timeout

	// Assert - should close after timeout (with time buffer)
	select {
	case <-channels.CloseFiles:
		// Expected - first close signal
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Expected file to close after timeout")
	}

	// Should get second close signal
	select {
	case <-channels.CloseFiles:
		// Expected - second close signal
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected second close signal after timeout")
	}

	// Wait for fileCloser to finish
	select {
	case <-done:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("fileCloser didn't finish after timeout")
	}
}
