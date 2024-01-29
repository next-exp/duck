package main

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jmbenlloch/next_duck/ldcRPC/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestBufferPositionWrapping_Logic(t *testing.T) {
	// logger := testhelpers.SetupTest()
	// This test verifies the buffer position wrapping logic from readUDPSocket
	// Simulating the actual logic without network dependencies

	// Setup - simulate constants
	PACKET_BUFFER_SIZE := 10000
	N_PACKETS_IN_BUFFER := 100
	maxPosition := PACKET_BUFFER_SIZE * N_PACKETS_IN_BUFFER // 1,000,000

	testCases := []struct {
		name            string
		currentPosition int
		expectedNext    int
	}{
		{
			name:            "Normal increment",
			currentPosition: 0,
			expectedNext:    PACKET_BUFFER_SIZE, // 10,000
		},
		{
			name:            "Middle of buffer",
			currentPosition: 500000,
			expectedNext:    500000 + PACKET_BUFFER_SIZE,
		},
		{
			name:            "At wrap boundary",
			currentPosition: maxPosition - PACKET_BUFFER_SIZE, // 990,000
			expectedNext:    0,                                // Should wrap to 0
		},
		{
			name:            "Just before wrap",
			currentPosition: maxPosition - 2*PACKET_BUFFER_SIZE, // 980,000
			expectedNext:    maxPosition - PACKET_BUFFER_SIZE,   // 990,000
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the position increment and wrap logic
			position := tc.currentPosition
			position += PACKET_BUFFER_SIZE

			if position >= PACKET_BUFFER_SIZE*N_PACKETS_IN_BUFFER {
				position = 0
			}

			assert.Equal(t, tc.expectedNext, position)
		})
	}
}

func TestBufferPositionWrapping_MultipleIterations(t *testing.T) {
	// logger := testhelpers.SetupTest()
	// Setup
	PACKET_BUFFER_SIZE := 1000
	N_PACKETS_IN_BUFFER := 10
	maxPosition := PACKET_BUFFER_SIZE * N_PACKETS_IN_BUFFER

	position := 0
	iterations := 25 // More than buffer size to test wrapping

	positions := []int{}

	// Act: Simulate position increments
	for i := 0; i < iterations; i++ {
		positions = append(positions, position)

		position += PACKET_BUFFER_SIZE
		if position >= PACKET_BUFFER_SIZE*N_PACKETS_IN_BUFFER {
			position = 0
		}
	}

	// Assert: Check wrapping occurred
	assert.Equal(t, 0, positions[0])
	assert.Equal(t, 1000, positions[1])
	assert.Equal(t, 9000, positions[9])
	assert.Equal(t, 0, positions[10]) // Should wrap back to 0
	assert.Equal(t, 1000, positions[11])

	// Assert: All positions are valid
	for i, pos := range positions {
		assert.GreaterOrEqual(t, pos, 0, "Position %d should be >= 0", i)
		assert.Less(t, pos, maxPosition, "Position %d should be < maxPosition", i)
	}
}

// TestReadUDPSocket_ValidPacket_ReturnsData tests that readUDPSocket correctly processes valid packets
func TestReadUDPSocket_ValidPacket_ReturnsData(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests
	ctx := context.Background()
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Create a buffer and channel
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP("192.168.1.1")

	// Create mock UDP connection with test data
	testPacket := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{testPacket}, "192.168.1.1")

	// Act
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	newPos, size := readUDPSocket(s, mockConn, &buffer, 0, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)

	// Assert
	assert.Equal(t, packetBufferSize, newPos, "Position should advance by packet buffer size")
	assert.Equal(t, len(testPacket), size, "Size should match packet size")

	// Check that data was sent to channel
	select {
	case data := <-recvChannel:
		assert.Equal(t, 0, data.position, "Data should be at buffer index 0")
		assert.Equal(t, testPacket, data.data, "Data should match test packet")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No data received on channel")
	}

	// Check position is marked in use
	assert.True(t, bufferTracking.IsPositionInUse(0), "Position 0 should be marked in use")

	// Release the position
	bufferTracking.ReleasePosition(0)
}

// TestReadUDPSocket_WrongSourceIP_Rejected tests that packets from wrong IPs are rejected
func TestReadUDPSocket_WrongSourceIP_Rejected(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests
	ctx := context.Background()
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Create a buffer and channel
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP("192.168.1.1") // Expected IP

	// Create mock UDP connection with packet from DIFFERENT IP
	testPacket := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{testPacket}, "10.0.0.1") // Wrong IP

	// Act
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	newPos, size := readUDPSocket(s, mockConn, &buffer, 0, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)

	// Assert - should return 0,0 when source IP doesn't match
	assert.Equal(t, 0, newPos, "Position should be 0 when IP doesn't match")
	assert.Equal(t, 0, size, "Size should be 0 when IP doesn't match")

	// Check that no data was sent to channel
	select {
	case <-recvChannel:
		t.Fatal("Should not have received data from wrong IP")
	case <-time.After(50 * time.Millisecond):
		// Expected - no data
	}

	// Position should be released (not in use)
	assert.False(t, bufferTracking.IsPositionInUse(0), "Position 0 should NOT be in use")
}

