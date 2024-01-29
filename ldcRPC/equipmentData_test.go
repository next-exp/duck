package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/ldcRPC/testhelpers"
)

func TestBuildEquipmentData_ValidSequence_AssemblesCorrectly(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx := context.Background()
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtError := false

	// Generate test packets
	packets := testhelpers.GenerateCompleteEvent(3, 100)

	// Act: Process all packets
	for i, packet := range packets {
		bufferData := RingBufferData{data: packet, position: i}
		// Mark position as in use (simulating what the writer does)
		bufferTracking.TryAcquirePosition(i, time.Second)
		bufferTracking.MarkPositionInUse(i)
		buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
			equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)
	}

	// Assert: Event should be sent to channel
	require.Len(t, equipmentDataChannel, 1)
	eqData := <-equipmentDataChannel

	assert.Equal(t, 1, eqData.EventID)
	assert.Equal(t, equipment.ID, eqData.EquipmentID)
	assert.False(t, eqData.Error)
	assert.NotEmpty(t, eqData.Data)

	// Assert: Fragments should be cleared
	assert.Empty(t, fragments)
	assert.Empty(t, bufferPositions)

	// Assert: Event ID should increment
	assert.Equal(t, 2, eventID)

	// Assert: Buffer positions should be released
	for i := 0; i < len(packets); i++ {
		assert.False(t, bufferTracking.IsPositionInUse(i), "Position %d should be released", i)
	}
}

func TestBuildEquipmentData_EndMarker_TriggersEventCompletion(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx := context.Background()
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtError := false

	// Act: Send a data packet first
	dataPacket := testhelpers.GenerateTestPacket(0, []byte{0x01, 0x02, 0x03})
	bufferTracking.TryAcquirePosition(0, time.Second)
	bufferTracking.MarkPositionInUse(0)
	bufferData := RingBufferData{data: dataPacket, position: 0}
	buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
		equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

	// No event should be sent yet
	assert.Empty(t, equipmentDataChannel)
	assert.Len(t, fragments, 1)

	// Act: Send end marker
	endMarker := testhelpers.GenerateEndMarker()
	bufferTracking.TryAcquirePosition(1, time.Second)
	bufferTracking.MarkPositionInUse(1)
	bufferData = RingBufferData{data: endMarker, position: 1}
	buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
		equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

	// Assert: Event should now be sent
	require.Len(t, equipmentDataChannel, 1)
	eqData := <-equipmentDataChannel

	assert.Equal(t, 1, eqData.EventID)
	assert.False(t, eqData.Error)

	// Fragments should be cleared
	assert.Empty(t, fragments)

	// Buffer positions should be released
	assert.False(t, bufferTracking.IsPositionInUse(0), "Position 0 should be released")
	assert.False(t, bufferTracking.IsPositionInUse(1), "Position 1 should be released")
}

func TestBuildEquipmentData_SequenceCounterError_CancelsContext(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ctx:           ctx,
		cancelCtx:     cancel,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtError := false

	// Act: Send packets with wrong sequence counter
	packet1 := testhelpers.GenerateTestPacket(1, []byte{0x01})
	bufferTracking.TryAcquirePosition(0, time.Second)
	bufferTracking.MarkPositionInUse(0)
	bufferData1 := RingBufferData{data: packet1, position: 0}
	buildEquipmentData(s, bufferData1, &fragments, &bufferPositions, equipment,
		equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

	// Now send packet 3 instead of packet 2
	packet3 := testhelpers.GenerateTestPacket(3, []byte{0x03})
	bufferTracking.TryAcquirePosition(1, time.Second)
	bufferTracking.MarkPositionInUse(1)
	bufferData3 := RingBufferData{data: packet3, position: 1}
	buildEquipmentData(s, bufferData3, &fragments, &bufferPositions, equipment,
		equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

	// Assert: Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected - context was cancelled due to sequence error
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context was not cancelled after sequence counter error")
	}

	// Assert: Error flag should be set
	assert.True(t, evtError)
}

func TestBuildEquipmentData_ErrorFlag_PreventsEventSending(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx := context.Background()
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtError := true // Start with error flag set

	// Act: Send a complete event with error flag set
	packets := testhelpers.GenerateCompleteEvent(3, 100)
	for i, packet := range packets {
		bufferTracking.TryAcquirePosition(i, time.Second)
		bufferTracking.MarkPositionInUse(i)
		bufferData := RingBufferData{data: packet, position: i}
		buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
			equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)
	}

	// Assert: Event should NOT be sent to channel
	assert.Empty(t, equipmentDataChannel)
}

