//go:build nohdf5

package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// Helper function to create a test server for ldcData tests
func createTestServerForLDCData(nGDCs int) *server {
	builder, cancel := testhelpers.NewMockContextBuilder().
		WithNGDCs(nGDCs).
		WithTimeout(100 * time.Millisecond)
	ctx := builder.Build()

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

// Helper function to create a test server without nGDCs (for error testing)
func createTestServerWithoutNGDCs() *server {
	ctx := context.Background()
	cancel := func() {}

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

// Helper function to create a test server with invalid nGDCs type (for error testing)
func createTestServerWithInvalidNGDCs() *server {
	ctx := context.WithValue(context.Background(), "nGDCs", "invalid")
	cancel := func() {}

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

// TestProcessData_SingleCompletePacket tests processing a complete event in one read
func TestProcessData_SingleCompletePacket(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(100, 1, 200)

	// Act
	err := processData(s, len(testEvent), testEvent, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert
	require.NoError(t, err)

	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 100, ldcData.EventID)
		assert.Equal(t, 1, ldcData.ldcID)
		assert.Equal(t, testEvent, ldcData.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for data")
	}

	assert.Equal(t, 101, expectedEventID, "expectedEventID should be incremented by nGDCs")
	assert.Equal(t, uint32(0), count, "count should be reset")
	assert.Equal(t, uint32(0), size, "size should be reset")
}

// TestProcessData_PacketFragmentation tests event split across two reads
func TestProcessData_PacketFragmentation(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(200, 2, 300)
	eventSize := len(testEvent)

	// Split into two parts
	splitPoint := eventSize / 2
	firstPart := testEvent[:splitPoint]
	secondPart := testEvent[splitPoint:]

	// Act - first part
	err1 := processData(s, len(firstPart), firstPart, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err1)

	// No data should be sent yet
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent for incomplete packet")
	default:
		// Expected
	}

	// Act - second part (completes the event)
	err2 := processData(s, len(secondPart), secondPart, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err2)

	// Assert - complete event should be sent
	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 200, ldcData.EventID)
		assert.Equal(t, 2, ldcData.ldcID)
		assert.Equal(t, testEvent, ldcData.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for data")
	}
}

// TestProcessData_MultipleEventsInOneRead tests recursive processing of multiple events
func TestProcessData_MultipleEventsInOneRead(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create two complete events
	event1 := testhelpers.CreateLDCEventData(300, 1, 150)
	event2 := testhelpers.CreateLDCEventData(301, 1, 150)

	// Concatenate them
	multiEventData := append(event1, event2...)

	// Act
	err := processData(s, len(multiEventData), multiEventData, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert
	require.NoError(t, err)

	// First event
	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 300, ldcData.EventID)
		assert.Equal(t, event1, ldcData.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for first event")
	}

	// Second event
	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 301, ldcData.EventID)
		assert.Equal(t, event2, ldcData.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for second event")
	}
}

// TestProcessData_EventIDSequence tests expectedEventID tracking
func TestProcessData_EventIDSequence(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create first event
	event1 := testhelpers.CreateLDCEventData(100, 1, 100)
	err1 := processData(s, len(event1), event1, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err1)

	assert.Equal(t, 101, expectedEventID, "After event 100, next expected should be 101")

	// Create second event with expected ID
	event2 := testhelpers.CreateLDCEventData(101, 1, 100)
	err2 := processData(s, len(event2), event2, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err2)

	assert.Equal(t, 102, expectedEventID, "After event 101, next expected should be 102")

	// Both events should be in data channel
	assert.Eventually(t, func() bool {
		return len(dataChannel) == 2
	}, 100*time.Millisecond, 10*time.Millisecond)
}

