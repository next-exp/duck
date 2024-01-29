package duck

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDuckLogger tests logger creation
func TestNewDuckLogger(t *testing.T) {
	logger := NewDuckLogger("testhost", nil, slog.LevelDebug)

	assert.Equal(t, "testhost", logger.Host)
	assert.Nil(t, logger.Subscription)
	assert.Equal(t, slog.LevelDebug, logger.LogLevel)
	assert.NotNil(t, logger.Slog)
	assert.Equal(t, 0, logger.RunNumber)
}

// TestNewDuckLoggerWithRun tests logger creation with run number
func TestNewDuckLoggerWithRun(t *testing.T) {
	logger := NewDuckLoggerWithRun("testhost", 123, nil, slog.LevelInfo)

	assert.Equal(t, "testhost", logger.Host)
	assert.Nil(t, logger.Subscription)
	assert.Equal(t, slog.LevelInfo, logger.LogLevel)
	assert.NotNil(t, logger.Slog)
	assert.Equal(t, 123, logger.RunNumber)
}

// TestDuckLogger_Metric tests Metric logging
func TestDuckLogger_Metric(t *testing.T) {
	mockPub := &MockPublisher{}

	opts := &slog.HandlerOptions{
		Level: LevelMetrics,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&bytes.Buffer{}, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := DuckLogger{Slog: slog.New(handler), RunNumber: 42}

	metricData := `{"event_counter": 100, "byte_counter": 1024}`
	logger.Metric(metricData)

	// Verify the metric was published
	messages := mockPub.GetPublishedMessages()
	require.Equal(t, 1, len(messages), "Should publish exactly one metric message")

	// Verify message structure
	assert.Equal(t, MessageMetric, messages[0].Type, "Type should be MessageMetric")
	assert.Equal(t, metricData, messages[0].Value, "Value should match the metric JSON")
	assert.Equal(t, "testhost", messages[0].Host, "Host should match")
	assert.Equal(t, 42, messages[0].RunNumber, "RunNumber should match")
	assert.False(t, messages[0].StopProcesses, "Metrics should not stop processes")
}

// TestDuckLogger_NonStoppingError tests NonStoppingError logging
func TestDuckLogger_NonStoppingError(t *testing.T) {
	mockPub := &MockPublisher{}

	opts := &slog.HandlerOptions{
		Level: LevelNonStoppingError,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&bytes.Buffer{}, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := DuckLogger{Slog: slog.New(handler), RunNumber: 42}
	errorMsg := "Test error message"
	logger.NonStoppingError(errorMsg)

	// Verify the error was published
	messages := mockPub.GetPublishedMessages()
	require.Equal(t, 1, len(messages), "Should publish exactly one error message")

	// Verify message structure
	assert.Equal(t, MessageError, messages[0].Type)
	assert.Equal(t, errorMsg, messages[0].Value)
	assert.Equal(t, "testhost", messages[0].Host)
	assert.Equal(t, 42, messages[0].RunNumber)
	assert.False(t, messages[0].StopProcesses, "NonStoppingError should not stop processes")
}

// TestDuckLogger_OutputFile tests OutputFile logging
func TestDuckLogger_OutputFile(t *testing.T) {
	mockPub := &MockPublisher{}

	opts := &slog.HandlerOptions{
		Level: LevelFile,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&bytes.Buffer{}, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := DuckLogger{Slog: slog.New(handler), RunNumber: 42}
	logger.OutputFile("gdc1", 5)

	// Verify the file info was published
	messages := mockPub.GetPublishedMessages()
	require.Equal(t, 1, len(messages), "Should publish exactly one file message")

	// Verify message structure
	assert.Equal(t, MessageFile, messages[0].Type)
	assert.Equal(t, "testhost", messages[0].Host)
	assert.Equal(t, 42, messages[0].RunNumber)

	// Parse and verify file JSON
	var fileData map[string]interface{}
	err := json.Unmarshal([]byte(messages[0].Value), &fileData)
	require.NoError(t, err)
	assert.Equal(t, "gdc1", fileData["server"])
	assert.Equal(t, float64(5), fileData["subrun"])
}

// TestDuckLogger_Summary tests Summary logging
func TestDuckLogger_Summary(t *testing.T) {
	mockPub := &MockPublisher{}

	opts := &slog.HandlerOptions{
		Level: LevelSummary,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&bytes.Buffer{}, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := DuckLogger{Slog: slog.New(handler), RunNumber: 42}

	stats := RunStatistics{
		Host:   "testhost",
		Events: 1000,
		Bytes:  1024000,
	}

	logger.Summary(stats)

	// Verify the summary was published
	messages := mockPub.GetPublishedMessages()
	require.Equal(t, 1, len(messages), "Should publish exactly one summary message")

	// Verify message structure
	assert.Equal(t, MessageSummary, messages[0].Type)
	assert.Equal(t, "testhost", messages[0].Host)
	assert.Equal(t, 42, messages[0].RunNumber)

	// Parse and verify summary JSON
	var summaryData map[string]interface{}
	err := json.Unmarshal([]byte(messages[0].Value), &summaryData)
	require.NoError(t, err)
	assert.Equal(t, "testhost", summaryData["host"])
	assert.Equal(t, float64(1000), summaryData["events"])
	assert.Equal(t, float64(1024000), summaryData["bytes"])
}

// TestAddRunNumberToLogger tests adding run number to existing logger
func TestAddRunNumberToLogger(t *testing.T) {
	mockPub := &MockPublisher{}

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Helper to create a handler with the given run number and publisher
	createHandler := func(runNumber int) *EventidHandler {
		return &EventidHandler{
			Handler:      slog.NewJSONHandler(&buf, opts),
			Host:         "testhost",
			Subscription: nil,
			publisher:    mockPub,
			RunNumber:    runNumber,
		}
	}

	// Create initial logger with run number 0
	logger := DuckLogger{
		Host:         "testhost",
		Subscription: nil,
		Slog:         slog.New(createHandler(0)),
		RunNumber:    0,
	}

	// Log with run number 0
	logger.Slog.Info("run 0")

	// Update to run number 100
	AddRunNumberToLogger(&logger, 100)
	logger.Slog = slog.New(createHandler(100)) // Reattach our mock publisher
	logger.Slog.Info("run 100")

	// Update to run number 200
	AddRunNumberToLogger(&logger, 200)
	logger.Slog = slog.New(createHandler(200)) // Reattach our mock publisher
	logger.Slog.Info("run 200")

	// Update to run number 300
	AddRunNumberToLogger(&logger, 300)
	logger.Slog = slog.New(createHandler(300)) // Reattach our mock publisher
	logger.Slog.Info("run 300")

	// Verify run number field was updated each time
	assert.Equal(t, 300, logger.RunNumber)

	// Verify all messages were published with correct run numbers
	messages := mockPub.GetPublishedMessages()
	require.Equal(t, 4, len(messages), "Should publish 4 messages")

	assert.Equal(t, 0, messages[0].RunNumber, "First message should have run 0")
	assert.Equal(t, 100, messages[1].RunNumber, "Second message should have run 100")
	assert.Equal(t, 200, messages[2].RunNumber, "Third message should have run 200")
	assert.Equal(t, 300, messages[3].RunNumber, "Fourth message should have run 300")
}

// TestMessage_String tests Message string formatting
func TestMessage_String(t *testing.T) {
	msg := Message{
		Host:          "testhost",
		Type:          MessageInfo,
		Value:         "Test message",
		StopProcesses: false,
		RunNumber:     123,
	}

	str := msg.String()

	assert.Contains(t, str, "testhost")
	assert.Contains(t, str, "info")
	assert.Contains(t, str, "Test message")
	assert.Contains(t, str, "r123")
	assert.Contains(t, str, "false")
}

// TestMessageTypes tests all message type constants
func TestMessageTypes(t *testing.T) {
	assert.Equal(t, MessageType("error"), MessageError)
	assert.Equal(t, MessageType("info"), MessageInfo)
	assert.Equal(t, MessageType("debug"), MessageDebug)
	assert.Equal(t, MessageType("metric"), MessageMetric)
	assert.Equal(t, MessageType("state"), MessageState)
	assert.Equal(t, MessageType("output"), MessageFile)
	assert.Equal(t, MessageType("summary"), MessageSummary)
}

// TestLevelNames tests custom level name mapping
func TestLevelNames(t *testing.T) {
	assert.Equal(t, "METRICS", LevelNames[LevelMetrics])
	assert.Equal(t, "STATE", LevelNames[LevelState])
	assert.Equal(t, "ERROR", LevelNames[LevelNonStoppingError])
	assert.Equal(t, "FILE", LevelNames[LevelFile])
	assert.Equal(t, "SUMMARY", LevelNames[LevelSummary])
}

// TestDuckLogger_SlogIntegration tests integration with slog
func TestDuckLogger_SlogIntegration(t *testing.T) {
	mockPub := &MockPublisher{}

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := DuckLogger{Slog: slog.New(handler), RunNumber: 42}

	// Test standard slog methods
	logger.Slog.Debug("Debug message")
	logger.Slog.Info("Info message")
	logger.Slog.Error("Error message")

	// Verify messages were published
	messages := mockPub.GetPublishedMessages()
	assert.Equal(t, 3, len(messages), "Should publish 3 messages")

	assert.Equal(t, MessageDebug, messages[0].Type)
	assert.Equal(t, "Debug message", messages[0].Value)

	assert.Equal(t, MessageInfo, messages[1].Type)
	assert.Equal(t, "Info message", messages[1].Value)

	assert.Equal(t, MessageError, messages[2].Type)
	assert.Equal(t, "Error message", messages[2].Value)
	assert.True(t, messages[2].StopProcesses, "Error should stop processes")

	// Verify logs also passed through to handler (debug, info, error pass through)
	output := buf.String()
	assert.Contains(t, output, "Debug message")
	assert.Contains(t, output, "Info message")
	assert.Contains(t, output, "Error message")
}

// TestDuckLogger_CustomLevels tests custom log levels
func TestDuckLogger_CustomLevels(t *testing.T) {
	assert.Equal(t, slog.Level(9), LevelMetrics)
	assert.Equal(t, slog.Level(10), LevelState)
	assert.Equal(t, slog.Level(11), LevelNonStoppingError)
	assert.Equal(t, slog.Level(12), LevelFile)
	assert.Equal(t, slog.Level(13), LevelSummary)
}

// TestEventidHandler_MetricsFiltering tests that metrics don't go to regular handler
func TestEventidHandler_MetricsFiltering(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: LevelMetrics,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		RunNumber:    100,
	}

	logger := slog.New(handler)

	// Log metrics - should not appear in regular handler
	logger.Log(nil, LevelMetrics, "metric data")

	// Metrics should be filtered out, so buffer should be empty
	output := buf.String()
	assert.Empty(t, output)
}

// TestEventidHandler_StateFiltering tests that state doesn't go to regular handler
func TestEventidHandler_StateFiltering(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: LevelState,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		RunNumber:    100,
	}

	logger := slog.New(handler)

	// Log state - should not appear in regular handler
	logger.Log(nil, LevelState, "state data")

	// State should be filtered out
	output := buf.String()
	assert.Empty(t, output)
}

// TestEventidHandler_FileFiltering tests that file logs don't go to regular handler
func TestEventidHandler_FileFiltering(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: LevelFile,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		RunNumber:    100,
	}

	logger := slog.New(handler)

	// Log file - should not appear in regular handler
	logger.Log(nil, LevelFile, "file data")

	// File logs should be filtered out
	output := buf.String()
	assert.Empty(t, output)
}

// TestEventidHandler_NormalLogsPassThrough tests that normal logs pass through
func TestEventidHandler_NormalLogsPassThrough(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		RunNumber:    100,
	}

	logger := slog.New(handler)

	// Log normal messages - should appear in regular handler
	logger.Info("Test info message")
	logger.Debug("Test debug message")
	logger.Error("Test error message")

	// Normal logs should pass through
	output := buf.String()
	assert.Contains(t, output, "Test info message")
	assert.Contains(t, output, "Test debug message")
	assert.Contains(t, output, "Test error message")
}