func TestBuildEquipmentData_MultipleEvents_SequentialProcessing(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 100)
	bufferTracking := NewRingBufferTracking(1000)

	ctx := context.Background()
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtError := false

	// Act: Process multiple complete events
	numEvents := 5
	positionCounter := 0
	for eventNum := 0; eventNum < numEvents; eventNum++ {
		packets := testhelpers.GenerateCompleteEvent(3, 100)
		for _, packet := range packets {
			bufferTracking.TryAcquirePosition(positionCounter, time.Second)
			bufferTracking.MarkPositionInUse(positionCounter)
			bufferData := RingBufferData{data: packet, position: positionCounter}
			buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
				equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)
			positionCounter++
		}
	}

	// Assert: Should have received all events
	assert.Equal(t, numEvents, len(equipmentDataChannel))

	// Verify event IDs are sequential
	receivedEventIDs := make([]int, numEvents)
	for i := 0; i < numEvents; i++ {
		eqData := <-equipmentDataChannel
		receivedEventIDs[i] = eqData.EventID
	}

	assert.Equal(t, []int{1, 2, 3, 4, 5}, receivedEventIDs)
}

func TestBuildEquipmentData_ShortPacket_HandledGracefully(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ctx:           ctx,
		cancelCtx:     cancel,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	testCases := []struct {
		name        string
		packet      []byte
		expectError bool
		expectPanic bool
	}{
		{"empty_packet", []byte{}, true, false},                                         // 0 bytes
		{"1_byte_packet", []byte{0x01}, true, false},                                    // 1 byte
		{"2_byte_packet", []byte{0x01, 0x02}, true, false},                              // 2 bytes
		{"3_byte_packet", []byte{0x01, 0x02, 0x03}, true, false},                        // 3 bytes
		{"4_byte_valid", testhelpers.GenerateTestPacket(0, []byte{0x01}), false, false}, // Valid data packet
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var fragments [][]byte
			var bufferPositions []int
			eventID := 1
			expectedSequenceCounter := 0
			evtError := false

			initialErrorCount := metrics.GetPacketErrorCounter()

			bufferData := RingBufferData{data: tc.packet, position: 0}
			bufferTracking.TryAcquirePosition(0, time.Second)
			bufferTracking.MarkPositionInUse(0)
			buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
				equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

			if tc.expectError {
				// Assert: Error flag should be set
				assert.True(t, evtError, "Error flag should be set for short packet")
				// Assert: Metrics should increment
				finalErrorCount := metrics.GetPacketErrorCounter()
				assert.Equal(t, initialErrorCount+1, finalErrorCount,
					"Packet error counter should increment")
				// Assert: Context should be cancelled
				assert.True(t, ctx.Err() != nil, "Context should be cancelled")
			} else {
				// Assert: No error for valid packets
				assert.False(t, evtError, "Error flag should not be set for valid packet")
			}
		})
	}
}