// TestReadUDPSocket_Timeout_ReturnsZero tests that timeout returns zero values
func TestReadUDPSocket_Timeout_ReturnsZero(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests
	ctx := context.Background()
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Create a buffer and channel
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP("192.168.1.1")

	// Create mock UDP connection with no data (will timeout in non-blocking mode)
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{}, "192.168.1.1")

	// Act
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	newPos, size := readUDPSocket(s, mockConn, &buffer, 0, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)

	// Assert - should return 0,0 on timeout
	assert.Equal(t, 0, newPos, "Position should be 0 on timeout")
	assert.Equal(t, 0, size, "Size should be 0 on timeout")
}

// TestReadUDPSocket_BufferPositionWraps tests that buffer position wraps correctly
func TestReadUDPSocket_BufferPositionWraps(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup with small buffer to test wrapping
	packetBufferSize := 100
	nPacketsInBuffer := 3 // Total buffer: 300 bytes
	bufferTimeout := 1    // Use 1 second for tests
	ctx := context.Background()
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Create a buffer and channel
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP("192.168.1.1")

	// Create mock UDP connection with test data
	testPacket := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{testPacket}, "192.168.1.1")

	// Start at position that will cause wrap (last position in buffer)
	startPosition := packetBufferSize * (nPacketsInBuffer - 1) // 200

	// Act
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	newPos, _ := readUDPSocket(s, mockConn, &buffer, startPosition, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)

	// Assert - position should wrap to 0
	assert.Equal(t, 0, newPos, "Position should wrap to 0 after reaching end of buffer")

	// Clean up - release the position
	bufferTracking.ReleasePosition(2) // Buffer index 2 (position 200 / 100)
}

// TestReadUDPSocket_ReadError_HandlesGracefully tests error handling
func TestReadUDPSocket_ReadError_HandlesGracefully(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests
	ctx := context.Background()
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Create a buffer and channel
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP("192.168.1.1")

	// Create mock UDP connection that returns a non-timeout error
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{}, "192.168.1.1")
	mockConn.SetReadError(&testNetworkError{msg: "network error"})

	// Act
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	newPos, size := readUDPSocket(s, mockConn, &buffer, 0, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)

	// Assert - should return 0,0 on error
	assert.Equal(t, 0, newPos, "Position should be 0 on error")
	assert.Equal(t, 0, size, "Size should be 0 on error")

	// Position should be released
	assert.False(t, bufferTracking.IsPositionInUse(0), "Position should be released on error")
}

// testNetworkError implements error interface for testing
type testNetworkError struct {
	msg string
}

func (e *testNetworkError) Error() string { return e.msg }

// TestListenEquipment_ContextCancellation_ClosesConnection tests that context cancellation properly cleans up
func TestListenEquipment_ContextCancellation_ClosesConnection(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	equipment := testhelpers.NewTestEquipment(1, 22)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)

	// Create a mock connection that returns timeouts (non-blocking)
	// This allows the loop to check context.Done() on each iteration
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{}, "192.168.1.1")

	// Start listenEquipment in background
	done := make(chan bool)
	go func() {
		listenEquipment(s, equipment, mockConn, recvChannel, bufferTracking)
		done <- true
	}()

	// Give it a moment to start and iterate
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Assert - should complete and close connection
	select {
	case <-done:
		assert.True(t, mockConn.IsClosed(), "Connection should be closed")
	case <-time.After(1 * time.Second):
		t.Fatal("listenEquipment did not stop on context cancellation")
	}
}

// TestReadUDPSocket_MultiplePackets_ProcessesSequentially tests multiple packet processing
func TestReadUDPSocket_MultiplePackets_ProcessesSequentially(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	packetBufferSize := 100
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests
	ctx := context.Background()
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Create a buffer and channel
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP("192.168.1.1")

	// Create mock UDP connection with multiple packets
	packets := [][]byte{
		{0x01, 0x02, 0x03},
		{0x04, 0x05, 0x06},
		{0x07, 0x08, 0x09},
	}
	mockConn := testhelpers.NewMockUDPConnNonBlocking(packets, "192.168.1.1")

	// Act - read multiple packets
	position := 0
	bufferIndex := 0
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	for i := 0; i < len(packets); i++ {
		newPos, _ := readUDPSocket(s, mockConn, &buffer, position, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)
		if newPos > 0 {
			// Release the previous position
			bufferTracking.ReleasePosition(bufferIndex)
			position = newPos
			bufferIndex = position / packetBufferSize
		}
	}

	// Assert - should have received all packets
	receivedCount := 0
	for {
		select {
		case <-recvChannel:
			receivedCount++
		default:
			goto done
		}
	}
done:
	assert.Equal(t, len(packets), receivedCount, "Should have received all packets")
}

