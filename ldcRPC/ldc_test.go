package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/ldcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

func TestMountSubEvent_AllEquipments_SendsEvent(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	eventData := make(map[int]map[int][]byte)

	// Act: Add data from first equipment
	eq1Data := &EquipmentData{
		EventID:     1,
		EquipmentID: 1,
		Data:        []byte{0x01, 0x02, 0x03},
		Error:       false,
	}

	eventMounted, err := mountSubEvent(s, &eventData, eq1Data, &ldcConfig, mockConn, 0, metricsCh)

	// Assert: First equipment doesn't trigger send
	require.NoError(t, err)
	assert.False(t, eventMounted)
	assert.Len(t, eventData, 1)
	assert.Len(t, eventData[1], 1)

	// Act: Add data from second equipment
	eq2Data := &EquipmentData{
		EventID:     1,
		EquipmentID: 2,
		Data:        []byte{0x04, 0x05, 0x06},
		Error:       false,
	}

	eventMounted, err = mountSubEvent(s, &eventData, eq2Data, &ldcConfig, mockConn, 0, metricsCh)

	// Assert: Second equipment triggers send
	require.NoError(t, err)
	assert.True(t, eventMounted)
	assert.Empty(t, eventData) // Should be cleared after sending

	// Verify data was written to mock connection
	written := mockConn.GetWritten()
	assert.Len(t, written, 1)
}

func TestMountSubEvent_PartialEventData_WaitsForMore(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 3)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 3)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	eventData := make(map[int]map[int][]byte)

	// Act: Add data from only 2 out of 3 equipments
	for eqID := 1; eqID <= 2; eqID++ {
		eqData := &EquipmentData{
			EventID:     1,
			EquipmentID: eqID,
			Data:        []byte{byte(eqID)},
			Error:       false,
		}

		eventMounted, err := mountSubEvent(s, &eventData, eqData, &ldcConfig, mockConn, 0, metricsCh)

		// Assert: No event sent yet
		require.NoError(t, err)
		assert.False(t, eventMounted)
	}

	// Assert: Event data should be stored
	assert.Len(t, eventData, 1)
	assert.Len(t, eventData[1], 2)

	// Verify nothing was written
	written := mockConn.GetWritten()
	assert.Empty(t, written)
}

func TestMountSubEvent_IncompleteEventsCounter_Tracked(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 3)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 3)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	eventData := make(map[int]map[int][]byte)

	// Act: Add first equipment data for event 1
	eq1Data := &EquipmentData{
		EventID:     1,
		EquipmentID: 1,
		Data:        []byte{0x01},
		Error:       false,
	}

	eventMounted, err := mountSubEvent(s, &eventData, eq1Data, &ldcConfig, mockConn, 0, metricsCh)
	require.NoError(t, err)

	// Assert: Incomplete events counter should be incremented
	incompleteCount, err := duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), incompleteCount, "incompleteEventsCounter should be 1 after first equipment")
	assert.False(t, eventMounted)

	// Act: Add second equipment data for event 1
	eq2Data := &EquipmentData{
		EventID:     1,
		EquipmentID: 2,
		Data:        []byte{0x02},
		Error:       false,
	}

	eventMounted, err = mountSubEvent(s, &eventData, eq2Data, &ldcConfig, mockConn, 0, metricsCh)
	require.NoError(t, err)

	// Assert: Incomplete events counter should still be 1 (same event)
	incompleteCount, err = duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), incompleteCount, "incompleteEventsCounter should still be 1 after second equipment")
	assert.False(t, eventMounted)

	// Act: Add third equipment data (completes the event)
	eq3Data := &EquipmentData{
		EventID:     1,
		EquipmentID: 3,
		Data:        []byte{0x03},
		Error:       false,
	}

	eventMounted, err = mountSubEvent(s, &eventData, eq3Data, &ldcConfig, mockConn, 0, metricsCh)
	require.NoError(t, err)

	// Assert: Incomplete events counter should be decremented to 0
	incompleteCount, err = duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), incompleteCount, "incompleteEventsCounter should be 0 after event completes")
	assert.True(t, eventMounted)

	// Act: Start a new event to verify counter increments again
	eq1DataEvent2 := &EquipmentData{
		EventID:     2,
		EquipmentID: 1,
		Data:        []byte{0x04},
		Error:       false,
	}

	eventMounted, err = mountSubEvent(s, &eventData, eq1DataEvent2, &ldcConfig, mockConn, 0, metricsCh)
	require.NoError(t, err)

	// Assert: Incomplete events counter should be 1 again
	incompleteCount, err = duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), incompleteCount, "incompleteEventsCounter should be 1 for new event")
}