// TestMessage_JSONSerialization tests Message JSON serialization
func TestMessage_JSONSerialization(t *testing.T) {
	msg := Message{
		Host:          "testhost",
		Type:          MessageInfo,
		Value:         "Test message",
		StopProcesses: true,
		RunNumber:     999,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.Host, decoded.Host)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.Value, decoded.Value)
	assert.Equal(t, msg.StopProcesses, decoded.StopProcesses)
	assert.Equal(t, msg.RunNumber, decoded.RunNumber)
}

// TestDuckLogger_MetricWithInvalidJSON tests Metric with malformed JSON
func TestDuckLogger_MetricWithInvalidJSON(t *testing.T) {
	logger := NewDuckLogger("testhost", nil, LevelMetrics)

	// This should not panic even with invalid JSON
	logger.Metric(`{invalid json`)

	// Verify logger is still functional
	assert.NotNil(t, logger.Slog)
}

// TestDuckLogger_StateTypes tests all state types
func TestDuckLogger_StateTypes(t *testing.T) {
	states := []struct {
		state    StateType
		expected string
	}{
		{INITIALIZED, "INITIALIZED"},
		{RUNNING, "RUNNING"},
		{STARTING, "STARTING"},
		{STOPPING, "STOPPING"},
		{PINGING, "PINGING"},
	}

	for _, tc := range states {
		assert.Equal(t, tc.expected, tc.state.String())
	}
}