// TestProcessData_OutOfSequence_CancelsContext tests context cancellation on out-of-sequence event ID
func TestProcessData_OutOfSequence_CancelsContext(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 100) // Larger buffer
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Drain dataChannel in background
	go func() {
		for range dataChannel {
			// Just drain
		}
	}()

	// Create first event
	event1 := testhelpers.CreateLDCEventData(100, 1, 100)
	err1 := processData(s, len(event1), event1, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err1)

	// Create out-of-sequence event (expected 101, but got 105)
	event2 := testhelpers.CreateLDCEventData(105, 1, 100)
	err2 := processData(s, len(event2), event2, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err2)

	// Context should be cancelled - check server's context
	select {
	case <-s.getContext().Done():
		// Expected - context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled when out-of-sequence event is detected")
	}

	// expectedEventID should be updated to the new value (105 + nGDCs = 106)
	assert.Equal(t, 106, expectedEventID)
}

// TestProcessData_EventIDOverflow tests handling of event ID overflow
func TestProcessData_EventIDOverflow(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := 1000000 // Start with high value
	ldc := 0

	// Create event
	event1 := testhelpers.CreateLDCEventData(1000000, 1, 100)
	err1 := processData(s, len(event1), event1, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err1)

	// Next expected event ID should be incremented by nGDCs
	assert.Equal(t, 1000001, expectedEventID)
}

// TestProcessData_SmallPacketFirstRead tests reading size field in first 4 bytes
func TestProcessData_SmallPacketFirstRead(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(500, 1, 200)

	// Read only first 4 bytes (size field)
	first4Bytes := testEvent[:4]

	// Act - read just size
	err := processData(s, 4, first4Bytes, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err)

	// Size should be extracted
	expectedSize := uint32(len(testEvent))
	assert.Equal(t, expectedSize, size)

	// No data should be sent yet
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent with incomplete packet")
	default:
		// Expected
	}
}

// TestProcessData_VeryLargePacket tests handling of large packets
func TestProcessData_VeryLargePacket(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create large event (9600 bytes is the TCP read buffer size)
	largeEvent := testhelpers.CreateLDCEventData(600, 1, 9500)

	// Act
	err := processData(s, len(largeEvent), largeEvent, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert
	require.NoError(t, err)

	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 600, ldcData.EventID)
		assert.Equal(t, len(largeEvent), len(ldcData.Data))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for large packet")
	}
}

// TestReadLDCData_TCPRead tests reading data from a mock TCP connection
func TestReadLDCData_TCPRead(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	testEvent := testhelpers.CreateLDCEventData(700, 1, 200)
	mockConn := testhelpers.NewMockTCPConn([][]byte{testEvent})

	dataChannel := make(chan LDCData, 10)
	ldcConnectionCloseChan := make(chan net.Addr, 10)

	done := make(chan bool, 1)

	// Act - start connection handler in goroutine
	go func() {
		handleConnection(s, mockConn, dataChannel, ldcConnectionCloseChan)
		done <- true
	}()

	// Assert
	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 700, ldcData.EventID)
		assert.Equal(t, 1, ldcData.ldcID)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for data from TCP connection")
	}

	// Close the connection to stop handler
	s.cancelCtx()

	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for connection handler to finish")
	}
}

// TestReadLDCData_EOF_StopsHandler tests that EOF stops the connection handler
func TestReadLDCData_EOF_StopsHandler(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	// Setup - mock connection that returns EOF
	mockConn := testhelpers.NewMockTCPConn([][]byte{}) // Empty slice causes immediate EOF

	dataChannel := make(chan LDCData, 10)
	ldcConnectionCloseChan := make(chan net.Addr, 10)

	done := make(chan bool, 1)

	// Act - start connection handler
	go func() {
		handleConnection(s, mockConn, dataChannel, ldcConnectionCloseChan)
		done <- true
	}()

	// Assert - handler should finish on EOF
	select {
	case <-done:
		// Expected - handler finished
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for handler to finish on EOF")
	}

	// Close signal should be sent
	select {
	case addr := <-ldcConnectionCloseChan:
		assert.Equal(t, "127.0.0.2:6006", addr.String())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for close signal")
	}
}

