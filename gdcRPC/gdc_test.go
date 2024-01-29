//go:build nohdf5

package main

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// TestCountEnabledGDCs_AllEnabled tests counting when all GDCs are enabled
func TestCountEnabledGDCs_AllEnabled(t *testing.T) {
	// Setup
	gdcs := []duck.GDCConfiguration{
		testhelpers.NewTestGDCConfiguration(1, "gdc1"),
		testhelpers.NewTestGDCConfiguration(2, "gdc2"),
		testhelpers.NewTestGDCConfiguration(3, "gdc3"),
	}

	// Act
	count := countEnabledGDCs(gdcs)

	// Assert
	assert.Equal(t, 3, count, "All enabled GDCs should be counted")
}

// TestCountEnabledGDCs_SomeDisabled tests counting when some GDCs are disabled
func TestCountEnabledGDCs_SomeDisabled(t *testing.T) {
	// Setup
	gdc1 := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	gdc2 := testhelpers.NewTestGDCConfiguration(2, "gdc2")
	gdc2.Enabled = false
	gdc3 := testhelpers.NewTestGDCConfiguration(3, "gdc3")

	gdcs := []duck.GDCConfiguration{gdc1, gdc2, gdc3}

	// Act
	count := countEnabledGDCs(gdcs)

	// Assert
	assert.Equal(t, 2, count, "Only enabled GDCs should be counted")
}

// TestCountEnabledGDCs_AllDisabled tests counting when all GDCs are disabled
func TestCountEnabledGDCs_AllDisabled(t *testing.T) {
	// Setup
	gdc1 := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	gdc1.Enabled = false
	gdc2 := testhelpers.NewTestGDCConfiguration(2, "gdc2")
	gdc2.Enabled = false

	gdcs := []duck.GDCConfiguration{gdc1, gdc2}

	// Act
	count := countEnabledGDCs(gdcs)

	// Assert
	assert.Equal(t, 0, count, "No GDCs should be counted when all are disabled")
}

// TestCountEnabledGDCs_EmptyList tests counting with empty GDC list
func TestCountEnabledGDCs_EmptyList(t *testing.T) {
	// Setup
	gdcs := []duck.GDCConfiguration{}

	// Act
	count := countEnabledGDCs(gdcs)

	// Assert
	assert.Equal(t, 0, count, "Empty list should return 0")
}

// TestLDCConnectionMonitor_AllDisconnected_CancelsContext tests that empty connections trigger cancel
func TestLDCConnectionMonitor_AllDisconnected_CancelsContext(t *testing.T) {
	// Setup
	s := createTestServer()
	s.setState(duck.RUNNING)

	openChan := make(chan net.Addr, 10)
	closeChan := make(chan net.Addr, 10)

	baseCtx, cancel := context.WithCancel(context.Background())
	s.ctx = context.WithValue(baseCtx, "cancelCtx", cancel)
	defer cancel()

	done := make(chan bool, 1)

	// Act - start monitor with no connections
	go func() {
		ldcConnectionMonitor(s, openChan, closeChan)
		done <- true
	}()

	// Send a close event (simulating all connections closed)
	closeChan <- &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 6006,
	}

	// Assert - context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected - context cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled when all connections are closed")
	}

	// Monitor should exit
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Monitor should exit after cancelling context")
	}
}

// TestLDCConnectionMonitor_PartialConnections_StateTransitions tests state transitions with partial connections
func TestLDCConnectionMonitor_PartialConnections_StateTransitions(t *testing.T) {
	// Setup
	s := createTestServer()
	s.setState(duck.RUNNING)

	openChan := make(chan net.Addr, 10)
	closeChan := make(chan net.Addr, 10)

	baseCtx, cancel := context.WithCancel(context.Background())
	s.ctx = context.WithValue(baseCtx, "cancelCtx", cancel)
	defer cancel()

	done := make(chan bool, 1)

	// Act - start monitor
	go func() {
		ldcConnectionMonitor(s, openChan, closeChan)
		done <- true
	}()

	// Open some connections
	addr1 := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6006}
	addr2 := &net.TCPAddr{IP: net.ParseIP("127.0.0.2"), Port: 6007}

	openChan <- addr1
	openChan <- addr2

	// Give monitor time to process
	time.Sleep(10 * time.Millisecond)

	// Close one connection
	closeChan <- addr1

	// Give monitor time to process
	time.Sleep(10 * time.Millisecond)

	// Assert - state should be STOPPING
	assert.Eventually(t, func() bool {
		return s.getState() == duck.STOPPING
	}, 100*time.Millisecond, 10*time.Millisecond, "State should transition to STOPPING")

	// Close remaining connection
	closeChan <- addr2

	// Assert - context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled when all connections close")
	}

	// Cleanup
	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Monitor should exit")
	}
}