// TestDuckLogger_HostnameInLogs tests that hostname is included in logs
func TestDuckLogger_HostnameInLogs(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "custom-hostname",
		Subscription: nil,
		RunNumber:    42,
	}

	logger := slog.New(handler)
	logger.Info("Test with hostname")

	output := buf.String()
	assert.Contains(t, output, "custom-hostname")
	assert.Contains(t, output, "42")
}

// TestDuckLogger_RunNumberInLogs tests that run number appears in logs
func TestDuckLogger_RunNumberInLogs(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		RunNumber:    12345,
	}

	logger := slog.New(handler)
	logger.Info("Test with run number")

	output := buf.String()
	assert.Contains(t, output, "12345")
	assert.Contains(t, output, "\"run\":12345")
}

// TestMessage_OmitEmptyRunNumber tests that run number is omitted when zero
func TestMessage_OmitEmptyRunNumber(t *testing.T) {
	msg := Message{
		Host:          "testhost",
		Type:          MessageInfo,
		Value:         "Test",
		StopProcesses: false,
		RunNumber:     0,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Run number should be omitted when it's 0 due to omitempty
	dataStr := string(data)
	assert.NotEmpty(t, dataStr)

	// Check that "run":0 is NOT present in the marshaled JSON
	assert.NotContains(t, dataStr, `"run":0`, "Run number should be omitted when zero")

	// Also check that the "run" field is not present at all
	assert.NotContains(t, dataStr, `"run":`, "Run field should not be present when zero")

	// Verify other fields are still present
	assert.Contains(t, dataStr, `"host":"testhost"`)
	assert.Contains(t, dataStr, `"type":"info"`)
	assert.Contains(t, dataStr, `"value":"Test"`)
}

// TestDuckLogger_AllMethodsDontPanic tests that all methods handle nil subscription
func TestDuckLogger_AllMethodsDontPanic(t *testing.T) {
	logger := NewDuckLogger("testhost", nil, slog.LevelDebug)

	// Test all methods don't panic with nil subscription
	assert.NotPanics(t, func() {
		logger.Metric(`{"test": true}`)
		logger.State(RUNNING)
		logger.NonStoppingError("error")
		logger.OutputFile("server", 1)
		logger.Summary(RunStatistics{Host: "test", Events: 100, Bytes: 1000})
		logger.Slog.Info("info")
		logger.Slog.Debug("debug")
		logger.Slog.Error("error")
	})
}

// TestEventidHandler_MessageContent tests complete Message structure in output
func TestEventidHandler_MessageContent(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				levelLabel, exists := LevelNames[level]
				if !exists {
					levelLabel = level.String()
				}
				a.Value = slog.StringValue(levelLabel)
			}
			return a
		},
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "test-host",
		Subscription: nil,
		RunNumber:    42,
	}

	logger := slog.New(handler)
	logger.Info("Test message content")

	// Parse the JSON output to verify structure
	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	// Verify expected fields
	assert.Contains(t, logEntry, "service")
	assert.Equal(t, "test-host", logEntry["service"])
	assert.Contains(t, logEntry, "run")
	assert.Equal(t, float64(42), logEntry["run"])
	assert.Contains(t, logEntry, "msg")
	assert.Equal(t, "Test message content", logEntry["msg"])
	assert.Contains(t, logEntry, "level")
}

