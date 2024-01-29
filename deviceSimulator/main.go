package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/control"
	pbconnect "github.com/jmbenlloch/next_duck/rpc/control/controlconnect"
)

var (
	state      duck.StateType
	logger     duck.DuckLogger
	dataSource DataSource // Can be *Replayer or *Generator
)

// server implements the RunControl service
type server struct {
	mode               string  // "replay" or "generate"
	equipmentID        int
	filePath           string
	rate               float64
	loop               bool
	maxEvents          int
	fragmentSize       int
	packetsPerEvent    int
	errorInjectionRate float64
}

func (s *server) StartRun(ctx context.Context, req *connect.Request[pb.DuckRequest]) (*connect.Response[pb.DuckReply], error) {
	if state != duck.INITIALIZED {
		return connect.NewResponse(&pb.DuckReply{Message: "Already running"}), nil
	}

	state = duck.STARTING
	logger.State(state)

	// Get database connection
	db, err := GetDatabaseConnection()
	if err != nil {
		logger.Slog.Error("Failed to connect to database", "error", err)
		state = duck.INITIALIZED
		return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Database error: %v", err)}), nil
	}
	defer db.Close()

	// Create data source based on mode
	switch s.mode {
	case "generate":
		// Build generator configuration
		config, err := BuildGeneratorConfig(db, s.equipmentID, s.rate, s.packetsPerEvent, s.errorInjectionRate, s.maxEvents, s.fragmentSize)
		if err != nil {
			logger.Slog.Error("Failed to build generator config", "error", err)
			state = duck.INITIALIZED
			return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Config error: %v", err)}), nil
		}

		// Create generator
		generator, err := NewGenerator(config)
		if err != nil {
			logger.Slog.Error("Failed to create generator", "error", err)
			state = duck.INITIALIZED
			return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Generator error: %v", err)}), nil
		}

		// Start generator
		if err := generator.Start(); err != nil {
			logger.Slog.Error("Failed to start generator", "error", err)
			state = duck.INITIALIZED
			return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Start error: %v", err)}), nil
		}

		dataSource = generator

	default: // "replay"
		// Build replay configuration
		config, err := BuildReplayConfig(db, s.equipmentID, s.filePath, s.rate, s.loop, s.maxEvents, s.fragmentSize)
		if err != nil {
			logger.Slog.Error("Failed to build replay config", "error", err)
			state = duck.INITIALIZED
			return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Config error: %v", err)}), nil
		}

		// Create replayer
		replayer, err := NewReplayer(config)
		if err != nil {
			logger.Slog.Error("Failed to create replayer", "error", err)
			state = duck.INITIALIZED
			return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Replayer error: %v", err)}), nil
		}

		// Start replay
		if err := replayer.Start(); err != nil {
			logger.Slog.Error("Failed to start replay", "error", err)
			state = duck.INITIALIZED
			return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Start error: %v", err)}), nil
		}

		dataSource = replayer
	}

	state = duck.RUNNING
	logger.State(state)

	// Reset metrics
	eventCounter.Set(0)
	packetCounter.Set(0)
	bytesCounter.Set(0)
	errorCounter.Set(0)
	errorInjectionCounter.Set(0)

	// Start metrics updater
	go updateMetrics()

	return connect.NewResponse(&pb.DuckReply{Message: fmt.Sprintf("Started %s", s.mode)}), nil
}

func (s *server) StopRun(ctx context.Context, req *connect.Request[pb.DuckRequest]) (*connect.Response[pb.DuckReply], error) {
	if state == duck.INITIALIZED {
		return connect.NewResponse(&pb.DuckReply{Message: "Not running"}), nil
	}

	state = duck.STOPPING
	logger.State(state)

	if dataSource != nil {
		if err := dataSource.Stop(); err != nil {
			logger.Slog.Error("Error stopping data source", "error", err)
		}
	}

	state = duck.INITIALIZED
	logger.State(state)

	return connect.NewResponse(&pb.DuckReply{Message: "Stopped"}), nil
}

