package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

func countEnabledGDCs(gdcs []duck.GDCConfiguration) int {
	nGDCs := 0
	for _, gdc := range gdcs {
		if gdc.Enabled {
			nGDCs++
		}
	}
	return nGDCs
}

func gdcServer(s *server, configFilename string, serverName string) {
	for {
		// Wait for run to start
		s.runActiveCond.L.Lock()
		for !s.runActive {
			s.runActiveCond.Wait()
		}
		s.runActiveCond.L.Unlock()

		// Re-read configuration for each run (in case it changed)
		configuration, err := duck.ReadConfiguration(configFilename, nil, nil)
		if err != nil {
			message := fmt.Sprintf("Error reading configuration %s", err.Error())
			s.logger.Slog.Error(message)
			s.runActiveCond.L.Lock()
			s.runActive = false
			s.runActiveCond.L.Unlock()
			s.setState(duck.INITIALIZED)
			continue
		}
		gdcConfiguration, err := duck.GetGDCConfiguration(configuration.GDCs, serverName)
		if err != nil {
			message := fmt.Sprintf("Error reading GDC configuration %s", err.Error())
			s.logger.Slog.Error(message)
			s.runActiveCond.L.Lock()
			s.runActive = false
			s.runActiveCond.L.Unlock()
			s.setState(duck.INITIALIZED)
			continue
		}

		// Add configuration values to context
		s.ctxMutex.Lock()
		s.ctx = context.WithValue(s.ctx, "gdcConfiguration", gdcConfiguration)
		s.ctx = context.WithValue(s.ctx, "nGDCs", countEnabledGDCs(configuration.GDCs))
		s.ctx = context.WithValue(s.ctx, "runNumber", configuration.RunNumber)
		s.ctx = context.WithValue(s.ctx, "maxFilesize", configuration.Duck.Filesize)
		s.ctx = context.WithValue(s.ctx, "experiment", configuration.Duck.Experiment)
		s.ctx = context.WithValue(s.ctx, "metricsChBufferSize", configuration.Duck.MetricsCh)
		s.ctx = context.WithValue(s.ctx, "tcpConnChBufferSize", configuration.Duck.TCPConnectionsCh)
		s.ctx = context.WithValue(s.ctx, "writerChBufferSize", configuration.Duck.WriterCh)
		s.ctx = context.WithValue(s.ctx, "decoderChBufferSize", configuration.Duck.DecoderCh)
		s.ctx = context.WithValue(s.ctx, "decoderWorkers", configuration.Duck.DecoderWorkers)
		s.ctx = context.WithValue(s.ctx, "writeOutput", gdcConfiguration.WriteOutput)
		s.ctx = context.WithValue(s.ctx, "decode", gdcConfiguration.Decode)
		s.ctx = context.WithValue(s.ctx, "decoderConfig", configuration.Decoder)
		s.ctxMutex.Unlock()

		duck.AddRunNumberToLogger(&s.logger, configuration.RunNumber)

		if gdcConfiguration.Decode {
			setupDecoder(s, configuration.RunNumber)
		}

		// Resolve the string address to a TCP address (on each run to allow port changes)
		listenAddr := fmt.Sprintf("0.0.0.0:%d", gdcConfiguration.Port)
		message := fmt.Sprintf("Listening on %s", listenAddr)
		s.logger.Slog.Info(message)

		// Start listening for TCP connections on the given address
		// Use SO_REUSEADDR to allow quick rebinding after StopRun -> StartRun
		lc := net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				return c.Control(func(fd uintptr) {
					_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				})
			},
		}

		// Retry bind multiple times to handle "address already in use" from previous process
		var listener net.Listener
		var listenErr error
		for retry := 0; retry < 10; retry++ {
			listener, listenErr = lc.Listen(context.Background(), "tcp4", listenAddr)
			if listenErr == nil {
				break
			}
			message := fmt.Errorf("error starting TCP listener (retry %d/10): %w", retry+1, listenErr)
			s.logger.Slog.Warn(message.Error())
			time.Sleep(500 * time.Millisecond)
		}

		if listenErr != nil {
			message := fmt.Errorf("error starting TCP listener after retries: %w", listenErr)
			s.logger.Slog.Error(message.Error())
			s.runActiveCond.L.Lock()
			s.runActive = false
			s.runActiveCond.L.Unlock()
			s.setState(duck.INITIALIZED)
			continue
		}
		tcpListener, ok := listener.(*net.TCPListener)
		if !ok {
			listener.Close()
			message := fmt.Errorf("expected TCPListener")
			s.logger.Slog.Error(message.Error())
			s.runActiveCond.L.Lock()
			s.runActive = false
			s.runActiveCond.L.Unlock()
			s.setState(duck.INITIALIZED)
			continue
		}

		dataChBufferSize := configuration.Duck.DataCh
		dataChannel := make(chan LDCData, dataChBufferSize)

		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			updateDataChannelGauge(s, dataChannel)
		}()
		go func() {
			defer wg.Done()
			readData(s, dataChannel, gdcConfiguration.ID, configuration.LDCs, gdcConfiguration.Path)
		}()
		go func() {
			defer wg.Done()
			acceptTCPConnections(s, tcpListener, dataChannel)
		}()

		// Transition to RUNNING state once TCP listener is ready
		s.setState(duck.RUNNING)

		// Wait for stop signal
		s.runActiveCond.L.Lock()
		for s.runActive {
			s.runActiveCond.Wait()
		}
		s.runActiveCond.L.Unlock()

		// Close listener first to unblock Accept() calls
		// Then cancel context to signal goroutines to stop
		s.logger.Slog.Info("Stopping GDC")
		err = tcpListener.Close()
		if err != nil {
			message := fmt.Errorf("error closing connection: %w", err)
			s.logger.Slog.Error(message.Error())
		}

		s.ctxMutex.Lock()
		s.cancelCtx()
		s.ctxMutex.Unlock()

		// Wait for goroutines to finish
		wg.Wait()

		// Reset state to INITIALIZED for next run
		s.setState(duck.INITIALIZED)
	}
}