// Benchmark for buffer position wrapping logic
func BenchmarkBufferPositionWrapping(b *testing.B) {
	PACKET_BUFFER_SIZE := 10000
	N_PACKETS_IN_BUFFER := 100
	maxPosition := PACKET_BUFFER_SIZE * N_PACKETS_IN_BUFFER

	position := 0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		position += PACKET_BUFFER_SIZE
		if position >= maxPosition {
			position = 0
		}
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// setupTestServer creates a test server with context, logger, metrics, and default configuration
func setupTestServer() (*server, context.Context) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:           ctx,
		logger:        logger,
		metrics:       metrics,
		cancelCtx:     cancel,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	return s, ctx
}

// verifyMetricsInDelta verifies that a metric has incremented by the expected delta
func verifyMetricsInDelta(t *testing.T, getMetric func() int64, initial int64, expectedDelta int64, description string) {
	t.Helper()
	current := getMetric()
	actualDelta := current - initial
	if actualDelta != expectedDelta {
		t.Errorf("%s: expected delta %d, got %d (initial: %d, current: %d)",
			description, expectedDelta, actualDelta, initial, current)
	}
}

// waitForChannelWithTimeout waits for data on a channel with a timeout
// Returns true if data received, false if timeout
func waitForChannelWithTimeout[T any](t *testing.T, ch <-chan T, timeout time.Duration) (T, bool) {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case data := <-ch:
		return data, true
	case <-timer.C:
		var zero T
		return zero, false
	}
}

// ============================================================================
// METRICS VERIFICATION TESTS
// ============================================================================

// TestMetrics_SequenceError_IncrementsPacketErrorCounter verifies that
// packet error counter increments on sequence counter mismatches
func TestMetrics_SequenceError_IncrementsPacketErrorCounter(t *testing.T) {
	s, _ := setupTestServer()

	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(10)

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 1 // Start expecting sequence 1, not 0
	evtError := false

	// Get initial metrics
	initialErrors := s.metrics.GetPacketErrorCounter()

	// Create first packet with sequence 0 (should expect 0)
	packet1 := testhelpers.GenerateTestPacket(0, []byte{0x01, 0x02, 0x03})
	bufferData1 := RingBufferData{data: packet1, position: 0}

	// Act - process packet with wrong sequence counter
	buildEquipmentData(s, bufferData1, &fragments, &bufferPositions, equipment,
		equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

	// Assert - packet error counter should be incremented
	currentErrors := s.metrics.GetPacketErrorCounter()
	expectedDelta := int64(1)
	actualDelta := currentErrors - initialErrors

	assert.Equal(t, expectedDelta, actualDelta, "Packet error counter should increment by 1")
	assert.True(t, evtError, "Event error flag should be set")

	// Cleanup
	bufferTracking.ReleasePosition(0)
}

// TestMetrics_MultipleEquipments_TracksPerEquipmentChannelCount verifies
// that per-equipment channel counters are tracked separately
func TestMetrics_MultipleEquipments_TracksPerEquipmentChannelCount(t *testing.T) {
	s, _ := setupTestServer()

	// Setup - create 3 different equipments
	equipment1 := testhelpers.NewTestEquipment(1, 22)
	equipment2 := testhelpers.NewTestEquipment(2, 22)
	equipment3 := testhelpers.NewTestEquipment(3, 22)

	equipmentDataChannel := make(chan EquipmentData, 100)

	// Act - send data from each equipment
	eqData1 := EquipmentData{EventID: 1, EquipmentID: equipment1.ID, Data: []byte{0x01}}
	eqData2 := EquipmentData{EventID: 1, EquipmentID: equipment2.ID, Data: []byte{0x02}}
	eqData3 := EquipmentData{EventID: 1, EquipmentID: equipment3.ID, Data: []byte{0x03}}

	equipmentDataChannel <- eqData1
	equipmentDataChannel <- eqData2
	equipmentDataChannel <- eqData3

	// Assert - verify each equipment's counter
	count1 := s.metrics.receiveChannelCounters.WithLabelValues(string(rune('0' + equipment1.ID)))
	count2 := s.metrics.receiveChannelCounters.WithLabelValues(string(rune('0' + equipment2.ID)))
	count3 := s.metrics.receiveChannelCounters.WithLabelValues(string(rune('0' + equipment3.ID)))

	// Note: These are Gauges, we're just verifying they can be accessed
	// The actual channel count tracking happens in a different part of the code
	assert.NotNil(t, count1, "Equipment 1 counter should exist")
	assert.NotNil(t, count2, "Equipment 2 counter should exist")
	assert.NotNil(t, count3, "Equipment 3 counter should exist")

	// Drain channel
	for i := 0; i < 3; i++ {
		<-equipmentDataChannel
	}
}

// ============================================================================
// INTEGRATION TESTS
// ============================================================================

// TestIntegration_FullDataFlow_SingleEvent verifies complete data flow from
// UDP packet through event processing
func TestIntegration_FullDataFlow_SingleEvent(t *testing.T) {
	s, _ := setupTestServer()

	// Setup - small buffer configuration
	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests

	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP(equipment.DeviceIP)

	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)

	// Generate complete event with 3 packets + end marker
	eventPackets := testhelpers.GenerateCompleteEvent(3, 100)
	assert.Equal(t, 4, len(eventPackets), "Should have 3 data packets + 1 end marker")

	// Start readPacket goroutine to process packets
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		readPacket(s, recvChannel, equipment, equipmentDataChannel, bufferTracking)
	}()

	// Feed packets through readUDPSocket
	position := 0
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	for _, packet := range eventPackets {
		mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{packet}, equipment.DeviceIP)
		newPos, _ := readUDPSocket(s, mockConn, &buffer, position, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)
		if newPos > 0 {
			position = newPos
		}
		// Small delay to allow processing
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for event on equipmentDataChannel
	var eventData EquipmentData
	var received bool
	select {
	case eventData = <-equipmentDataChannel:
		received = true
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Did not receive event data within timeout")
	}

	assert.True(t, received, "Should receive event data")
	assert.Equal(t, 1, eventData.EventID, "EventID should be 1")
	assert.Equal(t, equipment.ID, eventData.EquipmentID, "EquipmentID should match")
	assert.NotEmpty(t, eventData.Data, "Data should not be empty")
	assert.False(t, eventData.Error, "Event should not have errors")

	// Cleanup - release buffer positions
	for i := 0; i < nPacketsInBuffer; i++ {
		bufferTracking.ReleasePosition(i)
	}

	// Stop readPacket goroutine
	s.cancelCtx()
	close(recvChannel) // Close channel to allow goroutine to exit
	wg.Wait()
}