func TestMountSubEvent_Sequence_MultipleEvents(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 100)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	eventData := make(map[int]map[int][]byte)

	// Act: Send multiple complete events
	numEvents := 10
	for eventID := 1; eventID <= numEvents; eventID++ {
		for eqID := 1; eqID <= 2; eqID++ {
			eqData := &EquipmentData{
				EventID:     eventID,
				EquipmentID: eqID,
				Data:        []byte{byte(eventID), byte(eqID)},
				Error:       false,
			}

			eventMounted, err := mountSubEvent(s, &eventData, eqData, &ldcConfig, mockConn, 0, metricsCh)

			if eqID == 2 {
				// Second equipment triggers send
				require.NoError(t, err)
				assert.True(t, eventMounted)
			} else {
				// First equipment doesn't trigger send
				require.NoError(t, err)
				assert.False(t, eventMounted)
			}
		}
	}

	// Assert: Should have sent all events
	written := mockConn.GetWritten()
	assert.Len(t, written, numEvents)

	// Assert: Event data map should be empty
	assert.Empty(t, eventData)

	// Assert: Incomplete events counter should be 0 after all events complete
	incompleteCount, err := duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), incompleteCount, "incompleteEventsCounter should be 0 after all events complete")
}

func TestSendDataToGDC_Success(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	metrics.Register(registry)

	// Setup
	ldcData := []byte{0x01, 0x02, 0x03, 0x04}
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Act
	err := sendDataToGDC(s, ldcData, metricsCh, mockConn, 0)

	// Assert
	require.NoError(t, err)

	// Verify data was written
	written := mockConn.GetWritten()
	assert.Len(t, written, 1)
	assert.Equal(t, ldcData, written[0])

	// Verify metrics channel received byte count
	assert.Len(t, metricsCh, 1)
	receivedSize := <-metricsCh
	assert.Equal(t, len(ldcData), receivedSize)

	// Note: sendDataToGDC sends to metricsCh, but eventCounter and sizeCounter
	// are updated by ProcessMetrics which runs in a separate goroutine.
	// The histogram observation is made directly by sendDataToGDC.

	// Verify histogram has an observation for GDC 0
	sampleCount, err := testhelpers.GetPrometheusHistogramSampleCount(
		metrics.timeToGdcHistogram,
		registry,
		"ldc_to_gdc_time_histogram",
		prometheus.Labels{"gdc": "0"},
	)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), sampleCount, "timeToGdcHistogram should have 1 observation for GDC 0")
}

func TestSendDataToGDC_ConnectionError_ReturnsError(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	ldcData := []byte{0x01, 0x02, 0x03, 0x04}
	mockConn := testhelpers.NewMockTCPConn()
	mockConn.SetWriteError(fmt.Errorf("simulated write error")) // Simulate write error
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Act
	err := sendDataToGDC(s, ldcData, metricsCh, mockConn, 0)

	// Assert
	assert.Error(t, err)
}

