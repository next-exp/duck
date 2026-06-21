package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"connectrpc.com/connect"
	centrifuge "github.com/centrifugal/centrifuge-go"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/control"
	pbconnect "github.com/jmbenlloch/next_duck/rpc/control/controlconnect"
)

type server struct {
	ctx              context.Context
	cancelCtx        context.CancelFunc
	configFilename   string
	ldcConfiguration *duck.LDCConfiguration
	ldcName          string // Name from command line flag

	// Thread-safe state management
	state      duck.StateType
	stateMutex sync.RWMutex
	ctxMutex   sync.RWMutex // Protects ctx and cancelCtx fields
	logger     duck.DuckLogger
	metrics    *MetricsRegistry

	// Singleton ldcServer control
	runActive     bool
	runActiveCond *sync.Cond
}

// Thread-safe state access
func (s *server) getState() duck.StateType {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	return s.state
}

// Thread-safe context access
// Uses ctxMutex for synchronization if context was set via StartRun/StopRun.
// Also works correctly if tests set s.ctx directly without using the mutex.
func (s *server) getContext() context.Context {
	s.ctxMutex.RLock()
	ctx := s.ctx
	s.ctxMutex.RUnlock()
	return ctx
}

func (s *server) setState(newState duck.StateType) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	s.state = newState
	s.logger.State(newState)
}

// stopOnLocalError handles errors that occur during data processing.
// It cancels the context to stop all goroutines and signals ldcServer to exit.
// The error is logged to Centrifuge so API can coordinate system-wide stop.
// This is idempotent - safe to call multiple times.
func (s *server) stopOnLocalError(errMsg string) {
	s.logger.Slog.Error(errMsg)

	// Cancel context to stop local goroutines
	s.ctxMutex.Lock()
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	s.ctxMutex.Unlock()

	// Signal ldcServer to exit its wait loop and return to INITIALIZED
	s.runActiveCond.L.Lock()
	s.runActive = false
	s.runActiveCond.Broadcast()
	s.runActiveCond.L.Unlock()
}

func (s *server) StartRun(ctx context.Context, in *connect.Request[pb.DuckRequest]) (*connect.Response[pb.DuckReply], error) {
	s.stateMutex.Lock()

	if s.state != duck.INITIALIZED {
		s.stateMutex.Unlock()
		return connect.NewResponse(&pb.DuckReply{Message: "Cannot start before stop"}), nil
	}

	// Setup context (protected by ctxMutex to prevent data race with changeStateOnStop)
	s.ctxMutex.Lock()
	s.ctx = context.Background()
	s.ctx, s.cancelCtx = context.WithCancel(s.ctx)
	s.ctx = context.WithValue(s.ctx, "cancelCtx", s.cancelCtx)
	s.ctx = context.WithValue(s.ctx, "server", s)
	s.ctxMutex.Unlock()

	// Reset counters
	s.metrics.Reset()

	// Transition state (still under lock to prevent concurrent StartRun calls)
	s.state = duck.STARTING
	s.logger.State(duck.STARTING)

	s.stateMutex.Unlock()

	// Signal ldcServer singleton to start accepting connections
	s.runActiveCond.L.Lock()
	s.runActive = true
	s.runActiveCond.Broadcast()
	s.runActiveCond.L.Unlock()

	return connect.NewResponse(&pb.DuckReply{Message: "Started"}), nil
}

func (s *server) StopRun(ctx context.Context, in *connect.Request[pb.DuckRequest]) (*connect.Response[pb.DuckReply], error) {
	s.stateMutex.Lock()

	// Already stopped - return success (idempotent, handles local error recovery)
	if s.state == duck.INITIALIZED {
		s.stateMutex.Unlock()
		return connect.NewResponse(&pb.DuckReply{Message: "Stopped"}), nil
	}

	if s.state != duck.RUNNING && s.state != duck.STARTING {
		s.stateMutex.Unlock()
		return connect.NewResponse(&pb.DuckReply{Message: "Can not stop before start"}), nil
	}

	// Transition state to STOPPING while cleanup happens
	s.state = duck.STOPPING
	s.logger.State(duck.STOPPING)

	s.stateMutex.Unlock()

	// Cancel context (protected by ctxMutex to prevent data race)
	s.ctxMutex.Lock()
	s.cancelCtx()
	s.ctxMutex.Unlock()

	// Signal ldcServer singleton to stop accepting connections
	s.runActiveCond.L.Lock()
	s.runActive = false
	s.runActiveCond.Broadcast()
	s.runActiveCond.L.Unlock()

	return connect.NewResponse(&pb.DuckReply{Message: "Stopped"}), nil
}