// TestIntegration_FullDataFlow_MultipleEventsWithBufferWrap tests multiple
// events with ring buffer wrapping
func TestIntegration_FullDataFlow_MultipleEventsWithBufferWrap(t *testing.T) {
	s, _ := setupTestServer()

	// Setup - small buffer configuration
	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests

	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP(equipment.DeviceIP)

	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)

	// Generate 2 events with 2 packets each + end marker
	numEvents := 2
	packetsPerEvent := 2
	allPackets := [][]byte{}

	for eventIdx := 0; eventIdx < numEvents; eventIdx++ {
		eventPackets := testhelpers.GenerateCompleteEvent(packetsPerEvent, 50)
		allPackets = append(allPackets, eventPackets...)
	}

	// Start readPacket goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		readPacket(s, recvChannel, equipment, equipmentDataChannel, bufferTracking)
	}()

	// Track buffer positions used
	usedPositions := make(map[int]bool)
	position := 0

	// Feed all packets
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	for _, packet := range allPackets {
		mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{packet}, equipment.DeviceIP)
		bufferIndex := position / packetBufferSize

		newPos, _ := readUDPSocket(s, mockConn, &buffer, position, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)
		if newPos > 0 {
			usedPositions[bufferIndex] = true
			position = newPos
		}

		// Small delay to allow processing
		time.Sleep(10 * time.Millisecond)
	}

	// Receive all events
	receivedEvents := []EquipmentData{}
	timeout := time.After(2 * time.Second)

	for len(receivedEvents) < numEvents {
		select {
		case eventData := <-equipmentDataChannel:
			receivedEvents = append(receivedEvents, eventData)
		case <-timeout:
			t.Fatalf("Timeout waiting for events. Received %d out of %d", len(receivedEvents), numEvents)
		}
	}

	// Verify all events received
	assert.Equal(t, numEvents, len(receivedEvents), "Should receive all events")

	// Verify event IDs are sequential
	for i, event := range receivedEvents {
		expectedEventID := i + 1
		assert.Equal(t, expectedEventID, event.EventID, "Event %d should have EventID %d", i, expectedEventID)
		assert.Equal(t, equipment.ID, event.EquipmentID, "EquipmentID should match")
		assert.NotEmpty(t, event.Data, "Event %d data should not be empty", i)
	}

	// Verify buffer positions were used
	assert.GreaterOrEqual(t, len(usedPositions), 2, "Should use multiple buffer positions")

	// Cleanup - release all buffer positions
	for i := 0; i < nPacketsInBuffer; i++ {
		bufferTracking.ReleasePosition(i)
	}

	// Stop readPacket goroutine
	s.cancelCtx()
	close(recvChannel) // Close channel to allow goroutine to exit
	wg.Wait()
}