func (s *server) GetState(ctx context.Context, req *connect.Request[pb.DuckRequest]) (*connect.Response[pb.DuckReply], error) {
	logger.State(state)
	return connect.NewResponse(&pb.DuckReply{Message: state.String()}), nil
}

func (s *server) GetRunStatistics(ctx context.Context, req *connect.Request[pb.DuckRequest]) (*connect.Response[pb.RunStatisticsReply], error) {
	response := pb.RunStatisticsReply{
		Events: int64(getEventCounter()),
		Bytes:  int64(getBytesCounter()),
		Errors: int64(getErrorCounter()),
	}
	return connect.NewResponse(&response), nil
}

func (s *server) PingDevices(ctx context.Context, req *connect.Request[pb.DuckRequest]) (*connect.Response[pb.PingReply], error) {
	// For simulator, always return success
	return connect.NewResponse(&pb.PingReply{Success: true}), nil
}

// updateMetrics periodically updates prometheus metrics from data source
func updateMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if dataSource == nil || dataSource.GetState() == SourceStateStopped {
			return
		}

		metrics := dataSource.GetMetrics()
		eventCounter.Set(float64(metrics.EventsSent))
		packetCounter.Set(float64(metrics.PacketsSent))
		bytesCounter.Set(float64(metrics.BytesSent))
		errorCounter.Set(float64(metrics.Errors))
		errorInjectionCounter.Set(float64(metrics.ErrorsInjected))
	}
}

func main() {
	// CLI flags
	mode := flag.String("mode", "replay", "Operation mode: 'replay' (replay from file) or 'generate' (random data)")
	configFilename := flag.String("config", "", "Configuration file path (YAML)")
	equipmentID := flag.Int("id", 0, "Equipment ID")
	filePath := flag.String("file", "", "Path to .rd dump file (replay mode)")
	rate := flag.Float64("rate", 0, "Absolute event rate in Hz (0 = use database config)")
	loop := flag.Bool("loop", false, "Loop replay continuously (replay mode)")
	maxEvents := flag.Int("max-events", 0, "Maximum events to send (0 = unlimited)")
	fragmentSize := flag.Int("fragment-size", 0, "UDP fragment/packet size in bytes (0 = use database config, default: 7992)")
	packetsPerEvent := flag.Int("packets-per-event", 0, "Packets per event (generate mode, 0 = use database config)")
	errorInjection := flag.Float64("error-injection", -1, "Error injection rate 0.0-1.0 (generate mode, -1 = use database config)")
	serverMode := flag.Bool("server", true, "Run as ConnectRPC server")
	grpcPort := flag.Int("grpc-port", 0, "ConnectRPC port (default: 50070 + equipmentID - 1)")
	prometheusPort := flag.Int("prometheus-port", 0, "Prometheus port (default: 12130 + equipmentID - 1)")
	flag.Parse()

	// Validate required flags
	if *equipmentID == 0 {
		fmt.Println("Error: -id (equipment ID) is required")
		flag.Usage()
		os.Exit(1)
	}

	// Validate mode
	if *mode != "replay" && *mode != "generate" {
		fmt.Printf("Error: -mode must be 'replay' or 'generate', got '%s'\n", *mode)
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logger
	deviceName := fmt.Sprintf("simulator-%d", *equipmentID)
	logger = duck.NewDuckLogger(deviceName, nil, slog.LevelInfo)

	// Load configuration from file if provided
	if *configFilename != "" {
		configFile, err := duck.ReadConfigurationFile(*configFilename)
		if err != nil {
			log.Fatalf("Error reading configuration: %v", err)
		}

		// Setup centrifuge subscription
		sub, err := duck.CreateNewSubscription(configFile.Centrifugal, deviceName)
		if err != nil {
			log.Printf("Warning: Could not create centrifuge subscription: %v", err)
		} else {
			logger = duck.NewDuckLogger(deviceName, sub, configFile.LogLevel)
		}
	}

	state = duck.INITIALIZED
	logger.State(state)

	// Start Prometheus
	if *prometheusPort == 0 {
		*prometheusPort = 12130 + *equipmentID - 1
	}
	startPrometheus(*prometheusPort)
	logger.Slog.Info("Prometheus started", "port", *prometheusPort)

	// If not in server mode, run CLI mode
	if !*serverMode {
		runCLI(*mode, *equipmentID, *filePath, *rate, *loop, *maxEvents, *fragmentSize, *packetsPerEvent, *errorInjection)
		return
	}

	// Server mode - start ConnectRPC server
	if *grpcPort == 0 {
		*grpcPort = 50070 + *equipmentID - 1
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		logger.Fatalf("Error listening on ConnectRPC port: %v", err)
	}

	logger.Slog.Info("Starting device simulator",
		"mode", *mode,
		"equipment_id", *equipmentID,
		"grpc_port", *grpcPort,
		"file", *filePath,
		"rate", *rate,
		"loop", *loop,
	)

	// Create ConnectRPC handler
	mux := http.NewServeMux()
	path, handler := pbconnect.NewRunControlHandler(&server{
		mode:               *mode,
		equipmentID:        *equipmentID,
		filePath:           *filePath,
		rate:               *rate,
		loop:               *loop,
		maxEvents:          *maxEvents,
		fragmentSize:       *fragmentSize,
		packetsPerEvent:    *packetsPerEvent,
		errorInjectionRate: *errorInjection,
	})
	mux.Handle(path, handler)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *grpcPort),
		Handler: mux,
	}

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Slog.Info("Shutting down...")
		if dataSource != nil {
			dataSource.Stop()
		}
		httpServer.Close()
		os.Exit(0)
	}()

	// Start server
	err = httpServer.Serve(lis)
	if err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Failed to serve ConnectRPC server: %v", err)
	}
}