// TestReadLDCData_ReadError_HandlesGracefully tests handling of TCP read errors
func TestReadLDCData_ReadError_HandlesGracefully(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	// Setup - create a mock connection that returns a read error
	mockConn := testhelpers.NewMockTCPConn([][]byte{})
	mockConn.SetReadError(fmt.Errorf("simulated TCP read error"))

	dataChannel := make(chan LDCData, 10)
	ldcConnectionCloseChan := make(chan net.Addr, 10)

	done := make(chan error, 1)

	// Act - start connection handler that will encounter the read error
	go func() {
		handleConnection(s, mockConn, dataChannel, ldcConnectionCloseChan)
		done <- nil
	}()

	// Assert - handler should continue running despite the error
	// (it logs the error and continues at line 48-51 in ldcData.go)
	// Wait less than the context timeout (100ms) to ensure context hasn't expired
	select {
	case <-done:
		t.Fatal("Handler should not exit on read error")
	case <-time.After(50 * time.Millisecond):
		// Expected - handler is still running
	}

	// Cancel context to stop the handler
	s.cancelCtx()

	select {
	case <-done:
		// Expected - handler finished after cancellation
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for handler to finish")
	}

	// Verify the connection was NOT closed (read errors don't close the connection)
	assert.False(t, mockConn.CloseCalled, "Connection should not be closed on read error")
}

// TestProcessData_MultipleGDCs_IncrementSequence tests event ID increment with multiple GDCs
func TestProcessData_MultipleGDCs_IncrementSequence(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(3)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create first event
	event1 := testhelpers.CreateLDCEventData(100, 1, 100)
	err1 := processData(s, len(event1), event1, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err1)

	// Next expected event ID should be incremented by nGDCs (3)
	assert.Equal(t, 103, expectedEventID)
}

// TestProcessData_MultiStageFragmentation tests complex fragmentation scenario
func TestProcessData_MultiStageFragmentation(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(800, 1, 300)
	eventSize := len(testEvent)

	// Split into three parts
	part1 := testEvent[:eventSize/3]
	part2 := testEvent[eventSize/3 : 2*eventSize/3]
	part3 := testEvent[2*eventSize/3:]

	// Act - process in three stages
	err1 := processData(s, len(part1), part1, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err1)

	err2 := processData(s, len(part2), part2, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err2)

	// No data should be sent yet
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent for incomplete packet")
	default:
	}

	err3 := processData(s, len(part3), part3, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err3)

	// Assert - complete event should be sent
	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 800, ldcData.EventID)
		assert.Equal(t, testEvent, ldcData.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for data")
	}
}

// TestHandleConnection_ContextCancellation tests that context cancellation stops handler
func TestHandleConnection_ContextCancellation(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	testEvent := testhelpers.CreateLDCEventData(900, 1, 200)
	mockConn := testhelpers.NewMockTCPConn([][]byte{testEvent})

	dataChannel := make(chan LDCData, 10)
	ldcConnectionCloseChan := make(chan net.Addr, 10)

	done := make(chan bool, 1)

	// Act - start handler
	go func() {
		handleConnection(s, mockConn, dataChannel, ldcConnectionCloseChan)
		done <- true
	}()

	// Wait for some data
	select {
	case <-dataChannel:
		// Got data
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for initial data")
	}

	// Cancel context
	s.cancelCtx()

	// Assert - handler should stop
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for handler to stop")
	}
}