// TestDuckLogger_ReplaceAttrLevelNames tests custom level name replacement
func TestDuckLogger_ReplaceAttrLevelNames(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				levelLabel, exists := LevelNames[level]
				if !exists {
					levelLabel = level.String()
				}
				a.Value = slog.StringValue(levelLabel)
			}
			return a
		},
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		RunNumber:    100,
	}

	logger := slog.New(handler)

	// Test only levels that pass through the EventidHandler
	// (Metrics, State, and File are filtered out)
	testCases := []struct {
		level        slog.Level
		expectedName string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	for _, tc := range testCases {
		buf.Reset()
		logger.Log(nil, tc.level, "test message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err, "Level: %s", tc.level)
		assert.Equal(t, tc.expectedName, logEntry["level"], "Level: %s", tc.level)
	}
}

// ==================== NEW COMPREHENSIVE TEST SUITE ====================

// MockPublisher is a mock implementation of MessagePublisher for testing.
type MockPublisher struct {
	mu               sync.Mutex
	publishedMessages []Message
	publishError      error
	callCount         int32
}

// Publish stores the message in memory for test verification.
func (m *MockPublisher) Publish(ctx context.Context, data []byte) error {
	atomic.AddInt32(&m.callCount, 1)
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	m.mu.Lock()
	m.publishedMessages = append(m.publishedMessages, msg)
	m.mu.Unlock()
	return m.publishError
}

// GetPublishedMessages returns a copy of all published messages.
func (m *MockPublisher) GetPublishedMessages() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	messages := make([]Message, len(m.publishedMessages))
	copy(messages, m.publishedMessages)
	return messages
}

