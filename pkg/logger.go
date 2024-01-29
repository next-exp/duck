package duck

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/centrifugal/centrifuge-go"
)

type MessageType string

const (
	MessageError   MessageType = "error"
	MessageInfo    MessageType = "info"
	MessageDebug   MessageType = "debug"
	MessageMetric  MessageType = "metric"
	MessageState   MessageType = "state"
	MessageFile    MessageType = "output"
	MessageSummary MessageType = "summary"
)

type Message struct {
	Timestamp     time.Time   `json:"timestamp"`
	Host          string      `json:"host"`
	Type          MessageType `json:"type"`
	Value         string      `json:"value"`
	StopProcesses bool        `json:"stop_processes"`
	RunNumber     int         `json:"run,omitempty"`
}

func (m *Message) String() string {
	return fmt.Sprintf("%s %s r%d %s %s %t", m.Timestamp.Format(time.RFC3339), m.Host, m.RunNumber, m.Type, m.Value, m.StopProcesses)
}

type DuckLogger struct {
	Host         string
	Subscription *centrifuge.Subscription
	publisher    MessagePublisher // Interface for testing
	LogLevel     slog.Level
	Slog         *slog.Logger
	RunNumber    int
}

func (l DuckLogger) Metric(message string) {
	ctx := context.Background()
	l.Slog.Log(ctx, LevelMetrics, message)
}

func (l DuckLogger) State(state StateType) {
	ctx := context.Background()
	stateData := State{State: state.String()}
	message, err := json.Marshal(stateData)
	if err != nil {
		message := fmt.Errorf("error marshalling state: %w", err)
		l.Slog.Error(message.Error())
	}
	l.Slog.Log(ctx, LevelState, string(message))
}

func (l DuckLogger) NonStoppingError(message string) {
	ctx := context.Background()
	l.Slog.Log(ctx, LevelNonStoppingError, string(message))
}

func (l DuckLogger) OutputFile(server string, subrun int) {
	ctx := context.Background()
	fileData := OutputFile{
		Server: server,
		Subrun: subrun,
	}
	message, err := json.Marshal(fileData)
	if err != nil {
		message := fmt.Errorf("error marshalling state: %w", err)
		l.Slog.Error(message.Error())
	}
	l.Slog.Log(ctx, LevelFile, string(message))
}

func (l DuckLogger) Summary(data RunStatistics) {
	ctx := context.Background()
	message, err := json.Marshal(data)
	if err != nil {
		message := fmt.Errorf("error marshalling state: %w", err)
		l.Slog.Error(message.Error())
	}
	l.Slog.Log(ctx, LevelSummary, string(message))
}

func (l DuckLogger) Fatalf(message string, args ...interface{}) {
	l.Slog.Error(message, args...)
	os.Exit(1)
}

func AddRunNumberToLogger(logger *DuckLogger, run int) {
	newLogger := NewDuckLoggerWithRun(logger.Host, run, logger.Subscription, logger.LogLevel)
	logger.Slog = newLogger.Slog
	logger.RunNumber = run
}

func NewDuckLoggerWithRun(hostname string, run int,
	subscription *centrifuge.Subscription, logLevel slog.Level) DuckLogger {
	var publisher MessagePublisher
	if subscription != nil {
		publisher = &CentrifugePublisher{Sub: subscription}
	}

	duckLogger := DuckLogger{
		Host:         hostname,
		Subscription: subscription,
		publisher:    publisher,
		LogLevel:     logLevel,
		RunNumber:    run,
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
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

	var loghandler slog.Handler = &EventidHandler{
		Handler:      slog.NewJSONHandler(os.Stdout, opts),
		Host:         duckLogger.Host,
		Subscription: duckLogger.Subscription,
		publisher:    publisher,
		RunNumber:    run,
	}

	duckLogger.Slog = slog.New(loghandler)

	return duckLogger
}

func NewDuckLogger(hostname string, subscription *centrifuge.Subscription, logLevel slog.Level) DuckLogger {
	return NewDuckLoggerWithRun(hostname, 0, subscription, logLevel)
}

////////////////////
// slog functions //
////////////////////

const (
	LevelMetrics          = slog.Level(9)
	LevelState            = slog.Level(10)
	LevelNonStoppingError = slog.Level(11)
	LevelFile             = slog.Level(12)
	LevelSummary          = slog.Level(13)
)

var LevelNames = map[slog.Leveler]string{
	LevelMetrics:          "METRICS",
	LevelState:            "STATE",
	LevelNonStoppingError: "ERROR",
	LevelFile:             "FILE",
	LevelSummary:          "SUMMARY",
}

type EventidHandler struct {
	slog.Handler
	Host         string
	Subscription *centrifuge.Subscription
	publisher    MessagePublisher // Interface for testing
	RunNumber    int
}

func (h *EventidHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Add(slog.String("service", h.Host))
	r.Add(slog.Int("run", h.RunNumber))
	var messageType MessageType
	stopProcesses := false
	switch r.Level {
	case LevelMetrics:
		messageType = MessageMetric
	case LevelState:
		messageType = MessageState
	case LevelFile:
		messageType = MessageFile
	case LevelSummary:
		messageType = MessageSummary
	case slog.LevelDebug:
		messageType = MessageDebug
	case slog.LevelInfo:
		messageType = MessageInfo
	case slog.LevelError:
		messageType = MessageError
		stopProcesses = true
	case LevelNonStoppingError:
		messageType = MessageError
	}
	messageData := Message{
		Timestamp:     r.Time,
		Host:          h.Host,
		Type:          messageType,
		Value:         r.Message,
		StopProcesses: stopProcesses,
		RunNumber:     h.RunNumber,
	}
	if h.publisher != nil {
		PublishMessage(h.publisher, messageData)
	}

	// Do not process metrics in the regular handler
	if r.Level == LevelMetrics || r.Level == LevelState || r.Level == LevelFile {
		return nil
	}

	return h.Handler.Handle(ctx, r)
}
