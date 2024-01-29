package testhelpers

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"connectrpc.com/connect"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/control"
	pbconnect "github.com/jmbenlloch/next_duck/rpc/control/controlconnect"
	"github.com/stretchr/testify/require"
)

// TestGDC wraps a GDC process for testing
type TestGDC struct {
	cmd            *exec.Cmd
	config         *duck.GDCConfiguration
	configFilename string
	serverName     string
	cancel         context.CancelFunc
	rpcClient      *pbconnect.RunControlClient
	t              *testing.T
}

// TrackingGDCListener tracks which events are received by a dummy GDC
type TrackingGDCListener struct {
	listener    net.Listener
	eventIDs    []int
	mu          sync.Mutex
	connections []net.Conn
	closed      bool
}

// StartTrackingGDCListener creates a TCP listener that tracks received event IDs
// Note: This must be called BEFORE StartTestLDC to prevent the test helpers
// from starting non-tracking dummy listeners on the same ports
func StartTrackingGDCListener(t *testing.T, port int) *TrackingGDCListener {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err, "Failed to start tracking GDC listener on port %d", port)

	tracker := &TrackingGDCListener{
		listener:    ln,
		eventIDs:    make([]int, 0, 100),
		connections: make([]net.Conn, 0),
		closed:      false,
	}

	go func() {
		buf := make([]byte, 65536)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // Listener closed
			}

			// Track this connection
			tracker.mu.Lock()
			if !tracker.closed {
				tracker.connections = append(tracker.connections, conn)
			}
			tracker.mu.Unlock()

			go func(c net.Conn) {
				defer func() {
					c.Close()
					// Remove from connections list when done
					tracker.mu.Lock()
					for i, conn := range tracker.connections {
						if conn == c {
							tracker.connections = append(tracker.connections[:i], tracker.connections[i+1:]...)
							break
						}
					}
					tracker.mu.Unlock()
				}()
				for {
					n, err := c.Read(buf)
					if err != nil || n == 0 {
						return
					}
					// Extract event ID from LDC header (bytes 24-28)
					if n >= 28 {
						eventID := int(binary.LittleEndian.Uint32(buf[24:28]))
						tracker.mu.Lock()
						tracker.eventIDs = append(tracker.eventIDs, eventID)
						tracker.mu.Unlock()
						t.Logf("GDC on port %d received event %d (bytes: %d)", port, eventID, n)
					}
				}
			}(conn)
		}
	}()

	return tracker
}

// GetReceivedEvents returns the list of event IDs received by this GDC
func (t *TrackingGDCListener) GetReceivedEvents() []int {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]int, len(t.eventIDs))
	copy(result, t.eventIDs)
	return result
}

// Close closes the listener and all active connections
func (t *TrackingGDCListener) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Mark as closed to prevent new connections from being added
	t.closed = true

	// Close all active connections to simulate GDC failure
	// This will cause subsequent LDC writes to fail with "broken pipe"
	for _, conn := range t.connections {
		// Set read deadline to past to force immediate read failure
		conn.SetReadDeadline(time.Time{})
		// Close the connection
		conn.Close()
	}
	t.connections = nil

	// Close the listener to prevent new connections
	return t.listener.Close()
}

// StartTrackingGDCListenersForAllEnabled starts tracking GDC listeners for all enabled GDCs in the database
// Returns a cleanup function that should be called with defer to close all listeners
func StartTrackingGDCListenersForAllEnabled(t *testing.T) (listeners []*TrackingGDCListener, cleanup func()) {
	dbConfig := GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err, "Failed to connect to database")
	defer db.Close()

	rows, err := db.Query("SELECT port FROM gdcs WHERE enabled=true")
	require.NoError(t, err, "Failed to query enabled GDCs")
	defer rows.Close()

	listeners = make([]*TrackingGDCListener, 0)
	for rows.Next() {
		var port int
		require.NoError(t, rows.Scan(&port), "Failed to scan GDC port")
		listener := StartTrackingGDCListener(t, port)
		listeners = append(listeners, listener)
	}

	cleanup = func() {
		for _, l := range listeners {
			l.Close()
		}
	}

	return listeners, cleanup
}

// BuildGDCBinary builds the GDC binary
// Note: GDC requires HDF5 and Blosc libraries to be installed
func BuildGDCBinary(t *testing.T) {
	projectRoot := GetProjectRoot()
	gdcBinary := filepath.Join(projectRoot, "bin", "gdcRPC")

	// Create bin directory if it doesn't exist
	binDir := filepath.Join(projectRoot, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Always build GDC to ensure latest code
	t.Logf("Building GDC binary...")
	cmd := exec.Command("go", "build", "-o", gdcBinary, "./gdcRPC")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build GDC: %v\nOutput: %s", err, output)
	}
	t.Logf("GDC binary built successfully")
}