// CallCount returns the number of times Publish was called.
func (m *MockPublisher) CallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

// SetPublishError sets an error to be returned by Publish.
func (m *MockPublisher) SetPublishError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishError = err
}

// Clear clears all published messages and resets the call count.
func (m *MockPublisher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedMessages = nil
	m.publishError = nil
	atomic.StoreInt32(&m.callCount, 0)
}

// ==================== PHASE 2: SUBSCRIPTION TESTS ====================

// TestEventidHandler_PublishesMessagesWithSubscription tests that messages are published when subscription exists.
func TestEventidHandler_PublishesMessagesWithSubscription(t *testing.T) {
	mockPub := &MockPublisher{}

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := slog.New(handler)

	// Log at different levels
	logger.Info("info message")
	logger.Error("error message")
	logger.Log(nil, LevelMetrics, `{"metric": "value"}`)

	// Verify messages were published
	messages := mockPub.GetPublishedMessages()
	assert.Equal(t, 3, len(messages), "Should publish 3 messages")

	// Verify first message (info)
	assert.Equal(t, MessageInfo, messages[0].Type)
	assert.Equal(t, "info message", messages[0].Value)
	assert.Equal(t, "testhost", messages[0].Host)
	assert.Equal(t, 42, messages[0].RunNumber)
	assert.False(t, messages[0].StopProcesses)

	// Verify second message (error)
	assert.Equal(t, MessageError, messages[1].Type)
	assert.Equal(t, "error message", messages[1].Value)
	assert.True(t, messages[1].StopProcesses)

	// Verify third message (metric)
	assert.Equal(t, MessageMetric, messages[2].Type)
	assert.Equal(t, `{"metric": "value"}`, messages[2].Value)
}