// TestLDCConnectionMonitor_MultipleConnectionsOpenClose tests tracking multiple connections
func TestLDCConnectionMonitor_MultipleConnectionsOpenClose(t *testing.T) {
	// Setup
	s := createTestServer()
	s.setState(duck.RUNNING)

	openChan := make(chan net.Addr, 10)
	closeChan := make(chan net.Addr, 10)

	baseCtx, cancel := context.WithCancel(context.Background())
	s.ctx = context.WithValue(baseCtx, "cancelCtx", cancel)
	defer cancel()

	done := make(chan bool, 1)

	// Act - start monitor
	go func() {
		ldcConnectionMonitor(s, openChan, closeChan)
		done <- true
	}()

	// Open multiple connections
	numConnections := 5
	addrs := make([]net.Addr, numConnections)
	for i := 0; i < numConnections; i++ {
		addrs[i] = &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 6000 + i,
		}
		openChan <- addrs[i]
	}

	time.Sleep(10 * time.Millisecond)

	// Close connections one by one
	for i := 0; i < numConnections-1; i++ {
		closeChan <- addrs[i]
		time.Sleep(5 * time.Millisecond)
	}

	// State should be STOPPING after first close
	assert.Equal(t, duck.STOPPING, s.getState(), "State should be STOPPING")

	// Close last connection
	closeChan <- addrs[numConnections-1]

	// Assert - context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled")
	}

	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Monitor should exit")
	}
}

// TestLDCConnectionMonitor_ContextCancellation_ExitGracefully tests that monitor exits on context cancellation
func TestLDCConnectionMonitor_ContextCancellation_ExitGracefully(t *testing.T) {
	// Setup
	s := createTestServer()
	openChan := make(chan net.Addr, 10)
	closeChan := make(chan net.Addr, 10)

	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	defer cancel()

	done := make(chan bool, 1)

	// Act - start monitor
	go func() {
		ldcConnectionMonitor(s, openChan, closeChan)
		done <- true
	}()

	// Cancel context
	cancel()

	// Assert - monitor should exit
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Monitor should exit when context is cancelled")
	}
}

// TestUpdateDataChannelGauge_UpdatesRegularly tests that data channel gauge is updated
func TestUpdateDataChannelGauge_UpdatesRegularly(t *testing.T) {
	// Setup
	s := createTestServer()
	dataChannel := make(chan LDCData, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	s.ctx = ctx
	defer cancel()

	done := make(chan bool, 1)

	// Act - start gauge updater
	go func() {
		updateDataChannelGauge(s, dataChannel)
		done <- true
	}()

	// Add some data to channel
	for i := 0; i < 5; i++ {
		dataChannel <- LDCData{
			EventID: i,
			ldcID:   1,
			Data:    []byte{byte(i)},
		}
	}

	// Wait for some updates
	time.Sleep(50 * time.Millisecond)

	// Assert - gauge should be updated (we can't easily test the exact value without importing prometheus client)
	// But we can verify the goroutine runs without panicking

	// Cleanup
	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Gauge updater should exit when context is cancelled")
	}
}

// TestUpdateDataChannelGauge_EmptyChannel tests gauge updater with empty channel
func TestUpdateDataChannelGauge_EmptyChannel(t *testing.T) {
	// Setup
	s := createTestServer()
	dataChannel := make(chan LDCData, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	s.ctx = ctx
	defer cancel()

	done := make(chan bool, 1)

	// Act - start gauge updater with empty channel
	go func() {
		updateDataChannelGauge(s, dataChannel)
		done <- true
	}()

	// Assert - should not panic and should exit on context cancellation
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Gauge updater should exit when context is cancelled")
	}
}

// TestAcceptTCPConnections_ContextCancellation_StopsAccepting tests that acceptTCPConnections stops on context cancellation
func TestAcceptTCPConnections_ContextCancellation_StopsAccepting(t *testing.T) {
	// Setup - create a TCP listener
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0, // Let OS assign port
	})
	require.NoError(t, err)
	defer listener.Close()

	s := createTestServer()
	dataChannel := make(chan LDCData, 10)

	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = context.WithValue(ctx, "tcpConnChBufferSize", 10)
	s.ctx = context.WithValue(ctx, "cancelCtx", cancel)

	done := make(chan bool, 1)

	// Act - start acceptTCPConnections in goroutine
	go func() {
		acceptTCPConnections(s, listener, dataChannel)
		done <- true
	}()

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context and close listener (listener must be closed to unblock Accept())
	cancel()
	listener.Close()

	// Assert - function should return
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("acceptTCPConnections should stop when context is cancelled and listener is closed")
	}
}