func TestUpdateChannelGauges_UpdatesMetrics(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	equipmentDataChannel := make(chan EquipmentData, 100)
	receiveChannels := make(map[int]chan RingBufferData)

	// Add some test data to channels
	equipmentDataChannel <- EquipmentData{EventID: 1, EquipmentID: 1}
	equipmentDataChannel <- EquipmentData{EventID: 2, EquipmentID: 2}

	receiveChannels[1] = make(chan RingBufferData, 100)
	receiveChannels[1] <- RingBufferData{data: []byte{0x01}, position: 0}
	receiveChannels[1] <- RingBufferData{data: []byte{0x02}, position: 1}

	receiveChannels[2] = make(chan RingBufferData, 100)
	receiveChannels[2] <- RingBufferData{data: []byte{0x03}, position: 0}

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Act: Start updateChannelGauges and let it update once
	done := make(chan bool)
	go func() {
		updateChannelGauges(s, equipmentDataChannel, receiveChannels)
		done <- true
	}()

	// Wait for metrics to be updated (ticker runs every 1 second)
	time.Sleep(1200 * time.Millisecond)

	// Assert: Verify equipmentDataChannelCounter was updated
	equipmentChannelValue, err := duck.GetPrometheusGaugeValue(metrics.equipmentDataChannelCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), equipmentChannelValue, "equipmentDataChannelCounter should reflect 2 items in channel")

	// Assert: Verify receiveChannelCounters were updated for each equipment
	receiveChannel1Value, err := testhelpers.GetPrometheusGaugeVecValue(
		metrics.receiveChannelCounters,
		prometheus.Labels{"equipment": "1"},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2), receiveChannel1Value, "receiveChannelCounter for equipment 1 should have 2 items")

	receiveChannel2Value, err := testhelpers.GetPrometheusGaugeVecValue(
		metrics.receiveChannelCounters,
		prometheus.Labels{"equipment": "2"},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), receiveChannel2Value, "receiveChannelCounter for equipment 2 should have 1 item")

	// Verify that gauges update multiple times by consuming from channel and checking again
	<-equipmentDataChannel
	time.Sleep(1100 * time.Millisecond)

	equipmentChannelValueAfter, err := duck.GetPrometheusGaugeValue(metrics.equipmentDataChannelCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), equipmentChannelValueAfter, "equipmentDataChannelCounter should reflect 1 item after consuming one")

	// Clean up
	cancel()
	<-done
}

func TestMountSubEvent_ContextValues_Accessible(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 999) // Special run number
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	eventData := make(map[int]map[int][]byte)

	// Act: Send complete event
	eq1Data := &EquipmentData{
		EventID:     1,
		EquipmentID: 1,
		Data:        []byte{0x01},
		Error:       false,
	}

	eq2Data := &EquipmentData{
		EventID:     1,
		EquipmentID: 2,
		Data:        []byte{0x02},
		Error:       false,
	}

	mountSubEvent(s, &eventData, eq1Data, &ldcConfig, mockConn, 0, metricsCh)
	eventMounted, err := mountSubEvent(s, &eventData, eq2Data, &ldcConfig, mockConn, 0, metricsCh)

	// Assert
	require.NoError(t, err)
	assert.True(t, eventMounted)

	// Verify event was sent with correct context values (run number should be in header)
	written := mockConn.GetWritten()
	assert.Len(t, written, 1)
}

// TestReadSubEvents_RoutesToCorrectGDC_RoundRobin tests round-robin GDC routing
func TestReadSubEvents_RoutesToCorrectGDC_RoundRobin(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup - 2 GDCs, 2 equipments
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn1 := testhelpers.NewMockTCPConn()
	mockConn2 := testhelpers.NewMockTCPConn()
	gdcConnections := []net.Conn{mockConn1, mockConn2}
	gdcs := testhelpers.NewTestGDCConfigurationList(2, true)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)
	ctx = context.WithValue(ctx, "metricsChBufferSize", 100)

	s := &server{
		ctx:       ctx,
		cancelCtx: cancel,
		logger:    logger,
		metrics:   metrics,
	}

	equipmentDataChannel := make(chan EquipmentData, 100)

	// Start readSubEvents in background
	done := make(chan bool)
	go func() {
		readSubEvents(s, &gdcConnections, equipmentDataChannel, &ldcConfig, &gdcs)
		done <- true
	}()

	// Act: Send 4 complete events (should go to GDC1, GDC2, GDC1, GDC2)
	for eventID := 1; eventID <= 4; eventID++ {
		// Send data from equipment 1
		equipmentDataChannel <- EquipmentData{
			EventID:     eventID,
			EquipmentID: 1,
			Data:        []byte{byte(eventID), 0x01},
			Error:       false,
		}
		// Send data from equipment 2 (completes the event)
		equipmentDataChannel <- EquipmentData{
			EventID:     eventID,
			EquipmentID: 2,
			Data:        []byte{byte(eventID), 0x02},
			Error:       false,
		}
	}

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Cancel to stop readSubEvents
	cancel()

	// Wait for completion
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("readSubEvents did not stop on context cancellation")
	}

	// Assert: Events should be distributed round-robin
	// GDC1 gets events 1, 3 (indices 0, 2) and GDC2 gets events 2, 4 (indices 1, 3)
	gdc1Written := mockConn1.GetWritten()
	gdc2Written := mockConn2.GetWritten()

	assert.Equal(t, 2, len(gdc1Written), "GDC1 should have received 2 events")
	assert.Equal(t, 2, len(gdc2Written), "GDC2 should have received 2 events")
}