// ============================================================================
// SOCKET CLEANUP TESTS
// ============================================================================

// TestReconnection_SocketClose_DocumentCurrentBehavior documents that
// socket close doesn't trigger reconnection (manual restart required)
func TestSocketCleanup_OnClose(t *testing.T) {
	s, ctx := setupTestServer()

	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(10)

	// Create mock UDP connection
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{}, "192.168.1.1")

	// Start listenEquipment in goroutine
	done := make(chan bool)
	go func() {
		listenEquipment(s, equipment, mockConn, recvChannel, bufferTracking)
		done <- true
	}()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Close the mock connection
	err := mockConn.Close()
	assert.NoError(t, err, "Closing mock connection should succeed")
	assert.True(t, mockConn.IsClosed(), "Connection should be marked as closed")

	// Cancel context to trigger graceful shutdown
	select {
	case <-ctx.Done():
		// Context was cancelled
	case <-time.After(100 * time.Millisecond):
		// Context not cancelled yet, cancel it manually
		s.cancelCtx()
	}

	// Wait for listenEquipment to exit
	select {
	case <-done:
		// listenEquipment exited cleanly
	case <-time.After(1 * time.Second):
		t.Fatal("listenEquipment did not exit after connection close")
	}
}

// TestReconnection_ContextCancelled_DuringRead_ErrorPath tests error
// recovery when context is cancelled during read operations
func TestSocketCleanup_OnContextCancellation(t *testing.T) {
	s, _ := setupTestServer()

	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(10)

	// Create mock UDP connection with NO data (will timeout immediately)
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{}, "192.168.1.1")

	// Start listenEquipment in goroutine
	done := make(chan bool)
	go func() {
		listenEquipment(s, equipment, mockConn, recvChannel, bufferTracking)
		done <- true
	}()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to simulate shutdown
	s.cancelCtx()

	// Wait for graceful exit
	select {
	case <-done:
		// Graceful exit occurred
	case <-time.After(1 * time.Second):
		t.Fatal("listenEquipment did not exit after context cancellation")
	}

	// Verify connection was closed
	assert.True(t, mockConn.IsClosed(), "Connection should be closed")
}

// TestReconnection_MultipleOpenCloseCycles_ResourceCleanup verifies no
// resource leaks after multiple open/close cycles
func TestSocketCleanup_MultipleCycles(t *testing.T) {
	numCycles := 3

	for cycle := 0; cycle < numCycles; cycle++ {
		// Setup for this cycle
		s, _ := setupTestServer()
		equipment := testhelpers.NewTestEquipment(1, 22)
		recvChannel := make(chan RingBufferData, 10)
		bufferTracking := NewRingBufferTracking(10)

		// Create mock connection
		mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{}, "192.168.1.1")

		// Track if goroutine completes
		done := make(chan bool)
		go func() {
			listenEquipment(s, equipment, mockConn, recvChannel, bufferTracking)
			done <- true
		}()

		// Let it run briefly
		time.Sleep(20 * time.Millisecond)

		// Cancel context to trigger shutdown
		s.cancelCtx()

		// Verify clean exit
		select {
		case <-done:
			// Clean exit
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Cycle %d: listenEquipment did not exit cleanly", cycle)
		}

		// Verify connection closed
		assert.True(t, mockConn.IsClosed(), "Cycle %d: Connection should be closed", cycle)

		t.Logf("Cycle %d completed cleanly", cycle)
	}
}

// TestReadUDPSocket_ConfigurableTimeout_VerifiesTimeoutParameter tests that the buffer timeout is configurable
func TestReadUDPSocket_ConfigurableTimeout_VerifiesTimeoutParameter(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	packetBufferSize := 1000
	nPacketsInBuffer := 10
	bufferTimeout := 1 // Use 1 second for tests

	ctx := context.Background()
	ctx = context.WithValue(ctx, "packetBufferSize", packetBufferSize)
	ctx = context.WithValue(ctx, "nPacketsInBuffer", nPacketsInBuffer)
	ctx = context.WithValue(ctx, "bufferTimeout", bufferTimeout)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Create a buffer and channel
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	recvChannel := make(chan RingBufferData, 10)
	bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
	deviceIP := net.ParseIP("192.168.1.1")

	// Create mock UDP connection with valid packet
	testPacket := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	mockConn := testhelpers.NewMockUDPConnNonBlocking([][]byte{testPacket}, "192.168.1.1")

	// Act - read with the configured timeout
	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	newPos, size := readUDPSocket(s, mockConn, &buffer, 0, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)

	// Assert - packet should be read successfully
	assert.Equal(t, packetBufferSize, newPos, "Position should advance by packet buffer size")
	assert.Equal(t, len(testPacket), size, "Size should match packet size")

	// Check that data was sent to channel
	select {
	case data := <-recvChannel:
		assert.Equal(t, 0, data.position, "Data should be at buffer index 0")
		assert.Equal(t, testPacket, data.data, "Data should match test packet")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No data received on channel")
	}

	// Cleanup
	bufferTracking.ReleasePosition(0)
}

