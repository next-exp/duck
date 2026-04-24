package testhelpers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"connectrpc.com/connect"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	pbconnect "github.com/jmbenlloch/next_duck/rpc/api/apiconnect"
	"github.com/stretchr/testify/require"
)

type TestAPI struct {
	cmd            *exec.Cmd
	configFilename string
	port           int
	cancel         context.CancelFunc
	rpcClient      pbconnect.DuckAPIClient
	t              *testing.T
}

func findAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "Failed to find available port")
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func BuildAPIBinary(t *testing.T) {
	projectRoot := GetProjectRoot()
	apiBinary := filepath.Join(projectRoot, "bin", "api")

	binDir := filepath.Join(projectRoot, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	t.Logf("Building API binary...")
	cmd := exec.Command("go", "build", "-o", apiBinary, "./api")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build API: %v\nOutput: %s", err, output)
	}
	t.Logf("API binary built successfully")
}

func StartTestAPI(t *testing.T, configFilename string) *TestAPI {
	projectRoot := GetProjectRoot()
	apiBinary := filepath.Join(projectRoot, "bin", "api")

	BuildAPIBinary(t)

	if !filepath.IsAbs(configFilename) {
		configFilename = filepath.Join(projectRoot, configFilename)
	}

	configFilename = prepareTestConfig(t, configFilename)

	b, err := os.ReadFile(configFilename)
	require.NoError(t, err)
	var cfg duck.ConfigurationFile
	require.NoError(t, yaml.Unmarshal(b, &cfg))

	dbConfig := GetSharedDBConfig()
	cfg.Database.Port = dbConfig.Port
	cfg.Database.Hostname = dbConfig.Hostname

	cfg.Centrifugal.Host = ""
	cfg.Centrifugal.Port = 0

	out, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configFilename, out, 0644))

	port := findAvailablePort(t)

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, apiBinary, "-config", configFilename, "-port", fmt.Sprintf("%d", port))
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startErr := cmd.Start()
	require.NoError(t, startErr, "Failed to start API process")

	t.Logf("Started API process with PID %d on port %d", cmd.Process.Pid, port)

	testAPI := &TestAPI{
		cmd:            cmd,
		configFilename: configFilename,
		port:           port,
		cancel:         cancel,
		t:              t,
	}

	if err := testAPI.waitForServerReady(10 * time.Second); err != nil {
		t.Fatalf("API server not ready: %v", err)
	}

	testAPI.ConnectRPC()

	return testAPI
}

func (a *TestAPI) waitForServerReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	target := fmt.Sprintf("http://127.0.0.1:%d/daq", a.port)

	client := pbconnect.NewDuckAPIClient(http.DefaultClient, target)

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err := client.GetRunNumber(ctx, connect.NewRequest(&pb.GetRunNumberRequest{}))
		cancel()

		if err == nil {
			return nil
		}

		if !isConnectionRefused(err) {
			return fmt.Errorf("unexpected error: %w", err)
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("API server at %s not ready within timeout", target)
}

func (a *TestAPI) ConnectRPC() {
	if a.rpcClient != nil {
		return
	}

	target := fmt.Sprintf("http://127.0.0.1:%d/daq", a.port)
	client := pbconnect.NewDuckAPIClient(
		http.DefaultClient,
		target,
	)

	a.rpcClient = client
	a.t.Logf("Connected to API ConnectRPC server at %s", target)
}

func (a *TestAPI) StartRun(ctx context.Context) error {
	if a.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	resp, err := a.rpcClient.StartRun(ctx, connect.NewRequest(&pb.StartRunRequest{}))
	if err != nil {
		return fmt.Errorf("StartRun failed: %w", err)
	}

	a.t.Logf("API StartRun response: success=%v, message=%s", resp.Msg.Success, resp.Msg.Message)
	return nil
}

func (a *TestAPI) StopRun(ctx context.Context) error {
	if a.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	resp, err := a.rpcClient.StopRun(ctx, connect.NewRequest(&pb.StopRunRequest{}))
	if err != nil {
		return fmt.Errorf("StopRun failed: %w", err)
	}

	a.t.Logf("API StopRun response: success=%v, message=%s", resp.Msg.Success, resp.Msg.Message)

	return a.waitForTransitionIdle(ctx, 30*time.Second)
}