// StartTestGDC starts a GDC process for testing
func StartTestGDC(t *testing.T, configFilename string, serverName string) *TestGDC {
	projectRoot := GetProjectRoot()
	gdcBinary := filepath.Join(projectRoot, "bin", "gdcRPC")

	// Ensure binary exists
	BuildGDCBinary(t)

	// Create context for process management
	ctx, cancel := context.WithCancel(context.Background())

	// Make config path absolute
	if !filepath.IsAbs(configFilename) {
		configFilename = filepath.Join(projectRoot, configFilename)
	}

	// Copy config to temp location to avoid modifying git-controlled files
	configFilename = prepareTestConfig(t, configFilename)

	// Patch config file with database credentials if using shared MySQL
	if sharedMySQLResource != nil {
		t.Logf("Patching config file with shared MySQL credentials")
		b, err := os.ReadFile(configFilename)
		require.NoError(t, err)
		var cfg duck.ConfigurationFile
		require.NoError(t, yaml.Unmarshal(b, &cfg))

		dbConfig := GetSharedDBConfig()
		cfg.Database.Port = dbConfig.Port
		cfg.Database.Hostname = dbConfig.Hostname

		out, err := yaml.Marshal(&cfg)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configFilename, out, 0644))

		// Update GDC hostname in database to match current hostname
		// DO NOT update the name - each test should use the existing GDC name from database
		host, _ := os.Hostname()
		db, err := duck.ParseDatabaseConfiguration(dbConfig)
		require.NoError(t, err)
		defer db.Close()

		// Update all GDCs to use current hostname (container hostname)
		_, err = db.Exec("UPDATE gdcs SET hostname=?", host)
		require.NoError(t, err)
	}

	// Build command to start GDC
	cmd := exec.CommandContext(ctx, gdcBinary, "-config", configFilename, "-name", serverName)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start process
	err := cmd.Start()
	require.NoError(t, err, "Failed to start GDC process")

	t.Logf("Started GDC process with PID %d", cmd.Process.Pid)

	// Read configuration to get ConnectRPC port (stored in GRPCPort field for historical reasons)
	configuration, err := duck.ReadConfigurationWithoutRetry(configFilename)
	require.NoError(t, err, "Failed to read configuration")

	gdcConfig, err := duck.GetGDCConfiguration(configuration.GDCs, serverName)
	require.NoError(t, err, "Failed to get GDC configuration")

	testGDC := &TestGDC{
		cmd:            cmd,
		config:         gdcConfig,
		configFilename: configFilename,
		serverName:     serverName,
		cancel:         cancel,
		t:              t,
	}

	// Wait for GDC to be ready (ConnectRPC server to start)
	// Use a proper readiness check instead of sleep
	if err := testGDC.waitForServerReady(10 * time.Second); err != nil {
		t.Fatalf("GDC server not ready: %v", err)
	}

	// Connect to ConnectRPC server
	testGDC.ConnectRPC()

	return testGDC
}

// waitForServerReady waits for the ConnectRPC server to be ready
func (g *TestGDC) waitForServerReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	target := fmt.Sprintf("http://127.0.0.1:%d", g.config.GRPCPort)

	client := pbconnect.NewRunControlClient(http.DefaultClient, target)

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_, err := client.GetState(ctx, connect.NewRequest(&pb.DuckRequest{}))
		if err == nil {
			return nil
		}

		// Check if error is connection refused (server not ready yet)
		if !isConnectionRefused(err) {
			// Some other error occurred, might be real
			return fmt.Errorf("unexpected error: %w", err)
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("server at %s not ready within timeout", target)
}

// ConnectRPC establishes ConnectRPC connection to the GDC
func (g *TestGDC) ConnectRPC() {
	if g.rpcClient != nil {
		return // Already connected
	}

	target := fmt.Sprintf("http://127.0.0.1:%d", g.config.GRPCPort)
	client := pbconnect.NewRunControlClient(
		http.DefaultClient,
		target,
	)

	g.rpcClient = &client
	g.t.Logf("Connected to GDC ConnectRPC server at %s", target)
}

// WaitForState waits for the GDC to reach a specific state
func (g *TestGDC) WaitForState(expectedState string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := g.GetState()
		if err == nil && state == expectedState {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("GDC did not reach state %s within timeout", expectedState)
}

// StartRun starts a data acquisition run via ConnectRPC
func (g *TestGDC) StartRun() error {
	if g.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*g.rpcClient).StartRun(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return fmt.Errorf("StartRun failed: %w", err)
	}

	g.t.Logf("StartRun response: %s", resp.Msg.Message)
	return nil
}

// StopRun stops a data acquisition run via ConnectRPC
func (g *TestGDC) StopRun() error {
	if g.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*g.rpcClient).StopRun(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return fmt.Errorf("StopRun failed: %w", err)
	}

	g.t.Logf("StopRun response: %s", resp.Msg.Message)
	return nil
}

// GetState retrieves the current state via ConnectRPC
func (g *TestGDC) GetState() (string, error) {
	if g.rpcClient == nil {
		return "", fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*g.rpcClient).GetState(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return "", fmt.Errorf("GetState failed: %w", err)
	}

	return resp.Msg.Message, nil
}

// GetStatistics retrieves run statistics via ConnectRPC
func (g *TestGDC) GetStatistics() (*pb.RunStatisticsReply, error) {
	if g.rpcClient == nil {
		return nil, fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := (*g.rpcClient).GetRunStatistics(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return nil, fmt.Errorf("GetRunStatistics failed: %w", err)
	}

	return stats.Msg, nil
}

// PingDevices sends a ping request to test device connectivity via ConnectRPC
// Note: GDC returns true for compatibility but doesn't implement actual ping functionality
func (g *TestGDC) PingDevices() (bool, error) {
	if g.rpcClient == nil {
		return false, fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*g.rpcClient).PingDevices(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return false, fmt.Errorf("PingDevices failed: %w", err)
	}

	return resp.Msg.Success, nil
}

// Stop stops the GDC process
func (g *TestGDC) Stop() {
	g.t.Logf("Stopping GDC process (PID %d)", g.cmd.Process.Pid)

	// ConnectRPC client doesn't need explicit connection closing

	// Cancel context (kills the process)
	g.cancel()

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- g.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited normally
	case <-time.After(5 * time.Second):
		// Timeout - force kill
		g.t.Logf("Timeout waiting for GDC process %d to exit, force killing", g.cmd.Process.Pid)
		_ = g.cmd.Process.Kill()
		_ = g.cmd.Wait()
	}

	// Give the OS time to clean up the TCP socket
	// This prevents "address already in use" errors in subsequent tests
	time.Sleep(500 * time.Millisecond)
}
