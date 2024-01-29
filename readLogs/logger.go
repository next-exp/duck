package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/centrifugal/centrifuge-go"
	duck "github.com/jmbenlloch/next_duck/pkg"
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
}

func (m *Message) String() string {
	return fmt.Sprintf("%s %s %s %s %t", m.Timestamp.Format(time.RFC3339), m.Host, m.Type, m.Value, m.StopProcesses)
}

type DuckLogger struct {
	Host         string
	Subscription *centrifuge.Subscription
	Slog         *slog.Logger
	BaseSlog     *slog.Logger
	RunNumber    int
}

func (l DuckLogger) Metric(message string) {
	ctx := context.Background()
	l.Slog.Log(ctx, LevelMetrics, message)
}

func (l DuckLogger) State(state duck.StateType) {
	ctx := context.Background()
	stateData := duck.State{State: state.String()}
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
	fileData := duck.OutputFile{
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

func (l DuckLogger) Summary(data duck.RunStatistics) {
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

func AddHostnameToLogger(logger *DuckLogger, host string) {
	logger.Host = host
	logger.Slog = logger.Slog.With(slog.String("service", host))
}

func AddRunNumberToLogger(logger *DuckLogger, run int) {
	logger.RunNumber = run
	if logger.BaseSlog == nil {
		logger.BaseSlog = logger.Slog
	}
	logger.Slog = logger.BaseSlog.With(slog.Int("run", run))
}

func NewDuckLogger(hostname string, subscription *centrifuge.Subscription, logLevel slog.Level) DuckLogger {
	duckLogger := DuckLogger{Host: hostname, Subscription: subscription}

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
	}

	duckLogger.Slog = slog.New(loghandler)
	return duckLogger
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
}

func (h *EventidHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Add(slog.String("service", h.Host))
	//var messageType MessageType
	//stopProcesses := false
	//switch r.Level {
	//case LevelMetrics:
	//	messageType = MessageMetric
	//case LevelState:
	//	messageType = MessageState
	//case LevelFile:
	//	messageType = MessageFile
	//case LevelSummary:
	//	messageType = MessageSummary
	//case slog.LevelDebug:
	//	messageType = MessageDebug
	//case slog.LevelInfo:
	//	messageType = MessageInfo
	//case slog.LevelError:
	//	messageType = MessageError
	//	stopProcesses = true
	//case LevelNonStoppingError:
	//	messageType = MessageError
	//}
	//messageData := Message{
	//	Timestamp:     r.Time,
	//	Host:          h.Host,
	//	Type:          messageType,
	//	Value:         r.Message,
	//	StopProcesses: stopProcesses,
	//}
	//if h.Subscription != nil {
	//	PublishMessage(h.Subscription, messageData)
	//}

	// Do not process metrics in the regular handler
	if r.Level == LevelMetrics || r.Level == LevelState || r.Level == LevelFile {
		return nil
	}

	return h.Handler.Handle(ctx, r)
}