func (a *TestAPI) waitForTransitionIdle(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pollCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		statusResp, err := a.rpcClient.GetRunTransitionStatus(pollCtx, connect.NewRequest(&pb.GetRunTransitionStatusRequest{}))
		cancel()
		if err != nil {
			return fmt.Errorf("GetRunTransitionStatus failed: %w", err)
		}
		state := statusResp.Msg.State
		if state == "idle" || state == "error" {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("transition did not complete within timeout")
}

func (a *TestAPI) ForceStopRun(ctx context.Context) error {
	if a.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	resp, err := a.rpcClient.ForceStopRun(ctx, connect.NewRequest(&pb.ForceStopRunRequest{}))
	if err != nil {
		return fmt.Errorf("ForceStopRun failed: %w", err)
	}

	a.t.Logf("API ForceStopRun response: success=%v, message=%s", resp.Msg.Success, resp.Msg.Message)
	return nil
}

func (a *TestAPI) GetProcessStates(ctx context.Context) error {
	if a.rpcClient == nil {
		return fmt.Errorf("ConnectRPC client not connected")
	}

	resp, err := a.rpcClient.GetProcessStates(ctx, connect.NewRequest(&pb.GetProcessStatesRequest{}))
	if err != nil {
		return fmt.Errorf("GetProcessStates failed: %w", err)
	}

	a.t.Logf("API GetProcessStates response: success=%v, message=%s", resp.Msg.Success, resp.Msg.Message)
	return nil
}

func (a *TestAPI) GetRunNumber(ctx context.Context) (int32, error) {
	if a.rpcClient == nil {
		return 0, fmt.Errorf("ConnectRPC client not connected")
	}

	resp, err := a.rpcClient.GetRunNumber(ctx, connect.NewRequest(&pb.GetRunNumberRequest{}))
	if err != nil {
		return 0, fmt.Errorf("GetRunNumber failed: %w", err)
	}

	return resp.Msg.RunNumber, nil
}

func (a *TestAPI) IsRunning() bool {
	if a.cmd == nil || a.cmd.Process == nil {
		return false
	}

	err := a.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

func (a *TestAPI) HealthCheck() error {
	url := fmt.Sprintf("http://127.0.0.1:%d/daq/health", a.port)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

func (a *TestAPI) Stop() {
	a.t.Logf("Stopping API process (PID %d)", a.cmd.Process.Pid)

	a.cancel()

	done := make(chan error, 1)
	go func() {
		done <- a.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		a.t.Logf("Timeout waiting for API process %d to exit, force killing", a.cmd.Process.Pid)
		_ = a.cmd.Process.Kill()
		_ = a.cmd.Wait()
	}

	time.Sleep(200 * time.Millisecond)
}

func StartTestAPIWithCentrifuge(t *testing.T, configFilename string, centrifugeCfg duck.CentrifugalConfiguration) *TestAPI {
	projectRoot := GetProjectRoot()
	apiBinary := filepath.Join(projectRoot, "bin", "api")

	BuildAPIBinary(t)

	if !filepath.IsAbs(configFilename) {
		configFilename = filepath.Join(projectRoot, configFilename)
	}

	configFilename = prepareTestConfig(t, configFilename)

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

	t.Logf("API config: Centrifuge=%s:%d", cfg.Centrifugal.Host, cfg.Centrifugal.Port)

	port := findAvailablePort(t)

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, apiBinary, "-config", configFilename, "-port", fmt.Sprintf("%d", port))
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startErr := cmd.Start()
	require.NoError(t, startErr, "Failed to start API process")

	t.Logf("Started API process with PID %d on port %d (with Centrifuge)", cmd.Process.Pid, port)

	testAPI := &TestAPI{
		cmd:            cmd,
		configFilename: configFilename,
		port:           port,
		cancel:         cancel,
		t:              t,
	}

	if err := testAPI.waitForServerReady(10 * time.Second); err != nil {
		t.Fatalf("API server not ready: %v", err)
	}

	testAPI.ConnectRPC()

	return testAPI
}