// TestEventidHandler_PublishError tests that publish errors are handled gracefully.
func TestEventidHandler_PublishError(t *testing.T) {
	mockPub := &MockPublisher{}
	mockPub.SetPublishError(assert.AnError)

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := slog.New(handler)

	// This should not panic even though publishing fails
	logger.Info("info message")

	// Verify publish was called
	assert.Equal(t, 1, mockPub.CallCount())

	// Verify log was still written to buffer
	output := buf.String()
	assert.Contains(t, output, "info message")
}

// ==================== PHASE 3: LEVEL MAPPING TESTS ====================

// TestEventidHandler_LevelMappingFunctional functionally tests level to message type mapping.
// This replaces the tautological TestEventidHandler_LevelMapping test.
func TestEventidHandler_LevelMappingFunctional(t *testing.T) {
	tests := []struct {
		name          string
		level         slog.Level
		expectedType  MessageType
		stopProcesses bool
		logValue      string
	}{
		{"Metrics", LevelMetrics, MessageMetric, false, `{"counter": 123}`},
		{"State", LevelState, MessageState, false, `{"state": "RUNNING"}`},
		{"File", LevelFile, MessageFile, false, `{"server": "gdc1", "subrun": 1}`},
		{"Summary", LevelSummary, MessageSummary, false, `{"events": 1000}`},
		{"Debug", slog.LevelDebug, MessageDebug, false, "debug message"},
		{"Info", slog.LevelInfo, MessageInfo, false, "info message"},
		{"Error", slog.LevelError, MessageError, true, "error message"},
		{"NonStoppingError", LevelNonStoppingError, MessageError, false, "non-stopping error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPub := &MockPublisher{}

			handler := &EventidHandler{
				Handler:      slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}),
				Host:         "testhost",
				Subscription: nil,
				publisher:    mockPub,
				RunNumber:    42,
			}

			logger := slog.New(handler)
			logger.Log(nil, tt.level, tt.logValue)

			messages := mockPub.GetPublishedMessages()
			require.Equal(t, 1, len(messages), "Should publish exactly one message")

			assert.Equal(t, tt.expectedType, messages[0].Type, "Message type should match expected type")
			assert.Equal(t, tt.stopProcesses, messages[0].StopProcesses, "StopProcesses flag should match")
			assert.Equal(t, tt.logValue, messages[0].Value, "Log value should match")
			assert.Equal(t, "testhost", messages[0].Host, "Host should match")
			assert.Equal(t, 42, messages[0].RunNumber, "RunNumber should match")
		})
	}
}

// ==================== PHASE 4: JSON MARSHALLING TESTS ====================