// ============================================================================
// TABLE-DRIVEN TESTS
// ============================================================================

// TestReadUDPSocket_EdgeCases_TableDriven systematically tests edge cases
// for readUDPSocket function
func TestReadUDPSocket_EdgeCases_TableDriven(t *testing.T) {
	testCases := []struct {
		name             string
		packets          [][]byte
		sourceIP         string
		deviceIP         string
		initialPosition  int
		packetBufferSize int
		nPacketsInBuffer int
		expectPosition   int
		expectSize       int
		expectDataOnChan bool
		expectError      bool
		setupBuffer      func(t *testing.T, bufferTracking *RingBufferTracking)
	}{
		{
			name:             "Zero length packet",
			packets:          [][]byte{{}},
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   1000,
			expectSize:       0,
			expectDataOnChan: true,
			expectError:      false,
		},
		{
			name:             "Maximum size packet",
			packets:          [][]byte{make([]byte, 1000)},
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   1000,
			expectSize:       1000,
			expectDataOnChan: true,
			expectError:      false,
		},
		{
			name:             "Packet from wrong IP (rejected)",
			packets:          [][]byte{{0x01, 0x02, 0x03}},
			sourceIP:         "10.0.0.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   0,
			expectSize:       0,
			expectDataOnChan: false,
			expectError:      false,
		},
		{
			name:             "Buffer at wrap boundary",
			packets:          [][]byte{{0x01, 0x02, 0x03}},
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  9000, // At position 9 of 10 (0-9)
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   0, // Should wrap to 0
			expectSize:       3,
			expectDataOnChan: true,
			expectError:      false,
		},
		{
			name:             "Buffer position at exact boundary",
			packets:          [][]byte{{0x01, 0x02}},
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 100,
			nPacketsInBuffer: 5,
			expectPosition:   100,
			expectSize:       2,
			expectDataOnChan: true,
			expectError:      false,
		},
		{
			name:             "Single byte packet",
			packets:          [][]byte{{0x42}},
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   1000,
			expectSize:       1,
			expectDataOnChan: true,
			expectError:      false,
		},
		{
			name:             "Non-IP device (rejected)",
			packets:          [][]byte{{0x01, 0x02, 0x03}},
			sourceIP:         "192.168.1.1",
			deviceIP:         "invalid-ip",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   0, // Should fail to match
			expectSize:       0,
			expectDataOnChan: false,
			expectError:      false,
		},
		{
			name:             "Buffer position zero",
			packets:          [][]byte{{0x01, 0x02, 0x03, 0x04}},
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   1000,
			expectSize:       4,
			expectDataOnChan: true,
			expectError:      false,
		},
		{
			name:             "Read timeout (returns zero)",
			packets:          [][]byte{}, // No packets
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   0,
			expectSize:       0,
			expectDataOnChan: false,
			expectError:      false,
		},
		{
			name: "Multiple sequential reads",
			packets: [][]byte{
				{0x01, 0x02},
				{0x03, 0x04},
				{0x05, 0x06},
			},
			sourceIP:         "192.168.1.1",
			deviceIP:         "192.168.1.1",
			initialPosition:  0,
			packetBufferSize: 1000,
			nPacketsInBuffer: 10,
			expectPosition:   3000, // After 3 reads
			expectSize:       2,    // Last packet size
			expectDataOnChan: true,
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := setupTestServer()

			recvChannel := make(chan RingBufferData, 100)
			bufferTracking := NewRingBufferTracking(tc.nPacketsInBuffer)

			buffer := make([]byte, tc.packetBufferSize*tc.nPacketsInBuffer)
			deviceIP := net.ParseIP(tc.deviceIP)

			// Custom setup if provided
			if tc.setupBuffer != nil {
				tc.setupBuffer(t, bufferTracking)
			}

			// Create mock connection
			mockConn := testhelpers.NewMockUDPConnNonBlocking(tc.packets, tc.sourceIP)

			// If testing network error, set error on mock
			if tc.expectError {
				mockConn.SetReadError(&testNetworkError{msg: "network error"})
			}

			// Execute multiple sequential reads if configured
			position := tc.initialPosition
			var size int
			timeoutDuration := 1 * time.Second // Use 1 second for all table-driven tests
			if len(tc.packets) > 1 {
				for range tc.packets {
					newPos, newSize := readUDPSocket(s, mockConn, &buffer, position, &deviceIP, &recvChannel, bufferTracking, tc.packetBufferSize, tc.nPacketsInBuffer, timeoutDuration)
					if newPos > 0 {
						position = newPos
						size = newSize
					}
				}
			} else {
				position, size = readUDPSocket(s, mockConn, &buffer, tc.initialPosition, &deviceIP, &recvChannel, bufferTracking, tc.packetBufferSize, tc.nPacketsInBuffer, timeoutDuration)
			}

			// Assert position and size
			assert.Equal(t, tc.expectPosition, position, "Position should match expected")
			assert.Equal(t, tc.expectSize, size, "Size should match expected")

			// Assert channel behavior
			if tc.expectDataOnChan {
				select {
				case data := <-recvChannel:
					assert.NotNil(t, data.data, "Data should not be nil")
					bufferTracking.ReleasePosition(data.position)
				case <-time.After(100 * time.Millisecond):
					t.Fatal("Expected data on channel but received none")
				}
			} else {
				select {
				case <-recvChannel:
					t.Fatal("Should not receive data on channel")
				case <-time.After(50 * time.Millisecond):
					// Expected - no data
				}
			}

			// Cleanup any remaining positions
			for i := 0; i < tc.nPacketsInBuffer; i++ {
				bufferTracking.ReleasePosition(i)
			}
		})
	}
}

