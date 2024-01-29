package testhelpers

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"connectrpc.com/connect"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/control"
	pbconnect "github.com/jmbenlloch/next_duck/rpc/control/controlconnect"
	"github.com/stretchr/testify/require"
)

func CreateDummyGDCListeners(t *testing.T) {
	dbConfig := GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	rows, err := db.Query("SELECT DISTINCT port, enabled FROM gdcs")
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var port int
		var enabled bool
		require.NoError(t, rows.Scan(&port, &enabled))
		if enabled {
			ln, lerr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if lerr != nil {
				t.Logf("Port %d already in use, skipping dummy listener creation", port)
				continue
			}
			t.Cleanup(func() { ln.Close() })
			go func(l net.Listener) {
				for {
					c, aerr := l.Accept()
					if aerr != nil {
						return
					}
					go func(conn net.Conn) {
						io.Copy(io.Discard, conn)
						conn.Close()
					}(c)
				}
			}(ln)
		}
	}
}

// TestLDC wraps an LDC process for testing
type TestLDC struct {
	cmd            *exec.Cmd
	config         *duck.LDCConfiguration
	configFilename string
	cancel         context.CancelFunc
	rpcClient      *pbconnect.RunControlClient
	t              *testing.T
}

// EquipmentConfig specifies which equipments to enable for an LDC
type EquipmentConfig struct {
	LDCID        int
	EquipmentIDs []int // Empty means enable all equipments
}

// BuildLDCBinary builds the LDC binary
func BuildLDCBinary(t *testing.T) {
	projectRoot := GetProjectRoot()
	ldcBinary := filepath.Join(projectRoot, "bin", "ldcRPC")

	// Create bin directory if it doesn't exist
	binDir := filepath.Join(projectRoot, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Always build LDC to ensure latest code
	t.Logf("Building LDC binary...")
	cmd := exec.Command("go", "build", "-o", ldcBinary, "./ldcRPC")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build LDC: %v\nOutput: %s", err, output)
	}
	t.Logf("LDC binary built successfully")
}

// isConnectionRefused checks if the error indicates connection refused
func isConnectionRefused(err error) bool {
	return err != nil && (err.Error() == "connection refused" ||
		strings.Contains(err.Error(), "connect: connection refused"))
}

// StartTestLDC starts an LDC process for testing
// Uses the shared MySQL container for faster test execution
// Must be called after SetupSharedMySQL() in TestMain
func StartTestLDC(t *testing.T, configFilename string) *TestLDC {
	projectRoot := GetProjectRoot()
	ldcBinary := filepath.Join(projectRoot, "bin", "ldcRPC")

	// Ensure binary exists
	BuildLDCBinary(t)

	// Make config path absolute
	if !filepath.IsAbs(configFilename) {
		configFilename = filepath.Join(projectRoot, configFilename)
	}

	// Copy config to temp location to avoid modifying git-controlled files
	configFilename = prepareTestConfig(t, configFilename)

	// Setup shared MySQL - patch config file and update database
	ResetDatabase(t)

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

	host, _ := os.Hostname()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("UPDATE ldcs SET hostname=?", host)
	require.NoError(t, err)

	_, err = db.Exec("UPDATE equipments SET enabled=false WHERE ldcID=? AND id<>?", 1, 1)
	require.NoError(t, err)

	// Create context for process management
	ctx, cancel := context.WithCancel(context.Background())

	// Build command to start LDC
	cmd := exec.CommandContext(ctx, ldcBinary, "-config", configFilename)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start process
	startErr := cmd.Start()
	require.NoError(t, startErr, "Failed to start LDC process")

	t.Logf("Started LDC process with PID %d", cmd.Process.Pid)

	// Read configuration to get ConnectRPC port
	configuration, err := duck.ReadConfigurationWithoutRetry(configFilename)
	require.NoError(t, err, "Failed to read configuration")

	ldcConfig, err := duck.GetLDCConfiguration(configuration.LDCs, "")
	require.NoError(t, err, "Failed to get LDC configuration")

	testLDC := &TestLDC{
		cmd:            cmd,
		config:         ldcConfig,
		configFilename: configFilename,
		cancel:         cancel,
		t:              t,
	}

	// Wait for LDC to be ready
	if err := testLDC.waitForServerReady(10 * time.Second); err != nil {
		t.Fatalf("LDC server not ready: %v", err)
	}

	// Connect to ConnectRPC server
	testLDC.ConnectRPC()

	return testLDC
}

