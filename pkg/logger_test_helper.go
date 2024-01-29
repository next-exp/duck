package duck

import (
	"bytes"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// CreateTestLoggerWithCapture creates a logger that writes JSON to a buffer for verification
func CreateTestLoggerWithCapture(t *testing.T) (DuckLogger, *bytes.Buffer) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	testLogger := DuckLogger{
		Host:      "test",
		Slog:      slog.New(handler),
		RunNumber: 0,
	}
	return testLogger, &buf
}

// CreateTestLoggerWithCaptureLevel creates a logger with specified log level
func CreateTestLoggerWithCaptureLevel(t *testing.T, level slog.Level) (DuckLogger, *bytes.Buffer) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level})
	testLogger := DuckLogger{
		Host:      "test",
		Slog:      slog.New(handler),
		RunNumber: 0,
	}
	return testLogger, &buf
}

// MockTickerProvider provides controllable time for testing
type MockTickerProvider struct {
	TickChan chan time.Time
	stopOnce sync.Once
}

// NewMockTickerProvider creates a new mock ticker provider
func NewMockTickerProvider() *MockTickerProvider {
	return &MockTickerProvider{
		TickChan: make(chan time.Time, 100),
	}
}

// NewTicker creates a ticker that uses the mock tick channel
func (m *MockTickerProvider) NewTicker(d time.Duration) *time.Ticker {
	return &time.Ticker{C: m.TickChan}
}

// Tick triggers a ticker tick at the specified time
func (m *MockTickerProvider) Tick(t time.Time) {
	m.TickChan <- t
}