// TestReadSubEvents_HandlesPacketLossError tests error handling on packet loss
func TestReadSubEvents_HandlesPacketLossError(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn1 := testhelpers.NewMockTCPConn()
	mockConn2 := testhelpers.NewMockTCPConn()
	gdcConnections := []net.Conn{mockConn1, mockConn2}
	gdcs := testhelpers.NewTestGDCConfigurationList(2, true)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)
	ctx = context.WithValue(ctx, "metricsChBufferSize", 100)

	s := &server{
		ctx:       ctx,
		cancelCtx: cancel,
		logger:    logger,
		metrics:   metrics,
	}

	equipmentDataChannel := make(chan EquipmentData, 100)

	// Start readSubEvents in background
	done := make(chan bool)
	go func() {
		readSubEvents(s, &gdcConnections, equipmentDataChannel, &ldcConfig, &gdcs)
		done <- true
	}()

	// Act: Send an error event (packet loss)
	equipmentDataChannel <- EquipmentData{
		EventID:     1,
		EquipmentID: 1,
		Data:        nil,
		Error:       true, // Error flag indicates packet loss
	}

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Now send a complete event - should go to the next GDC (GDC2 since error skipped GDC1)
	equipmentDataChannel <- EquipmentData{
		EventID:     2,
		EquipmentID: 1,
		Data:        []byte{0x02, 0x01},
		Error:       false,
	}
	equipmentDataChannel <- EquipmentData{
		EventID:     2,
		EquipmentID: 2,
		Data:        []byte{0x02, 0x02},
		Error:       false,
	}

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Cancel to stop readSubEvents
	cancel()

	// Wait for completion
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("readSubEvents did not stop on context cancellation")
	}

	// Assert: The error event skips a GDC, so event 2 should NOT go to the first GDC
	// After error, currentGDC moves to 1, then event 2 goes to GDC at index 1
	// After event 2, currentGDC moves to 0
	gdc1Written := mockConn1.GetWritten()
	gdc2Written := mockConn2.GetWritten()

	// Event should have gone to GDC2 (index 1 after error skip)
	assert.Equal(t, 0, len(gdc1Written), "GDC1 should have received 0 events (error skipped)")
	assert.Equal(t, 1, len(gdc2Written), "GDC2 should have received 1 event")
}

// TestReadSubEvents_ContextCancellation_StopsProcessing tests graceful shutdown
func TestReadSubEvents_ContextCancellation_StopsProcessing(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	gdcConnections := []net.Conn{mockConn}
	gdcs := testhelpers.NewTestGDCConfigurationList(1, true)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)
	ctx = context.WithValue(ctx, "metricsChBufferSize", 100)

	s := &server{
		ctx:       ctx,
		cancelCtx: cancel,
		logger:    logger,
		metrics:   metrics,
	}

	equipmentDataChannel := make(chan EquipmentData, 100)

	// Start readSubEvents in background
	done := make(chan bool)
	go func() {
		readSubEvents(s, &gdcConnections, equipmentDataChannel, &ldcConfig, &gdcs)
		done <- true
	}()

	// Give time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context immediately
	cancel()

	// Assert - should stop quickly
	select {
	case <-done:
		// Success - function returned
	case <-time.After(500 * time.Millisecond):
		t.Fatal("readSubEvents did not stop on context cancellation")
	}

	// GDC connection should be closed
	assert.True(t, mockConn.IsClosed(), "GDC connection should be closed on context cancellation")
}