// StartTestLDCWithConfig starts an LDC process with custom equipment configuration
// Parameters:
//   - ldcName: Name of the LDC to start (empty string uses first LDC in config)
//   - skipDBReset: If true, skips database reset (useful for starting multiple LDCs sharing same DB)
func StartTestLDCWithConfig(t *testing.T, configFilename string, equipConfigs []EquipmentConfig, ldcName string, skipDBReset bool) *TestLDC {
	projectRoot := GetProjectRoot()
	ldcBinary := filepath.Join(projectRoot, "bin", "ldcRPC")

	// Ensure binary exists
	BuildLDCBinary(t)

	// Make config path absolute
	if !filepath.IsAbs(configFilename) {
		configFilename = filepath.Join(projectRoot, configFilename)
	}

	// Copy config to temp location to avoid modifying git-controlled files
	configFilename = prepareTestConfig(t, configFilename)

	// Reset database and setup with custom equipment config (unless skipped)
	if !skipDBReset {
		ResetDatabase(t)
	}

	// Read YAML config
	b, err := os.ReadFile(configFilename)
	require.NoError(t, err)
	var cfg duck.ConfigurationFile
	require.NoError(t, yaml.Unmarshal(b, &cfg))

	// Patch DB port to use shared MySQL port
	dbConfig := GetSharedDBConfig()
	cfg.Database.Port = dbConfig.Port
	cfg.Database.Hostname = dbConfig.Hostname

	// Write back YAML
	out, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configFilename, out, 0644))

	// Update LDC hostname in database
	host, _ := os.Hostname()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("UPDATE ldcs SET hostname=?", host)
	require.NoError(t, err)

	// Configure equipments based on EquipmentConfig
	for _, equipConfig := range equipConfigs {
		// Ensure all specified equipments exist for this LDC in the database
		for _, eqID := range equipConfig.EquipmentIDs {
			// Check if equipment exists for this LDC
			var exists int
			err = db.QueryRow("SELECT COUNT(*) FROM equipments WHERE ldcID=? AND id=?", equipConfig.LDCID, eqID).Scan(&exists)
			require.NoError(t, err)

			if exists == 0 {
				// Check if equipment ID exists globally (for a different LDC)
				var globalExists int
				err = db.QueryRow("SELECT COUNT(*) FROM equipments WHERE id=?", eqID).Scan(&globalExists)
				require.NoError(t, err)

				if globalExists == 0 {
					// Insert missing equipment (using default values from seed data)
					_, err = db.Exec("INSERT INTO equipments (id, type, device_ip, host_ip, host_port, ldcID, enabled) VALUES (?, 22, '127.0.0.1', '127.0.0.1', ?, ?, true)",
						eqID, 16006+eqID, equipConfig.LDCID)
					require.NoError(t, err, "Failed to insert equipment %d for LDC %d", eqID, equipConfig.LDCID)
					t.Logf("Created equipment %d for LDC %d with port %d", eqID, equipConfig.LDCID, 16006+eqID)
				} else {
					// Equipment exists but for different LDC - this is an error in test setup
					t.Fatalf("Equipment ID %d already exists in database for a different LDC. Test cannot proceed. Please use a different equipment ID.", eqID)
				}
			}
		}

		if len(equipConfig.EquipmentIDs) == 0 {
			// Empty list means enable all - do nothing
			continue
		}

		// Disable all equipments for this LDC first
		_, err = db.Exec("UPDATE equipments SET enabled=false WHERE ldcID=?", equipConfig.LDCID)
		require.NoError(t, err)

		// Enable only the specified equipments
		for _, eqID := range equipConfig.EquipmentIDs {
			_, err = db.Exec("UPDATE equipments SET enabled=true WHERE ldcID=? AND id=?", equipConfig.LDCID, eqID)
			require.NoError(t, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Build command to start LDC
	args := []string{"-config", configFilename}
	if ldcName != "" {
		args = append(args, "-name", ldcName)
	}
	cmd := exec.CommandContext(ctx, ldcBinary, args...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start process
	startErr := cmd.Start()
	require.NoError(t, startErr, "Failed to start LDC process")

	if ldcName != "" {
		t.Logf("Started LDC process with name '%s' and PID %d", ldcName, cmd.Process.Pid)
	} else {
		t.Logf("Started LDC process with PID %d", cmd.Process.Pid)
	}

	// Read configuration to get ConnectRPC port
	configuration, err := duck.ReadConfigurationWithoutRetry(configFilename)
	require.NoError(t, err, "Failed to read configuration")

	ldcConfig, err := duck.GetLDCConfiguration(configuration.LDCs, ldcName)
	require.NoError(t, err, "Failed to get LDC configuration")

	testLDC := &TestLDC{
		cmd:            cmd,
		config:         ldcConfig,
		configFilename: configFilename,
		cancel:         cancel,
		t:              t,
	}

	// Wait for LDC to be ready
	if err := testLDC.waitForServerReady(10 * time.Second); err != nil {
		t.Fatalf("LDC server not ready: %v", err)
	}

	// Connect to ConnectRPC server
	testLDC.ConnectRPC()

	return testLDC
}

// waitForServerReady waits for the ConnectRPC server to be ready
func (l *TestLDC) waitForServerReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	target := fmt.Sprintf("http://127.0.0.1:%d", l.config.GRPCPort)

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

// ConnectRPC establishes ConnectRPC connection to the LDC
func (l *TestLDC) ConnectRPC() {
	if l.rpcClient != nil {
		return // Already connected
	}

	target := fmt.Sprintf("http://127.0.0.1:%d", l.config.GRPCPort)
	client := pbconnect.NewRunControlClient(
		http.DefaultClient,
		target,
	)

	l.rpcClient = &client
	l.t.Logf("Connected to LDC ConnectRPC server at %s", target)
}

// WaitForState waits for the LDC to reach a specific state
func (l *TestLDC) WaitForState(expectedState string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := l.GetState()
		if err == nil && state == expectedState {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("LDC did not reach state %s within timeout", expectedState)
}

// StartRun starts a data acquisition run via ConnectRPC
func (l *TestLDC) StartRun() error {
	if l.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*l.rpcClient).StartRun(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return fmt.Errorf("StartRun failed: %w", err)
	}

	l.t.Logf("StartRun response: %s", resp.Msg.Message)

	// Wait until LDC reports RUNNING to avoid race in tests
	return l.WaitForState("RUNNING", 10*time.Second)
}

// StopRun stops a data acquisition run via ConnectRPC
func (l *TestLDC) StopRun() error {
	if l.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*l.rpcClient).StopRun(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return fmt.Errorf("StopRun failed: %w", err)
	}

	l.t.Logf("StopRun response: %s", resp.Msg.Message)
	return nil
}

// GetState retrieves the current state via ConnectRPC
func (l *TestLDC) GetState() (string, error) {
	if l.rpcClient == nil {
		return "", fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*l.rpcClient).GetState(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return "", fmt.Errorf("GetState failed: %w", err)
	}

	return resp.Msg.Message, nil
}

// GetStatistics retrieves run statistics via ConnectRPC
func (l *TestLDC) GetStatistics() (*pb.RunStatisticsReply, error) {
	if l.rpcClient == nil {
		return nil, fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := (*l.rpcClient).GetRunStatistics(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return nil, fmt.Errorf("GetRunStatistics failed: %w", err)
	}

	return stats.Msg, nil
}

// PingDevices sends a ping request to test device connectivity via ConnectRPC
func (l *TestLDC) PingDevices() (bool, error) {
	if l.rpcClient == nil {
		return false, fmt.Errorf("ConnectRPC client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := (*l.rpcClient).PingDevices(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		return false, fmt.Errorf("PingDevices failed: %w", err)
	}

	return resp.Msg.Success, nil
}

// WaitForProcessedEvents waits for the LDC to process at least the specified number of events
func (l *TestLDC) WaitForProcessedEvents(minEvents int64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		stats, err := l.GetStatistics()
		if err == nil && stats.Events >= minEvents {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("LDC did not process %d events within timeout", minEvents)
}

// IsRunning checks if the LDC process is still running
func (l *TestLDC) IsRunning() bool {
	if l.cmd == nil || l.cmd.Process == nil {
		return false
	}

	// Check if process is still alive by sending signal 0
	err := l.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPrometheusMetric scrapes a specific metric from the Prometheus endpoint
func (l *TestLDC) GetPrometheusMetric(metricName string) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/metrics", l.config.PrometheusPort)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to scrape Prometheus: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Find the metric line
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, metricName+" ") {
			// Return just the value (everything after the metric name and space)
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("metric %s not found", metricName)
}

// Stop stops the LDC process
func (l *TestLDC) Stop() {
	l.t.Logf("Stopping LDC process (PID %d)", l.cmd.Process.Pid)

	l.cancel()

	_ = l.cmd.Wait()
}

func StartTestLDCWithCentrifuge(t *testing.T, configFilename string, equipConfigs []EquipmentConfig, ldcName string, skipDBReset bool, centrifugeCfg duck.CentrifugalConfiguration) *TestLDC {
	projectRoot := GetProjectRoot()
	ldcBinary := filepath.Join(projectRoot, "bin", "ldcRPC")

	BuildLDCBinary(t)

	if !filepath.IsAbs(configFilename) {
		configFilename = filepath.Join(projectRoot, configFilename)
	}

	configFilename = prepareTestConfig(t, configFilename)

	if !skipDBReset {
		ResetDatabase(t)
	}

	b, err := os.ReadFile(configFilename)
	require.NoError(t, err)
	var cfg duck.ConfigurationFile
	require.NoError(t, yaml.Unmarshal(b, &cfg))

	dbConfig := GetSharedDBConfig()
	cfg.Database.Port = dbConfig.Port
	cfg.Database.Hostname = dbConfig.Hostname

	cfg.Centrifugal.Host = centrifugeCfg.Host
	cfg.Centrifugal.Port = centrifugeCfg.Port
	cfg.Centrifugal.Token = centrifugeCfg.Token

	out, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configFilename, out, 0644))

	t.Logf("LDC config: Centrifuge=%s:%d", cfg.Centrifugal.Host, cfg.Centrifugal.Port)

	host, _ := os.Hostname()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("UPDATE ldcs SET hostname=?", host)
	require.NoError(t, err)

	for _, equipConfig := range equipConfigs {
		for _, eqID := range equipConfig.EquipmentIDs {
			var exists int
			err = db.QueryRow("SELECT COUNT(*) FROM equipments WHERE ldcID=? AND id=?", equipConfig.LDCID, eqID).Scan(&exists)
			require.NoError(t, err)

			if exists == 0 {
				var globalExists int
				err = db.QueryRow("SELECT COUNT(*) FROM equipments WHERE id=?", eqID).Scan(&globalExists)
				require.NoError(t, err)

				if globalExists == 0 {
					_, err = db.Exec("INSERT INTO equipments (id, type, device_ip, host_ip, host_port, ldcID, enabled) VALUES (?, 22, '127.0.0.1', '127.0.0.1', ?, ?, true)",
						eqID, 16006+eqID, equipConfig.LDCID)
					require.NoError(t, err, "Failed to insert equipment %d for LDC %d", eqID, equipConfig.LDCID)
					t.Logf("Created equipment %d for LDC %d with port %d", eqID, equipConfig.LDCID, 16006+eqID)
				} else {
					t.Fatalf("Equipment ID %d already exists in database for a different LDC.", eqID)
				}
			}
		}

		if len(equipConfig.EquipmentIDs) == 0 {
			continue
		}

		_, err = db.Exec("UPDATE equipments SET enabled=false WHERE ldcID=?", equipConfig.LDCID)
		require.NoError(t, err)

		for _, eqID := range equipConfig.EquipmentIDs {
			_, err = db.Exec("UPDATE equipments SET enabled=true WHERE ldcID=? AND id=?", equipConfig.LDCID, eqID)
			require.NoError(t, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	args := []string{"-config", configFilename}
	if ldcName != "" {
		args = append(args, "-name", ldcName)
	}
	cmd := exec.CommandContext(ctx, ldcBinary, args...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startErr := cmd.Start()
	require.NoError(t, startErr, "Failed to start LDC process")

	if ldcName != "" {
		t.Logf("Started LDC process with name '%s' and PID %d (with Centrifuge)", ldcName, cmd.Process.Pid)
	} else {
		t.Logf("Started LDC process with PID %d (with Centrifuge)", cmd.Process.Pid)
	}

	configuration, err := duck.ReadConfigurationWithoutRetry(configFilename)
	require.NoError(t, err, "Failed to read configuration")

	ldcConfig, err := duck.GetLDCConfiguration(configuration.LDCs, ldcName)
	require.NoError(t, err, "Failed to get LDC configuration")

	testLDC := &TestLDC{
		cmd:            cmd,
		config:         ldcConfig,
		configFilename: configFilename,
		cancel:         cancel,
		t:              t,
	}

	if err := testLDC.waitForServerReady(10 * time.Second); err != nil {
		t.Fatalf("LDC server not ready: %v", err)
	}

	testLDC.ConnectRPC()

	return testLDC
}