// TestDuckLogger_AllStatesMarshaling tests all state types are properly marshaled.
func TestDuckLogger_AllStatesMarshaling(t *testing.T) {
	states := []StateType{INITIALIZED, RUNNING, STARTING, STOPPING, PINGING}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			mockPub := &MockPublisher{}

			opts := &slog.HandlerOptions{
				Level: LevelState,
			}

			handler := &EventidHandler{
				Handler:      slog.NewJSONHandler(&bytes.Buffer{}, opts),
				Host:         "testhost",
				Subscription: nil,
				publisher:    mockPub,
				RunNumber:    1,
			}

			logger := DuckLogger{Slog: slog.New(handler), RunNumber: 1}
			logger.State(state)

			// Verify the message was published
			messages := mockPub.GetPublishedMessages()
			require.Equal(t, 1, len(messages))

			// Parse and verify state structure
			var stateData map[string]interface{}
			err := json.Unmarshal([]byte(messages[0].Value), &stateData)
			require.NoError(t, err)

			assert.Equal(t, state.String(), stateData["state"])
		})
	}
}

// TestDuckLogger_SummaryMarshalingInLog tests that Summary() properly marshals statistics.
func TestDuckLogger_SummaryMarshalingInLog(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: LevelSummary,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		RunNumber:    42,
	}

	logger := DuckLogger{Slog: slog.New(handler), RunNumber: 42}

	stats := RunStatistics{
		Host:   "testhost",
		Events: 12345,
		Bytes:  67890,
	}

	logger.Summary(stats)

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Extract and parse the summary JSON
	msgStr, ok := logEntry["msg"].(string)
	require.True(t, ok)

	var summaryData map[string]interface{}
	err = json.Unmarshal([]byte(msgStr), &summaryData)
	require.NoError(t, err)

	// Verify summary structure
	assert.Equal(t, "testhost", summaryData["host"])
	assert.Equal(t, float64(12345), summaryData["events"])
	assert.Equal(t, float64(67890), summaryData["bytes"])
}

// ==================== PHASE 5: FATALF TESTS ====================

// TestDuckLogger_FatalExitsWithCode1 tests that Fatalf exits with code 1.
func TestDuckLogger_FatalExitsWithCode1(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		logger := NewDuckLogger("test-host", nil, slog.LevelError)
		logger.Fatalf("Critical error: %s", "connection failed")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestDuckLogger_FatalExitsWithCode1")
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")

	err := cmd.Run()

	require.Error(t, err, "Fatalf should cause process to exit")

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok, "Error should be ExitError")
	assert.Equal(t, 1, exitErr.ExitCode(), "Exit code should be 1")
}

// ==================== PHASE 6: THREAD-SAFETY TESTS ====================

// TestEventidHandler_ConcurrentLevelMapping tests concurrent logging at different levels.
func TestEventidHandler_ConcurrentLevelMapping(t *testing.T) {
	mockPub := &MockPublisher{}

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := slog.New(handler)

	levels := []slog.Level{
		LevelMetrics,
		LevelState,
		LevelFile,
		LevelSummary,
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelError,
		LevelNonStoppingError,
	}

	// Launch multiple goroutines logging at different levels
	done := make(chan bool, len(levels)*10)
	for i := 0; i < 10; i++ {
		for _, level := range levels {
			go func(l slog.Level) {
				logger.Log(nil, l, "test message")
				done <- true
			}(level)
		}
	}

	// Wait for all goroutines
	for i := 0; i < len(levels)*10; i++ {
		<-done
	}

	// Verify all messages were published correctly
	assert.Equal(t, len(levels)*10, mockPub.CallCount())

	messages := mockPub.GetPublishedMessages()
	assert.Equal(t, len(levels)*10, len(messages))

	// Verify all messages have correct structure
	for _, msg := range messages {
		assert.Equal(t, "testhost", msg.Host)
		assert.Equal(t, 42, msg.RunNumber)
		assert.NotEmpty(t, msg.Type)
		assert.NotEmpty(t, msg.Value)
	}
}