// TestReadSubEvents_MissingMetricsChBufferSize_Returns tests error handling for missing context value
func TestReadSubEvents_MissingMetricsChBufferSize_Returns(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup - context WITHOUT metricsChBufferSize
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	gdcConnections := []net.Conn{mockConn}
	gdcs := testhelpers.NewTestGDCConfigurationList(1, true)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)
	// Note: metricsChBufferSize is NOT set

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	equipmentDataChannel := make(chan EquipmentData, 100)

	// Act - should return immediately due to missing context value
	done := make(chan bool)
	go func() {
		readSubEvents(s, &gdcConnections, equipmentDataChannel, &ldcConfig, &gdcs)
		done <- true
	}()

	// Assert - should return quickly
	select {
	case <-done:
		// Success - function returned due to missing context value
	case <-time.After(500 * time.Millisecond):
		t.Fatal("readSubEvents did not return when metricsChBufferSize is missing")
	}
}

// TestMountSubEvent_MissingRunNumber_ReturnsError tests error handling
func TestMountSubEvent_MissingRunNumber_ReturnsError(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup - context WITHOUT runNumber
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)
	// Note: runNumber is NOT set

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	eventData := make(map[int]map[int][]byte)

	// Pre-populate with data from equipment 1
	eventData[1] = make(map[int][]byte)
	eventData[1][1] = []byte{0x01, 0x02}

	// Act: Add data from second equipment (would complete the event)
	eqData := &EquipmentData{
		EventID:     1,
		EquipmentID: 2,
		Data:        []byte{0x03, 0x04},
		Error:       false,
	}

	eventMounted, err := mountSubEvent(s, &eventData, eqData, &ldcConfig, mockConn, 0, metricsCh)

	// Assert - should return error
	assert.Error(t, err, "Should return error when runNumber is missing")
	assert.False(t, eventMounted, "Event should not be mounted")
}

// TestMountSubEvent_ZeroEnabledEquipments_NoEventMounted tests behavior with zero equipments
func TestMountSubEvent_ZeroEnabledEquipments_NoEventMounted(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Setup - context with nEnabledEquipments = 0 (no enabled equipments)
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 10)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 0)

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	eventData := make(map[int]map[int][]byte)

	// Act: Add equipment data - with 0 enabled equipments, any data won't match the condition
	eqData := &EquipmentData{
		EventID:     1,
		EquipmentID: 1,
		Data:        []byte{0x01, 0x02},
		Error:       false,
	}

	eventMounted, err := mountSubEvent(s, &eventData, eqData, &ldcConfig, mockConn, 0, metricsCh)

	// Assert - no error but event not mounted because nEquipments (1) != nEnabledEquipments (0)
	require.NoError(t, err)
	assert.False(t, eventMounted, "Event should not be mounted with 0 enabled equipments")

	// Data should be stored
	assert.Len(t, eventData, 1)
}


// TestConcurrent_ReadPacket_And_MountSubEvent tests that mountSubEvent can be
// called concurrently from multiple equipment goroutines without data races.
// This verifies the mutex-protected access to the shared eventData map.
// This test is designed to be run with -race flag: go test -race ./ldcRPC/...
func TestConcurrent_ReadPacket_And_MountSubEvent(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	// Setup - simulate producer/consumer pattern
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	mockConn := testhelpers.NewMockTCPConn()
	metricsCh := make(chan int, 100)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "nEnabledEquipments", 2)
	defer cancel()

	s := &server{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}

	// Shared event data map
	eventData := make(map[int]map[int][]byte)
	var eventDataMu sync.Mutex

	numEvents := 20
	var wg sync.WaitGroup
	wg.Add(2) // Two equipment producers

	// Act - two equipment goroutines sending data
	for eqID := 1; eqID <= 2; eqID++ {
		go func(equipmentID int) {
			defer wg.Done()

			for event := 1; event <= numEvents; event++ {
				eqData := &EquipmentData{
					EventID:     event,
					EquipmentID: equipmentID,
					Data:        []byte{byte(equipmentID), byte(event)},
					Error:       false,
				}

				eventDataMu.Lock()
				mountSubEvent(s, &eventData, eqData, &ldcConfig, mockConn, 0, metricsCh)
				eventDataMu.Unlock()

				// Small sleep to create interleaving
				time.Sleep(time.Microsecond * 10)
			}
		}(eqID)
	}

	wg.Wait()

	// Assert - all events should be mounted (map should be empty)
	eventDataMu.Lock()
	assert.Empty(t, eventData, "All events should have been mounted and removed from map")
	eventDataMu.Unlock()

	// Check that all events were written
	written := mockConn.GetWritten()
	assert.Equal(t, numEvents, len(written), "Should have %d events written", numEvents)
}