// TestProcessData_ConcurrentProcessing tests thread-safety of processData
func TestProcessData_ConcurrentProcessing(t *testing.T) {
	// Setup
	numGoroutines := 10
	eventsPerGoroutine := 50

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			s := createTestServerForLDCData(1)
			eventData := make([][]byte, 0)
			dataChannel := make(chan LDCData, 100)
			count := uint32(0)
			size := uint32(0)
			expectedEventID := -1
			ldc := 0

			for j := 0; j < eventsPerGoroutine; j++ {
				eventID := goroutineID*eventsPerGoroutine + j
				testEvent := testhelpers.CreateLDCEventData(eventID, 1, 150)

				err := processData(s, len(testEvent), testEvent, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
				assert.NoError(t, err)

				// Consume from channel
				select {
				case <-dataChannel:
				case <-time.After(50 * time.Millisecond):
					// Channel might be full, that's ok
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestBinaryLittleEndian_PacketParsing tests binary parsing of packet headers
func TestBinaryLittleEndian_PacketParsing(t *testing.T) {
	// Setup
	testEvent := testhelpers.CreateLDCEventData(1000, 5, 200)

	// Act - parse fields
	eventSize := binary.LittleEndian.Uint32(testEvent[0:4])
	magicNumber := binary.LittleEndian.Uint32(testEvent[4:8])
	headerSize := binary.LittleEndian.Uint32(testEvent[8:12])
	eventID := binary.LittleEndian.Uint32(testEvent[24:28])
	ldcID := binary.LittleEndian.Uint32(testEvent[64:68])

	// Assert
	assert.Equal(t, uint32(len(testEvent)), eventSize)
	assert.Equal(t, uint32(duck.EVENT_MAGIC_NUMBER), magicNumber)
	assert.Equal(t, uint32(80), headerSize)
	assert.Equal(t, uint32(1000), eventID)
	assert.Equal(t, uint32(5), ldcID)
}

// TestProcessData_ContextCancellation_PartialDataNotSent tests that partial events are not sent to channel when context is cancelled
func TestProcessData_ContextCancellation_PartialDataNotSent(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create partial event
	partialEvent := testhelpers.CreateLDCEventData(1100, 1, 200)[:50]

	// Act - process partial event
	err := processData(s, len(partialEvent), partialEvent, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err)

	// Cancel context
	s.cancelCtx()

	// Assert - context should be done
	<-s.getContext().Done()

	// No data should be sent
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent after context cancellation")
	default:
		// Expected
	}
}

// TestProcessData_MissingNGDCs_Context tests error handling when nGDCs is missing from context
func TestProcessData_MissingNGDCs_Context(t *testing.T) {
	// Setup
	s := createTestServerWithoutNGDCs()
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(100, 1, 200)

	// Act
	err := processData(s, len(testEvent), testEvent, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting nGDCs from context")

	// No data should be sent
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent when nGDCs is missing")
	default:
		// Expected
	}
}

// TestProcessData_MissingNGDCs_InvalidType tests error handling when nGDCs has wrong type
func TestProcessData_MissingNGDCs_InvalidType(t *testing.T) {
	// Setup
	s := createTestServerWithInvalidNGDCs()
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(100, 1, 200)

	// Act
	err := processData(s, len(testEvent), testEvent, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error getting nGDCs from context")
}

// TestProcessData_TinyFragments_OneByte tests processing 1-byte fragments
func TestProcessData_TinyFragments_OneByte(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(100, 1, 200)

	// Send 1 byte at a time for first 10 bytes
	for i := 0; i < 10; i++ {
		err := processData(s, 1, testEvent[i:i+1], &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
		require.NoError(t, err)
	}

	// After 4 bytes, size should be extracted
	expectedSize := uint32(len(testEvent))
	assert.Equal(t, expectedSize, size)

	// No data should be sent yet (incomplete packet)
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent with incomplete packet")
	default:
		// Expected
	}

	// Send the rest
	err := processData(s, len(testEvent)-10, testEvent[10:], &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err)

	// Assert - complete event should be sent
	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 100, ldcData.EventID)
		assert.Equal(t, testEvent, ldcData.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for data")
	}
}

// TestProcessData_TinyFragments_ThreeBytes tests processing 3-byte fragments
func TestProcessData_TinyFragments_ThreeBytes(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event
	testEvent := testhelpers.CreateLDCEventData(200, 2, 200)

	// Send 3 bytes (not enough for size field)
	err := processData(s, 3, testEvent[:3], &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err)

	// Size should still be 0 (not enough bytes to read size)
	assert.Equal(t, uint32(0), size)
	assert.Equal(t, uint32(3), count)

	// Send 4th byte to complete size field
	err = processData(s, 1, testEvent[3:4], &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err)

	// Size should now be extracted
	expectedSize := uint32(len(testEvent))
	assert.Equal(t, expectedSize, size)

	// Send the rest
	err = processData(s, len(testEvent)-4, testEvent[4:], &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)
	require.NoError(t, err)

	// Assert - complete event should be sent
	select {
	case ldcData := <-dataChannel:
		assert.Equal(t, 200, ldcData.EventID)
		assert.Equal(t, testEvent, ldcData.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for data")
	}
}

// TestProcessData_CorruptedMagicNumber tests rejection of packets with invalid magic number
func TestProcessData_CorruptedMagicNumber(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create test event and corrupt the magic number (bytes 4-8)
	corruptedEvent := testhelpers.CreateLDCEventData(100, 1, 200)
	// Change magic number to something invalid
	binary.LittleEndian.PutUint32(corruptedEvent[4:8], 0xDEADBEEF)

	// Act
	err := processData(s, len(corruptedEvent), corruptedEvent, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid magic number")

	// No data should be sent
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent with corrupted magic number")
	default:
		// Expected
	}

	// State should be reset
	assert.Equal(t, uint32(0), count)
	assert.Equal(t, uint32(0), size)
	assert.Nil(t, eventData)

	// Context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected - context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled when corrupted magic number is detected")
	}
}

// TestProcessData_InvalidHeaders tests that data with valid magic but garbage headers is still sent
func TestProcessData_InvalidHeaders(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create garbage data with valid size and magic, but random content elsewhere
	garbageData := make([]byte, 200)
	binary.LittleEndian.PutUint32(garbageData[0:4], 200) // Size = 200
	binary.LittleEndian.PutUint32(garbageData[4:8], uint32(duck.EVENT_MAGIC_NUMBER)) // Valid magic
	// Fill rest with recognizable garbage pattern
	for i := 8; i < 200; i++ {
		garbageData[i] = byte(i % 256)
	}

	// Act
	err := processData(s, len(garbageData), garbageData, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert - should process (magic is valid, we only validate magic number)
	require.NoError(t, err)

	select {
	case ldcData := <-dataChannel:
		// Data was sent - verify it matches what we sent
		assert.Equal(t, garbageData, ldcData.Data, "Sent data should match received data")
		// Verify we can read the garbage values from the headers
		eventID := binary.LittleEndian.Uint32(ldcData.Data[24:28])
		ldcID := binary.LittleEndian.Uint32(ldcData.Data[64:68])
		// These will have the garbage values we filled in (little-endian)
		assert.Equal(t, uint32(0x1b1a1918), eventID) // bytes 24-27 = 24,25,26,27
		assert.Equal(t, uint32(0x43424140), ldcID)   // bytes 64-67 = 64,65,66,67
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout - data should be sent with valid magic number")
	}
}

// TestProcessData_TooSmallSize tests rejection of packets with size < 80 bytes
func TestProcessData_TooSmallSize(t *testing.T) {
	// Setup
	s := createTestServerForLDCData(1)
	eventData := make([][]byte, 0)
	dataChannel := make(chan LDCData, 10)
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1
	ldc := 0

	// Create packet with size 50 (less than minimum header size)
	smallPacket := make([]byte, 50)
	binary.LittleEndian.PutUint32(smallPacket[0:4], 50)
	binary.LittleEndian.PutUint32(smallPacket[4:8], uint32(duck.EVENT_MAGIC_NUMBER))

	// Act
	err := processData(s, len(smallPacket), smallPacket, &eventData, dataChannel, &count, &size, &expectedEventID, &ldc)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete event header")

	// No data should be sent
	select {
	case <-dataChannel:
		t.Fatal("No data should be sent with too small size")
	default:
		// Expected
	}

	// Context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected - context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled when invalid size is detected")
	}
}