func (s *server) GetState(ctx context.Context, in *connect.Request[pb.DuckRequest]) (*connect.Response[pb.DuckReply], error) {
	s.logger.State(s.getState())
	return connect.NewResponse(&pb.DuckReply{Message: s.getState().String()}), nil
}

func (s *server) GetRunStatistics(ctx context.Context, in *connect.Request[pb.DuckRequest]) (*connect.Response[pb.RunStatisticsReply], error) {
	response := pb.RunStatisticsReply{
		Events: s.metrics.GetEventCounter(),
		Bytes:  s.metrics.GetBytesCounter(),
		Errors: s.metrics.GetPacketErrorCounter(),
	}
	return connect.NewResponse(&response), nil
}

func (s *server) PingDevices(ctx context.Context, in *connect.Request[pb.DuckRequest]) (*connect.Response[pb.PingReply], error) {
	s.setState(duck.PINGING)

	// Re-read configuration to get latest equipment settings
	configuration, err := duck.ReadConfiguration(s.configFilename, nil, nil)
	if err != nil {
		s.logger.Slog.Error(fmt.Sprintf("Error reading configuration: %s", err.Error()))
		s.setState(duck.INITIALIZED)
		return connect.NewResponse(&pb.PingReply{Success: false}), nil
	}
	ldcConfiguration, err := duck.GetLDCConfiguration(configuration.LDCs, s.ldcName)
	if err != nil {
		s.logger.Slog.Error(fmt.Sprintf("Error reading LDC configuration: %s", err.Error()))
		s.setState(duck.INITIALIZED)
		return connect.NewResponse(&pb.PingReply{Success: false}), nil
	}

	factory := &ProbingFactory{}
	success := pingDevices(s, factory, ldcConfiguration.Equipments)
	s.setState(duck.INITIALIZED)
	return connect.NewResponse(&pb.PingReply{Success: success}), nil
}

func main() {
	logger := duck.NewDuckLogger("", nil, slog.LevelInfo)
	configFilename := flag.String("config", "", "Configuration file path")
	ldcName := flag.String("name", "", "LDC name")
	flag.Parse()

	configuration, err := duck.ReadConfiguration(*configFilename, nil, nil)
	if err != nil {
		message := fmt.Errorf("error reading configuration %w", err)
		logger.Fatalf(message.Error())
	}

	ldcConfiguration, err := duck.GetLDCConfiguration(configuration.LDCs, *ldcName)
	if err != nil {
		message := fmt.Errorf("error reading LDC configuration %w", err)
		logger.Fatalf(message.Error())
	}
	logger = duck.NewDuckLogger(ldcConfiguration.Name, nil, slog.LevelInfo)

	configFile, err := duck.ReadConfigurationFile(*configFilename)
	if err != nil {
		message := fmt.Errorf("error reading configuration %w", err)
		logger.Fatalf(message.Error())
		return
	}
	var sub *centrifuge.Subscription
	var errSub error
	if configFile.Centrifugal.Port != 0 { // allow disabling Centrifuge in tests
		sub, errSub = duck.CreateNewSubscription(configFile.Centrifugal, ldcConfiguration.Name)
		if errSub != nil {
			message := fmt.Errorf("error starting centrifuge: %w", errSub)
			logger.Fatalf(message.Error())
		}
	}
	logger = duck.NewDuckLogger(ldcConfiguration.Name, sub, configFile.LogLevel)

	// Create metrics registry
	metrics := NewMetricsRegistry()

	// Start ConnectRPC listener (HTTP server)
	// Note: GRPCPort field name is historical - this is actually an HTTP/ConnectRPC port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", ldcConfiguration.GRPCPort))
	if err != nil {
		logger.Fatalf("Error listening on ConnectRPC port: %v", err)
	}

	startPrometheus(metrics, ldcConfiguration.PrometheusPort, configFile.GoStats, logger)

	s := &server{
		ldcConfiguration: ldcConfiguration,
		configFilename:   *configFilename,
		ldcName:          *ldcName,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Start ldcServer as singleton - runs once for the lifetime of the process
	go ldcServer(s, s.configFilename)

	mux := http.NewServeMux()
	path, handler := pbconnect.NewRunControlHandler(s)
	mux.Handle(path, handler)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", ldcConfiguration.GRPCPort),
		Handler: mux,
	}
	err = httpServer.Serve(lis)
	if err != nil {
		message := fmt.Errorf("failed to serve ConnectRPC server: %w", err)
		logger.Slog.Error(message.Error())
	}
}