func acceptTCPConnections(s *server, listener *net.TCPListener, dataChannel chan LDCData) {
	tcpConnChBufferSize, ok := s.getContext().Value("tcpConnChBufferSize").(int)
	if !ok {
		message := "error getting TCP connection channel buffer size"
		s.logger.Slog.Error(message)
		return
	}
	ldcConnectionOpenChan := make(chan net.Addr, tcpConnChBufferSize)
	ldcConnectionCloseChan := make(chan net.Addr, tcpConnChBufferSize)
	go ldcConnectionMonitor(s, ldcConnectionOpenChan, ldcConnectionCloseChan)

	for {
		select {
		case <-s.getContext().Done():
			return
		default:
			// Accept new connections
			// This is a blocking method. The listener has to be closed from another Goroutine
			s.logger.Slog.Debug("Waiting for connection")
			conn, err := listener.Accept()
			if err != nil {
				// Accept will block until a new connection is received or the listener is closed
				// If the listener is closed, the error will be "use of closed network connection"
				// When the server is stopped, the listener is closed and the error is expected
				// Therefore, show error only if the state is not RUNNING
				if s.getState() == duck.RUNNING {
					message := fmt.Errorf("error accepting TCP connection: %w", err)
					s.logger.Slog.Error(message.Error())
				}
				return
			}
			// Handle new connections in a Goroutine for concurrency
			ldcConnectionOpenChan <- conn.RemoteAddr()
			go handleConnection(s, conn, dataChannel, ldcConnectionCloseChan)
		}
	}
}

func ldcConnectionMonitor(s *server, ldcConnectionOpenChan chan net.Addr,
	ldcConnectionCloseChan chan net.Addr) {
	ldcConnections := make(map[net.Addr]bool)
	for {
		select {
		case <-s.getContext().Done():
			return
		case address := <-ldcConnectionOpenChan:
			ldcConnections[address] = true
		case address := <-ldcConnectionCloseChan:
			delete(ldcConnections, address)
			// Once connections start to be closed, state is STOPPING
			if s.getState() == duck.RUNNING {
				s.setState(duck.STOPPING)
			}
			if len(ldcConnections) == 0 {
				s.stopOnLocalError("No LDC connections, stopping GDC")
			}
		}
	}
}

func updateDataChannelGauge(s *server, dataChannel chan LDCData) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			s.metrics.dataChannelCounter.Set(float64(len(dataChannel)))
		case <-s.getContext().Done():
			return
		}
	}
}