func TestBuildEquipmentData_SequenceCounterError_IncrementsMetrics(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ctx:           ctx,
		cancelCtx:     cancel,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtError := false

	// Get initial metrics value
	initialErrorCount := metrics.GetPacketErrorCounter()

	// Act: Send packets with wrong sequence counter
	packet1 := testhelpers.GenerateTestPacket(1, []byte{0x01})
	bufferTracking.TryAcquirePosition(0, time.Second)
	bufferTracking.MarkPositionInUse(0)
	bufferData1 := RingBufferData{data: packet1, position: 0}
	buildEquipmentData(s, bufferData1, &fragments, &bufferPositions, equipment,
		equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

	// Send another packet with wrong sequence (skip 2)
	packet3 := testhelpers.GenerateTestPacket(3, []byte{0x03})
	bufferTracking.TryAcquirePosition(1, time.Second)
	bufferTracking.MarkPositionInUse(1)
	bufferData3 := RingBufferData{data: packet3, position: 1}
	buildEquipmentData(s, bufferData3, &fragments, &bufferPositions, equipment,
		equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)

	// Assert: Error counter should increment by 2 (one for each error)
	finalErrorCount := metrics.GetPacketErrorCounter()
	assert.Equal(t, initialErrorCount+2, finalErrorCount,
		"Packet error counter should increment by 2")
	assert.Equal(t, int64(2), finalErrorCount, "Total error count should be 2")

	// Assert: Other state should also be updated
	assert.True(t, evtError, "Error flag should be set")
	assert.True(t, ctx.Err() != nil, "Context should be cancelled")
}

func TestBuildEquipmentData_NonWordAlignedData_LogsError(t *testing.T) {
	logger, logCapture := testhelpers.SetupTestWithCapture()
	metrics := NewMetricsRegistry()
	equipment := testhelpers.NewTestEquipment(1, 22)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx := context.Background()
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtError := false

	// Create packets with non-word-aligned payload sizes
	// Payload size: 103 bytes = 102 (25.5 words) + 1 byte remainder
	// This ensures non-word-aligned data
	packets := testhelpers.GenerateCompleteEvent(3, 103)

	// Process all packets
	for i, packet := range packets {
		bufferTracking.TryAcquirePosition(i, time.Second)
		bufferTracking.MarkPositionInUse(i)
		bufferData := RingBufferData{data: packet, position: i}
		buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
			equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)
	}

	// Assert: Event should still be sent
	require.Len(t, equipmentDataChannel, 1)
	eqData := <-equipmentDataChannel
	assert.Equal(t, 1, eqData.EventID)
	assert.False(t, eqData.Error)
	assert.NotEmpty(t, eqData.Data)

	// Assert: Log should contain non-word-aligned error message
	logOutput := logCapture.String()
	assert.Contains(t, logOutput, "Non-word-aligned equipment data",
		"Log should contain non-word-aligned error message")
	assert.Contains(t, logOutput, "remainder_bytes",
		"Log should specify the remainder bytes")

	// Verify the data integrity - total size should account for header + non-aligned data
	expectedMinSize := 103 + 28 // 103 bytes payload + 28 bytes header
	assert.GreaterOrEqual(t, len(eqData.Data), expectedMinSize,
		"Equipment data should contain at least payload + header size")
}

func TestBuildEquipmentData_VariousRemainderSizes(t *testing.T) {
	testCases := []struct {
		name        string
		payloadSize int
		expectedRem int
	}{
		{"aligned_4_bytes", 100, 0},   // 100 % 4 = 0
		{"remainder_1_byte", 101, 1},  // 101 % 4 = 1
		{"remainder_2_bytes", 102, 2}, // 102 % 4 = 2
		{"remainder_3_bytes", 103, 3}, // 103 % 4 = 3
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := testhelpers.SetupTest()
			metrics := NewMetricsRegistry()
			equipment := testhelpers.NewTestEquipment(1, 22)
			equipmentDataChannel := make(chan EquipmentData, 10)
			bufferTracking := NewRingBufferTracking(100)

			ctx := context.Background()
			s := &server{
				ctx:     ctx,
				logger:  logger,
				metrics: metrics,
			}

			var fragments [][]byte
			var bufferPositions []int
			eventID := 1
			expectedSequenceCounter := 0
			evtError := false

			// Generate complete event with specific payload size
			packets := testhelpers.GenerateCompleteEvent(2, tc.payloadSize)

			for i, packet := range packets {
				bufferTracking.TryAcquirePosition(i, time.Second)
				bufferTracking.MarkPositionInUse(i)
				bufferData := RingBufferData{data: packet, position: i}
				buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
					equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)
			}

			// Event should always be sent
			require.Len(t, equipmentDataChannel, 1)
			eqData := <-equipmentDataChannel

			// Verify event properties
			assert.Equal(t, 1, eqData.EventID)
			assert.False(t, eqData.Error)
			assert.NotEmpty(t, eqData.Data)

			// Verify size calculation: header (28) + total payload
			totalPayload := tc.payloadSize * 2 // 2 packets
			expectedMinSize := totalPayload + 28
			assert.GreaterOrEqual(t, len(eqData.Data), expectedMinSize)
		})
	}
}

