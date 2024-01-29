package testhelpers

import (
	"bytes"
	"log/slog"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// SetupTest initializes the logger for tests and returns it
func SetupTest() duck.DuckLogger {
	return duck.NewDuckLogger("test", nil, slog.LevelError)
}

// SetupTestWithCapture creates a logger that writes to a buffer for log verification
func SetupTestWithCapture() (duck.DuckLogger, *bytes.Buffer) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	})
	testLogger := duck.DuckLogger{
		Host:      "test",
		Slog:      slog.New(handler),
		RunNumber: 0,
	}
	return testLogger, &buf
}