// TestLDCConnectionMonitor_ConcurrentAccess tests thread-safety of connection monitor
func TestLDCConnectionMonitor_ConcurrentAccess(t *testing.T) {
	// Setup
	s := createTestServer()
	s.setState(duck.RUNNING)

	openChan := make(chan net.Addr, 100)
	closeChan := make(chan net.Addr, 100)

	baseCtx, cancel := context.WithCancel(context.Background())
	s.ctx = context.WithValue(baseCtx, "cancelCtx", cancel)
	defer cancel()

	done := make(chan bool, 1)

	// Act - start monitor
	go func() {
		ldcConnectionMonitor(s, openChan, closeChan)
		done <- true
	}()

	// Send many open/close events concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				addr := &net.TCPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 6000 + id*10 + j,
				}
				openChan <- addr
				closeChan <- addr
			}
		}(i)
	}

	// Wait for all events to be sent
	wg.Wait()

	// Give monitor time to process
	time.Sleep(100 * time.Millisecond)

	// All connections should be closed, so context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected - context cancelled when all connections closed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Context should be cancelled when all connections are closed")
	}

	// Cleanup
	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Monitor should exit")
	}
}

// TestCountEnabledGDCs_SingleGDC tests counting with single GDC
func TestCountEnabledGDCs_SingleGDC(t *testing.T) {
	// Setup
	gdc := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	gdcs := []duck.GDCConfiguration{gdc}

	// Act
	count := countEnabledGDCs(gdcs)

	// Assert
	assert.Equal(t, 1, count, "Single enabled GDC should return 1")
}

// TestCountEnabledGDCs_LargeScale tests counting with many GDCs
func TestCountEnabledGDCs_LargeScale(t *testing.T) {
	// Setup
	numGDCs := 100
	gdcs := make([]duck.GDCConfiguration, numGDCs)

	for i := 0; i < numGDCs; i++ {
		gdcs[i] = testhelpers.NewTestGDCConfiguration(i+1, "gdc")
		// Disable every third GDC
		if i%3 == 0 {
			gdcs[i].Enabled = false
		}
	}

	// Act
	count := countEnabledGDCs(gdcs)

	// Assert - should be roughly 2/3 of 100
	// Indices 0, 3, 6, ..., 99 are disabled (34 values: 0 to 99 in steps of 3)
	// So enabled = 100 - 34 = 66
	expectedCount := 66
	assert.Equal(t, expectedCount, count)
}

// TestLDCConnectionMonitor_ReopenedConnection tests handling of reopened connections
func TestLDCConnectionMonitor_ReopenedConnection(t *testing.T) {
	// Setup
	s := createTestServer()
	s.setState(duck.RUNNING)

	openChan := make(chan net.Addr, 10)
	closeChan := make(chan net.Addr, 10)

	baseCtx, cancel := context.WithCancel(context.Background())
	s.ctx = context.WithValue(baseCtx, "cancelCtx", cancel)
	defer cancel()

	done := make(chan bool, 1)

	// Act - start monitor
	go func() {
		ldcConnectionMonitor(s, openChan, closeChan)
		done <- true
	}()

	// Open connection
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6006}
	openChan <- addr

	time.Sleep(10 * time.Millisecond)

	// Close connection
	closeChan <- addr

	time.Sleep(10 * time.Millisecond)

	// Reopen same address
	openChan <- addr

	time.Sleep(10 * time.Millisecond)

	// Close again - should trigger context cancel
	closeChan <- addr

	// Assert - context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled when all connections are closed")
	}

	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Monitor should exit")
	}
}

// TestUpdateDataChannelGauge_VariableChannelSize tests gauge with varying channel sizes
func TestUpdateDataChannelGauge_VariableChannelSize(t *testing.T) {
	// Setup
	s := createTestServer()
	dataChannel := make(chan LDCData, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	s.ctx = ctx
	defer cancel()

	done := make(chan bool, 1)

	// Act - start gauge updater
	go func() {
		updateDataChannelGauge(s, dataChannel)
		done <- true
	}()

	// Vary channel size
	for cycle := 0; cycle < 3; cycle++ {
		// Fill channel
		for i := 0; i < 10; i++ {
			dataChannel <- LDCData{EventID: i, ldcID: 1}
		}
		time.Sleep(20 * time.Millisecond)

		// Drain channel
		for i := 0; i < 10; i++ {
			select {
			case <-dataChannel:
			default:
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Assert - goroutine should still be running
	select {
	case <-done:
		t.Fatal("Gauge updater should still be running")
	default:
		// Expected - still running
	}

	// Cleanup
	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Gauge updater should exit when context is cancelled")
	}
}