// TestReadPacket_ContextCancellation_ExitsGracefully verifies that readPacket responds
// to context cancellation and exits gracefully without hanging.
func TestReadPacket_ContextCancellation_ExitsGracefully(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	equipment := testhelpers.NewTestEquipment(1, 22)
	recvChannel := make(chan RingBufferData, 10)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ctx:           ctx,
		cancelCtx:     cancel,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	// Use a channel to detect when goroutine exits
	done := make(chan struct{})

	// Act: Start readPacket in goroutine
	go func() {
		readPacket(s, recvChannel, equipment, equipmentDataChannel, bufferTracking)
		close(done)
	}()

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Assert: Goroutine should exit
	select {
	case <-done:
		// Success - goroutine exited
	case <-time.After(1 * time.Second):
		t.Fatal("readPacket did not exit after context cancellation")
	}
}

// TestReadPacket_BasicProcessing_ProcessesSingleEvent verifies that readPacket
// processes a single complete event correctly and sends it to the output channel.
func TestReadPacket_BasicProcessing_ProcessesSingleEvent(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	equipment := testhelpers.NewTestEquipment(1, 22)
	recvChannel := make(chan RingBufferData, 10)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &server{
		ctx:           ctx,
		cancelCtx:     cancel,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	// Use a channel to detect when goroutine exits
	done := make(chan struct{})

	// Act: Start readPacket in goroutine
	go func() {
		readPacket(s, recvChannel, equipment, equipmentDataChannel, bufferTracking)
		close(done)
	}()

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Send a complete event through recvChannel
	packets := testhelpers.GenerateCompleteEvent(3, 100)
	for i, packet := range packets {
		bufferTracking.TryAcquirePosition(i, time.Second)
		bufferTracking.MarkPositionInUse(i)
		recvChannel <- RingBufferData{data: packet, position: i}
	}

	// Assert: Equipment data should be received on output channel
	select {
	case eqData := <-equipmentDataChannel:
		assert.Equal(t, 1, eqData.EventID, "Event ID should be 1")
		assert.Equal(t, equipment.ID, eqData.EquipmentID, "Equipment ID should match")
		assert.False(t, eqData.Error, "Event should not have errors")
		assert.NotEmpty(t, eqData.Data, "Event data should not be empty")
	case <-time.After(1 * time.Second):
		t.Fatal("Did not receive equipment data within timeout")
	}

	// Cancel context to stop goroutine
	cancel()

	// Wait for goroutine to exit
	select {
	case <-done:
		// Success - goroutine exited cleanly
	case <-time.After(1 * time.Second):
		t.Fatal("readPacket did not exit after context cancellation")
	}
}

// TestReadPacket_MultipleEvents_ProcessesSequentially verifies that readPacket
// correctly processes multiple events and increments event IDs.
func TestReadPacket_MultipleEvents_ProcessesSequentially(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	equipment := testhelpers.NewTestEquipment(1, 22)
	recvChannel := make(chan RingBufferData, 100)
	equipmentDataChannel := make(chan EquipmentData, 100)
	bufferTracking := NewRingBufferTracking(1000)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &server{
		ctx:           ctx,
		cancelCtx:     cancel,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	// Use a channel to detect when goroutine exits
	done := make(chan struct{})

	// Act: Start readPacket in goroutine
	go func() {
		readPacket(s, recvChannel, equipment, equipmentDataChannel, bufferTracking)
		close(done)
	}()

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Send 3 complete events through recvChannel
	numEvents := 3
	positionCounter := 0
	for eventNum := 0; eventNum < numEvents; eventNum++ {
		packets := testhelpers.GenerateCompleteEvent(3, 100)
		for _, packet := range packets {
			bufferTracking.TryAcquirePosition(positionCounter, time.Second)
			bufferTracking.MarkPositionInUse(positionCounter)
			recvChannel <- RingBufferData{data: packet, position: positionCounter}
			positionCounter++
		}
	}

	// Assert: Should receive 3 equipment data items
	receivedEvents := make([]EquipmentData, 0, numEvents)
	timeout := time.After(2 * time.Second)
	for len(receivedEvents) < numEvents {
		select {
		case eqData := <-equipmentDataChannel:
			receivedEvents = append(receivedEvents, eqData)
		case <-timeout:
			t.Fatalf("Expected %d events, received %d", numEvents, len(receivedEvents))
		}
	}

	// Verify event IDs are sequential
	for i, event := range receivedEvents {
		expectedEventID := i + 1
		assert.Equal(t, expectedEventID, event.EventID,
			"Event %d should have EventID %d", i, expectedEventID)
		assert.Equal(t, equipment.ID, event.EquipmentID,
			"Event %d should have correct EquipmentID", i)
		assert.False(t, event.Error,
			"Event %d should not have errors", i)
		assert.NotEmpty(t, event.Data,
			"Event %d should have data", i)
	}

	// Cancel context to stop goroutine
	cancel()

	// Wait for goroutine to exit
	select {
	case <-done:
		// Success - goroutine exited cleanly
	case <-time.After(1 * time.Second):
		t.Fatal("readPacket did not exit after context cancellation")
	}
}

// TestReadPacket_TimeBasedCancellation verifies that readPacket can be stopped
// using a timeout-based context cancellation.
func TestReadPacket_TimeBasedCancellation(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	equipment := testhelpers.NewTestEquipment(1, 22)
	recvChannel := make(chan RingBufferData, 10)
	equipmentDataChannel := make(chan EquipmentData, 10)
	bufferTracking := NewRingBufferTracking(100)

	// Create a context with automatic timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	s := &server{
		ctx:           ctx,
		cancelCtx:     cancel,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	done := make(chan struct{})

	// Act: Start readPacket in goroutine
	go func() {
		readPacket(s, recvChannel, equipment, equipmentDataChannel, bufferTracking)
		close(done)
	}()

	// Send one event before timeout
	packets := testhelpers.GenerateCompleteEvent(2, 50)
	for i, packet := range packets {
		bufferTracking.TryAcquirePosition(i, time.Second)
		bufferTracking.MarkPositionInUse(i)
		recvChannel <- RingBufferData{data: packet, position: i}
	}

	// Assert: Should receive the event
	select {
	case eqData := <-equipmentDataChannel:
		assert.Equal(t, 1, eqData.EventID)
		assert.False(t, eqData.Error)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Did not receive equipment data within timeout")
	}

	// Assert: Goroutine should exit when context times out
	select {
	case <-done:
		// Success - goroutine exited due to timeout
	case <-time.After(1 * time.Second):
		t.Fatal("readPacket did not exit after context timeout")
	}
}

// TestConcurrent_MultipleEquipments_SendToChannel tests that multiple equipment
// goroutines can concurrently send data to a shared equipmentDataChannel without races.
// This test is designed to be run with -race flag: go test -race ./ldcRPC/...
func TestConcurrent_MultipleEquipments_SendToChannel(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup - create shared equipment data channel
	equipmentDataChannel := make(chan EquipmentData, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	numEquipments := 5
	eventsPerEquipment := 10

	var wg sync.WaitGroup
	wg.Add(numEquipments)

	// Act - simulate multiple equipments sending data concurrently
	for eqID := 1; eqID <= numEquipments; eqID++ {
		go func(equipmentID int) {
			defer wg.Done()

			equipment := testhelpers.NewTestEquipment(equipmentID, 22)
			bufferTracking := NewRingBufferTracking(1000)

			var fragments [][]byte
			var bufferPositions []int
			eventID := 1
			expectedSequenceCounter := 0
			evtError := false

			positionCounter := 0
			for event := 0; event < eventsPerEquipment; event++ {
				// Generate a small event
				packets := testhelpers.GenerateCompleteEvent(3, 100)
				for _, packet := range packets {
					bufferTracking.TryAcquirePosition(positionCounter, time.Second)
					bufferTracking.MarkPositionInUse(positionCounter)
					bufferData := RingBufferData{data: packet, position: positionCounter}
					buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipment,
						equipmentDataChannel, &eventID, &evtError, &expectedSequenceCounter, bufferTracking)
					positionCounter++
				}
			}
		}(eqID)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Assert - should have received events from all equipments
	expectedEvents := numEquipments * eventsPerEquipment
	assert.Equal(t, expectedEvents, len(equipmentDataChannel))

	// Drain and verify channel
	receivedPerEquipment := make(map[int]int)
	for i := 0; i < expectedEvents; i++ {
		data := <-equipmentDataChannel
		receivedPerEquipment[data.EquipmentID]++
	}

	// Each equipment should have sent eventsPerEquipment events
	for eqID := 1; eqID <= numEquipments; eqID++ {
		assert.Equal(t, eventsPerEquipment, receivedPerEquipment[eqID],
			"Equipment %d should have sent %d events", eqID, eventsPerEquipment)
	}
}