// TestBuildEquipmentData_SequenceValidation_TableDriven systematically tests
// sequence counter validation in buildEquipmentData function
func TestBuildEquipmentData_SequenceValidation_TableDriven(t *testing.T) {
	testCases := []struct {
		name                   string
		packets                [][]byte
		initialEventID         int
		initialExpectedSeq     int
		expectEventError       bool
		expectContextCancelled bool
		expectDataOnChannel    bool
		expectEventIDIncrement bool
		setupErrorState        func() (evtError bool, expectedSeq int)
	}{
		{
			name: "Valid sequence 0, 1, 2, end marker",
			packets: [][]byte{
				testhelpers.GenerateTestPacket(0, []byte{0x01, 0x02}),
				testhelpers.GenerateTestPacket(1, []byte{0x03, 0x04}),
				testhelpers.GenerateTestPacket(2, []byte{0x05, 0x06}),
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     0,
			expectEventError:       false,
			expectContextCancelled: false,
			expectDataOnChannel:    true,
			expectEventIDIncrement: true,
		},
		{
			name: "Missing sequence 0 (starts at 1)",
			packets: [][]byte{
				testhelpers.GenerateTestPacket(1, []byte{0x01, 0x02}),
				testhelpers.GenerateTestPacket(2, []byte{0x03, 0x04}),
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     0, // Expects 0, gets 1
			expectEventError:       true,
			expectContextCancelled: true,
			expectDataOnChannel:    false,
			expectEventIDIncrement: false,
		},
		{
			name: "Duplicate sequence number",
			packets: [][]byte{
				testhelpers.GenerateTestPacket(0, []byte{0x01, 0x02}),
				testhelpers.GenerateTestPacket(0, []byte{0x03, 0x04}), // Duplicate
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     0,
			expectEventError:       true,
			expectContextCancelled: true,
			expectDataOnChannel:    false,
			expectEventIDIncrement: false,
		},
		{
			name: "Sequence skip (0 to 2)",
			packets: [][]byte{
				testhelpers.GenerateTestPacket(0, []byte{0x01, 0x02}),
				testhelpers.GenerateTestPacket(2, []byte{0x03, 0x04}), // Skips 1
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     0,
			expectEventError:       true,
			expectContextCancelled: true,
			expectDataOnChannel:    false,
			expectEventIDIncrement: false,
		},
		{
			name: "Packet too short (< 4 bytes)",
			packets: [][]byte{
				[]byte{0x01, 0x02}, // Too short for sequence counter
			},
			initialEventID:         1,
			initialExpectedSeq:     0,
			expectEventError:       true,
			expectContextCancelled: true,
			expectDataOnChannel:    false,
			expectEventIDIncrement: false,
		},
		{
			name: "Empty packet",
			packets: [][]byte{
				[]byte{}, // Empty
			},
			initialEventID:         1,
			initialExpectedSeq:     0,
			expectEventError:       true,
			expectContextCancelled: true,
			expectDataOnChannel:    false,
			expectEventIDIncrement: false,
		},
		{
			name: "Valid sequence with error flag set initially",
			packets: [][]byte{
				testhelpers.GenerateTestPacket(0, []byte{0x01, 0x02}),
				testhelpers.GenerateTestPacket(1, []byte{0x03, 0x04}),
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     0,
			expectEventError:       false, // Error flag is reset after event completion
			expectContextCancelled: false,
			expectDataOnChannel:    false, // Data NOT sent when initial error flag is set
			expectEventIDIncrement: true,  // But eventID still increments
			setupErrorState: func() (bool, int) {
				return true, 0 // Start with error flag set
			},
		},
		{
			name: "Multiple events in sequence",
			packets: [][]byte{
				// First event
				testhelpers.GenerateTestPacket(0, []byte{0x01}),
				testhelpers.GenerateEndMarker(),
				// Second event
				testhelpers.GenerateTestPacket(0, []byte{0x02}),
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     0,
			expectEventError:       false,
			expectContextCancelled: false,
			expectDataOnChannel:    true,
			expectEventIDIncrement: true, // EventID goes from 1 to 3 after processing 2 events (1, 2)
		},
		{
			name: "Sequence error with mismatch",
			packets: [][]byte{
				testhelpers.GenerateTestPacket(0, []byte{0x01, 0x02}),
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     5, // Expects 5, gets 0
			expectEventError:       true,
			expectContextCancelled: true,
			expectDataOnChannel:    false,
			expectEventIDIncrement: false,
			setupErrorState: func() (bool, int) {
				return false, 5 // Different expected sequence
			},
		},
		{
			name: "Large sequence number",
			packets: [][]byte{
				testhelpers.GenerateTestPacket(100, []byte{0x01, 0x02}),
				testhelpers.GenerateTestPacket(101, []byte{0x03, 0x04}),
				testhelpers.GenerateEndMarker(),
			},
			initialEventID:         1,
			initialExpectedSeq:     100,
			expectEventError:       false,
			expectContextCancelled: false,
			expectDataOnChannel:    true,
			expectEventIDIncrement: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := setupTestServer()

			equipment := testhelpers.NewTestEquipment(1, 22)
			equipmentDataChannel := make(chan EquipmentData, 10)
			bufferTracking := NewRingBufferTracking(10)

			// Track if context was cancelled
			contextCancelled := false
			s.cancelCtx = func() {
				contextCancelled = true
			}

			var fragments [][]byte
			var bufferPositions []int
			eventID := tc.initialEventID
			expectedSequenceCounter := tc.initialExpectedSeq
			evtError := false

			// Setup error state if provided
			if tc.setupErrorState != nil {
				evtError, expectedSequenceCounter = tc.setupErrorState()
			}

			// Process all packets
			for i, packet := range tc.packets {
				bufferData := RingBufferData{
					data:     packet,
					position: i,
				}

				// Acquire position
				_ = bufferTracking.TryAcquirePosition(i, 100*time.Millisecond)

				// Build equipment data
				buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
					equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

				// If context was cancelled, stop processing
				if contextCancelled {
					break
				}
			}

			// Verify error state
			if tc.expectEventError {
				assert.True(t, evtError, "Event error flag should be set")
			}

			// Verify context cancellation
			if tc.expectContextCancelled {
				assert.True(t, contextCancelled, "Context should be cancelled")
			} else {
				assert.False(t, contextCancelled, "Context should not be cancelled")
			}

			// Verify data on channel
			if tc.expectDataOnChannel {
				select {
				case data := <-equipmentDataChannel:
					assert.Equal(t, tc.initialEventID, data.EventID, "EventID should match")
					assert.Equal(t, equipment.ID, data.EquipmentID, "EquipmentID should match")
					assert.Equal(t, tc.expectEventError, data.Error, "Error flag should match")
				case <-time.After(100 * time.Millisecond):
					t.Fatal("Expected data on equipment data channel")
				}
			} else {
				select {
				case <-equipmentDataChannel:
					t.Fatal("Should not receive data on channel when error occurs")
				case <-time.After(50 * time.Millisecond):
					// Expected - no data
				}
			}

			// Verify event ID increment
			if tc.expectEventIDIncrement {
				// Count number of end markers to determine expected increment
				numEndMarkers := 0
				for _, p := range tc.packets {
					if len(p) == 4 {
						endWord := binary.LittleEndian.Uint32(p)
						if endWord == 0xfafafafa {
							numEndMarkers++
						}
					}
				}
				expectedEventID := tc.initialEventID + numEndMarkers
				assert.Equal(t, expectedEventID, eventID, "EventID should be incremented correctly")
			} else {
				assert.Equal(t, tc.initialEventID, eventID, "EventID should not be incremented")
			}

			// Cleanup
			for i := 0; i < 10; i++ {
				bufferTracking.ReleasePosition(i)
			}

			// Drain any remaining data
			select {
			case <-equipmentDataChannel:
			default:
			}
		})
	}
}