// runCLI runs in CLI-only mode without ConnectRPC server
func runCLI(mode string, equipmentID int, filePath string, rate float64, loop bool, maxEvents int, fragmentSize int, packetsPerEvent int, errorInjection float64) {
	// Get database connection
	db, err := GetDatabaseConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	var source DataSource

	switch mode {
	case "generate":
		logger.Slog.Info("Running in CLI mode (generate)",
			"equipment_id", equipmentID,
			"rate", rate,
			"packets_per_event", packetsPerEvent,
			"error_injection", errorInjection,
			"max_events", maxEvents,
		)

		// Build generator configuration
		config, err := BuildGeneratorConfig(db, equipmentID, rate, packetsPerEvent, errorInjection, maxEvents, fragmentSize)
		if err != nil {
			log.Fatalf("Failed to build generator config: %v", err)
		}

		// Create and start generator
		generator, err := NewGenerator(config)
		if err != nil {
			log.Fatalf("Failed to create generator: %v", err)
		}

		if err := generator.Start(); err != nil {
			log.Fatalf("Failed to start generator: %v", err)
		}

		source = generator

	default: // "replay"
		if filePath == "" {
			fmt.Println("Error: -file is required in replay mode")
			os.Exit(1)
		}

		logger.Slog.Info("Running in CLI mode (replay)",
			"equipment_id", equipmentID,
			"file", filePath,
			"rate", rate,
			"loop", loop,
			"max_events", maxEvents,
		)

		// Build replay configuration
		config, err := BuildReplayConfig(db, equipmentID, filePath, rate, loop, maxEvents, fragmentSize)
		if err != nil {
			log.Fatalf("Failed to build replay config: %v", err)
		}

		// Create and start replayer
		replayer, err := NewReplayer(config)
		if err != nil {
			log.Fatalf("Failed to create replayer: %v", err)
		}

		if err := replayer.Start(); err != nil {
			log.Fatalf("Failed to start replay: %v", err)
		}

		source = replayer
	}

	// Print status periodically
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			fmt.Println(source.GetStatus())
		case <-sigCh:
			fmt.Printf("\nStopping %s...\n", mode)
			source.Stop()
			return
		}

		// Check if source is done
		if source.GetState() == SourceStateStopped {
			fmt.Printf("%s completed\n", mode)
			return
		}
	}
}