// TestEventidHandler_ConcurrentRunNumberUpdates tests concurrent run number updates.
func TestEventidHandler_ConcurrentRunNumberUpdates(t *testing.T) {
	// Launch multiple goroutines creating loggers with different run numbers
	done := make(chan bool, 20)
	publishers := make([]*MockPublisher, 20)

	for i := 0; i < 20; i++ {
		go func(id int) {
			mockPub := &MockPublisher{}

			var buf bytes.Buffer
			opts := &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}

			handler := &EventidHandler{
				Handler:      slog.NewJSONHandler(&buf, opts),
				Host:         "testhost",
				Subscription: nil,
				publisher:    mockPub,
				RunNumber:    id,
			}

			logger := slog.New(handler)
			logger.Info("test message")

			publishers[id] = mockPub
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify each publisher has the correct run number
	for i, mockPub := range publishers {
		if mockPub == nil {
			continue
		}
		messages := mockPub.GetPublishedMessages()
		require.Equal(t, 1, len(messages))
		assert.Equal(t, i, messages[0].RunNumber)
	}
}

// TestDuckLogger_ConcurrentMethodAccess tests concurrent access to all DuckLogger methods.
func TestDuckLogger_ConcurrentMethodAccess(t *testing.T) {
	mockPub := &MockPublisher{}

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := DuckLogger{
		Slog:      slog.New(handler),
		RunNumber: 42,
	}

	// Launch multiple goroutines calling different methods concurrently
	done := make(chan bool, 50)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Metric(`{"counter": 1}`)
			done <- true
		}(i)
		go func(id int) {
			logger.State(RUNNING)
			done <- true
		}(i)
		go func(id int) {
			logger.NonStoppingError("error")
			done <- true
		}(i)
		go func(id int) {
			logger.OutputFile("gdc1", id)
			done <- true
		}(i)
		go func(id int) {
			logger.Summary(RunStatistics{Host: "test", Events: int64(id * 100), Bytes: int64(id * 1000)})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Verify all 50 messages were published
	messages := mockPub.GetPublishedMessages()
	assert.Equal(t, 50, len(messages), "Should have exactly 50 messages")

	// Verify message types
	typeCounts := map[MessageType]int{}
	for _, msg := range messages {
		typeCounts[msg.Type]++

		// Verify all messages have correct structure
		assert.Equal(t, "testhost", msg.Host)
		assert.Equal(t, 42, msg.RunNumber)
		assert.NotEmpty(t, msg.Value)
	}

	// Verify we have the expected count of each type
	assert.Equal(t, 10, typeCounts[MessageMetric], "Should have 10 metric messages")
	assert.Equal(t, 10, typeCounts[MessageState], "Should have 10 state messages")
	assert.Equal(t, 10, typeCounts[MessageError], "Should have 10 error messages (NonStoppingError)")
	assert.Equal(t, 10, typeCounts[MessageFile], "Should have 10 file messages")
	assert.Equal(t, 10, typeCounts[MessageSummary], "Should have 10 summary messages")
}

// TestDuckLogger_ConcurrentStateChanges tests concurrent state changes.
func TestDuckLogger_ConcurrentStateChanges(t *testing.T) {
	mockPub := &MockPublisher{}

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: LevelState,
	}

	handler := &EventidHandler{
		Handler:      slog.NewJSONHandler(&buf, opts),
		Host:         "testhost",
		Subscription: nil,
		publisher:    mockPub,
		RunNumber:    42,
	}

	logger := DuckLogger{
		Slog:      slog.New(handler),
		RunNumber: 42,
	}

	states := []StateType{INITIALIZED, RUNNING, STARTING, STOPPING, PINGING}

	// Launch multiple goroutines changing state concurrently
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(id int) {
			state := states[id%len(states)]
			logger.State(state)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Verify all state changes were logged
	assert.Equal(t, 50, mockPub.CallCount())

	messages := mockPub.GetPublishedMessages()
	assert.Equal(t, 50, len(messages))

	// Verify all are state messages
	for _, msg := range messages {
		assert.Equal(t, MessageState, msg.Type)
		assert.Equal(t, "testhost", msg.Host)
		assert.Equal(t, 42, msg.RunNumber)
	}
}

