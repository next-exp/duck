//go:build integration
// +build integration

package centrifuge_test

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// TestCentrifuge_DuckLoggerMetric tests that DuckLogger.Metric() with Metrics struct works correctly
func TestCentrifuge_DuckLoggerMetric(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create publisher
	pubClient, err := duck.CentrifugeConnection(Config, "testuser-logger-metric")
	require.NoError(t, err)
	defer pubClient.Close()

	err = testhelpers.WaitForConnection(pubClient, 5*time.Second)
	require.NoError(t, err)

	pubSub, err := duck.SubscribeCentrifuge(pubClient, nil)
	require.NoError(t, err)
	defer pubSub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(pubSub, 5*time.Second)
	require.NoError(t, err)

	// Create subscriber to receive metrics
	subClient, err := duck.CentrifugeConnection(Config, "testuser-logger-metric-sub")
	require.NoError(t, err)
	defer subClient.Close()

	err = testhelpers.WaitForConnection(subClient, 5*time.Second)
	require.NoError(t, err)

	tracker := testhelpers.NewMessageTracker()
	sub, err := duck.SubscribeCentrifuge(subClient, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Create DuckLogger with Centrifuge subscription
	logger := duck.NewDuckLogger("logger-service", pubSub, slog.LevelInfo)

	// Create metrics using the Metrics struct (like production code)
	metrics := duck.Metrics{
		EventCounter:    5000,
		ByteCounter:     1024000,
		CurrentTrgRate:  250.5,
		CurrentDataRate: 51200.0,
		AvgTrgRate:      245.0,
		AvgDataRate:     50000.0,
	}

	// Marshal to JSON (like production code does in pkg/metrics.go)
	metricJSON, err := json.Marshal(metrics)
	require.NoError(t, err, "Should marshal metrics to JSON")

	// Log metrics using DuckLogger
	logger.Metric(string(metricJSON))

	// Wait for message to be received
	err = tracker.WaitFor(1, 5*time.Second)
	require.NoError(t, err, "Should receive metrics message within 5 seconds")

	// Verify message properties
	messages := tracker.GetAll()
	require.Len(t, messages, 1, "Should have received exactly one metrics message")

	msg := messages[0]
	assert.Equal(t, duck.MessageMetric, msg.Type)
	assert.Equal(t, 0, msg.RunNumber, "Run number should be 0 (default)")
	assert.Equal(t, "logger-service", msg.Host, "Host should match logger service name")

	// Verify metrics JSON structure
	var receivedMetrics map[string]interface{}
	err = json.Unmarshal([]byte(msg.Value), &receivedMetrics)
	require.NoError(t, err, "Metrics value should be valid JSON")

	// Note: Go's default JSON marshaling uses PascalCase field names
	assert.Equal(t, float64(5000), receivedMetrics["EventCounter"], "EventCounter should match")
	assert.Equal(t, float64(1024000), receivedMetrics["ByteCounter"], "ByteCounter should match")
	assert.Equal(t, 250.5, receivedMetrics["CurrentTrgRate"], "CurrentTrgRate should match")
	assert.Equal(t, 51200.0, receivedMetrics["CurrentDataRate"], "CurrentDataRate should match")
	assert.Equal(t, 245.0, receivedMetrics["AvgTrgRate"], "AvgTrgRate should match")
	assert.Equal(t, 50000.0, receivedMetrics["AvgDataRate"], "AvgDataRate should match")

	assert.False(t, msg.StopProcesses)
}

// TestCentrifuge_DuckLoggerState tests that DuckLogger.State() publishes state messages correctly
func TestCentrifuge_DuckLoggerState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create subscriber
	client, err := duck.CentrifugeConnection(Config, "testuser-logger-state")
	require.NoError(t, err)
	defer client.Close()

	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	tracker := testhelpers.NewMessageTracker()
	sub, err := duck.SubscribeCentrifuge(client, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Create DuckLogger
	logger := duck.NewDuckLogger("state-test-service", sub, slog.LevelInfo)

	// Test multiple state transitions
	states := []duck.StateType{
		duck.INITIALIZED,
		duck.STARTING,
		duck.RUNNING,
		duck.PINGING,
		duck.STOPPING,
	}

	for _, state := range states {
		logger.State(state)
	}

	// Wait for all state messages
	err = tracker.WaitFor(len(states), 10*time.Second)
	require.NoError(t, err, "Should receive all state messages")

	// Verify all state messages
	messages := tracker.GetAll()
	require.Len(t, messages, len(states), "Should have exactly 5 state messages")

	for i, msg := range messages {
		assert.Equal(t, duck.MessageState, msg.Type, "Message type should be STATE")
		assert.Equal(t, "state-test-service", msg.Host, "Host should match logger service name")
		assert.Equal(t, 0, msg.RunNumber, "Run number should be 0 (default)")
		assert.False(t, msg.StopProcesses, "State messages should not stop processes")

		// Verify state JSON structure
		var stateData map[string]string
		err := json.Unmarshal([]byte(msg.Value), &stateData)
		require.NoError(t, err, "State value should be valid JSON")
		assert.Equal(t, states[i].String(), stateData["state"], "State should match expected value")
	}
}

// TestCentrifuge_DuckLoggerError tests that DuckLogger error messages are published correctly
func TestCentrifuge_DuckLoggerError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create subscriber
	client, err := duck.CentrifugeConnection(Config, "testuser-logger-error")
	require.NoError(t, err)
	defer client.Close()

	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	tracker := testhelpers.NewMessageTracker()
	sub, err := duck.SubscribeCentrifuge(client, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Create DuckLogger
	logger := duck.NewDuckLogger("error-test-service", sub, slog.LevelInfo)

	// Test both error types
	errorMsg := "Critical error occurred"
	logger.Slog.Error(errorMsg)

	nonStoppingErrorMsg := "Non-critical warning"
	logger.NonStoppingError(nonStoppingErrorMsg)

	// Wait for both error messages
	err = tracker.WaitFor(2, 10*time.Second)
	require.NoError(t, err, "Should receive both error messages")

	// Verify error messages
	messages := tracker.GetAll()
	require.Len(t, messages, 2, "Should have exactly 2 error messages")

	// First message should be stopping error (from Slog.Error)
	assert.Equal(t, duck.MessageError, messages[0].Type, "First message type should be ERROR")
	assert.Equal(t, "error-test-service", messages[0].Host)
	assert.Equal(t, errorMsg, messages[0].Value)
	assert.True(t, messages[0].StopProcesses, "Regular errors should stop processes")

	// Second message should be non-stopping error
	assert.Equal(t, duck.MessageError, messages[1].Type, "Second message type should be ERROR")
	assert.Equal(t, "error-test-service", messages[1].Host)
	assert.Equal(t, nonStoppingErrorMsg, messages[1].Value)
	assert.False(t, messages[1].StopProcesses, "NonStoppingError should not stop processes")
}

// TestCentrifuge_DuckLoggerSummary tests that DuckLogger.Summary() publishes summary messages correctly
func TestCentrifuge_DuckLoggerSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create subscriber
	client, err := duck.CentrifugeConnection(Config, "testuser-logger-summary")
	require.NoError(t, err)
	defer client.Close()

	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	tracker := testhelpers.NewMessageTracker()
	sub, err := duck.SubscribeCentrifuge(client, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Create DuckLogger with run number
	logger := duck.NewDuckLoggerWithRun("summary-test-service", 42, sub, slog.LevelInfo)

	// Create summary statistics
	stats := duck.RunStatistics{
		Host:   "summary-test-service",
		Events: 15000,
		Bytes:  1024000,
		Errors: 5,
	}

	logger.Summary(stats)

	// Wait for summary message
	err = tracker.WaitFor(1, 10*time.Second)
	require.NoError(t, err, "Should receive summary message")

	// Verify summary message
	messages := tracker.GetAll()
	require.Len(t, messages, 1, "Should have exactly 1 summary message")

	msg := messages[0]
	assert.Equal(t, duck.MessageSummary, msg.Type, "Message type should be SUMMARY")
	assert.Equal(t, "summary-test-service", msg.Host)
	assert.Equal(t, 42, msg.RunNumber, "Run number should match")
	assert.False(t, msg.StopProcesses, "Summary messages should not stop processes")

	// Verify summary JSON structure
	var summaryData map[string]interface{}
	err = json.Unmarshal([]byte(msg.Value), &summaryData)
	require.NoError(t, err, "Summary value should be valid JSON")

	assert.Equal(t, "summary-test-service", summaryData["host"])
	assert.Equal(t, float64(15000), summaryData["events"])
	assert.Equal(t, float64(1024000), summaryData["bytes"])
	assert.Equal(t, float64(5), summaryData["errors"])
}

// TestCentrifuge_DuckLoggerOutputFile tests that DuckLogger.OutputFile() publishes file messages correctly
func TestCentrifuge_DuckLoggerOutputFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create subscriber
	client, err := duck.CentrifugeConnection(Config, "testuser-logger-file")
	require.NoError(t, err)
	defer client.Close()

	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	tracker := testhelpers.NewMessageTracker()
	sub, err := duck.SubscribeCentrifuge(client, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Create DuckLogger
	logger := duck.NewDuckLogger("file-test-service", sub, slog.LevelInfo)

	// Test output file messages
	fileInfos := []struct {
		server string
		subrun int
	}{
		{"gdc1", 1},
		{"gdc2", 2},
		{"gdc3", 3},
	}

	for _, fi := range fileInfos {
		logger.OutputFile(fi.server, fi.subrun)
	}

	// Wait for all file messages
	err = tracker.WaitFor(len(fileInfos), 10*time.Second)
	require.NoError(t, err, "Should receive all file messages")

	// Verify file messages
	messages := tracker.GetAll()
	require.Len(t, messages, len(fileInfos), "Should have exactly 3 file messages")

	for i, msg := range messages {
		assert.Equal(t, duck.MessageFile, msg.Type, "Message type should be FILE")
		assert.Equal(t, "file-test-service", msg.Host)
		assert.False(t, msg.StopProcesses, "File messages should not stop processes")

		// Verify file JSON structure
		var fileData map[string]interface{}
		err := json.Unmarshal([]byte(msg.Value), &fileData)
		require.NoError(t, err, "File value should be valid JSON")
		assert.Equal(t, fileInfos[i].server, fileData["server"])
		assert.Equal(t, float64(fileInfos[i].subrun), fileData["subrun"])
	}
}

// TestCentrifuge_DuckLoggerMixedMessages tests that different DuckLogger methods work together correctly
func TestCentrifuge_DuckLoggerMixedMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create subscriber
	client, err := duck.CentrifugeConnection(Config, "testuser-logger-mixed")
	require.NoError(t, err)
	defer client.Close()

	err = testhelpers.WaitForConnection(client, 5*time.Second)
	require.NoError(t, err)

	tracker := testhelpers.NewMessageTracker()
	sub, err := duck.SubscribeCentrifuge(client, func(msg *duck.Message) {
		tracker.Add(msg)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = testhelpers.WaitForSubscriptionReady(sub, 5*time.Second)
	require.NoError(t, err)

	// Create DuckLogger
	logger := duck.NewDuckLoggerWithRun("mixed-test-service", 100, sub, slog.LevelInfo)

	// Send various message types
	logger.State(duck.INITIALIZED)

	// Create and marshal metrics (like production code)
	metrics := duck.Metrics{
		EventCounter:    100,
		ByteCounter:     1024,
		CurrentTrgRate:  10.0,
		CurrentDataRate: 102.4,
		AvgTrgRate:      9.5,
		AvgDataRate:     100.0,
	}
	metricJSON, _ := json.Marshal(metrics)
	logger.Metric(string(metricJSON))

	logger.Slog.Info("Processing started")
	logger.Slog.Error("Something went wrong")
	logger.NonStoppingError("Non-critical issue")
	logger.OutputFile("gdc1", 1)
	logger.Summary(duck.RunStatistics{
		Host:   "mixed-test-service",
		Events: 500,
		Bytes:  51200,
		Errors: 2,
	})

	// Wait for all messages
	expectedCount := 7 // state, metric, info, error, non-stopping error, file, summary
	err = tracker.WaitFor(expectedCount, 15*time.Second)
	require.NoError(t, err, "Should receive all messages")

	// Verify all messages
	messages := tracker.GetAll()
	require.Len(t, messages, expectedCount, "Should have exactly 7 messages")

	// Verify message types
	types := []duck.MessageType{
		duck.MessageState,
		duck.MessageMetric,
		duck.MessageInfo,
		duck.MessageError,
		duck.MessageError,
		duck.MessageFile,
		duck.MessageSummary,
	}

	for i, msg := range messages {
		assert.Equal(t, types[i], msg.Type, "Message type mismatch at index %d", i)
		assert.Equal(t, "mixed-test-service", msg.Host)
		assert.Equal(t, 100, msg.RunNumber)

		// Verify StopProcesses flags
		if msg.Type == duck.MessageError && i == 3 { // The Slog.Error message
			assert.True(t, msg.StopProcesses, "Slog.Error should stop processes")
		} else {
			assert.False(t, msg.StopProcesses, "Other messages should not stop processes")
		}
	}
}
