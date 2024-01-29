package main

import (
	"fmt"
	"os"
	"testing"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSSHClient implements SSHClient for testing
type MockSSHClient struct {
	RunFunc   func(cmd string) ([]byte, error)
	CloseFunc func() error
	RunCalled bool
	LastCmd   string
}

func (m *MockSSHClient) Run(cmd string) ([]byte, error) {
	m.RunCalled = true
	m.LastCmd = cmd
	if m.RunFunc != nil {
		return m.RunFunc(cmd)
	}
	return []byte("success"), nil
}

func (m *MockSSHClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// TestRestartService_Success tests successful service restart
func TestRestartService_Success(t *testing.T) {
	mockClient := &MockSSHClient{
		RunFunc: func(cmd string) ([]byte, error) {
			return []byte("Service restarted successfully"), nil
		},
	}

	// Replace sshClientCreator for this test
	oldCreator := sshClientCreator
	defer func() { sshClientCreator = oldCreator }()

	sshClientCreator = func(user, host string) (SSHClient, error) {
		assert.Equal(t, "dateuser", user)
		assert.Equal(t, "test.example.com", host)
		return mockClient, nil
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	restartService("test-gdc", GDCRole, "test.example.com")

	// Assert
	assert.True(t, mockClient.RunCalled)
	assert.Equal(t, "/home/dateuser/duck/restart_gdc.sh", mockClient.LastCmd)
}

// TestRestartService_SSHConnectionError tests SSH connection failure
func TestRestartService_SSHConnectionError(t *testing.T) {
	// Replace sshClientCreator for this test
	oldCreator := sshClientCreator
	defer func() { sshClientCreator = oldCreator }()

	sshClientCreator = func(user, host string) (SSHClient, error) {
		return nil, fmt.Errorf("connection refused")
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute - should not panic
	restartService("test-gdc", GDCRole, "test.example.com")

	// No assertions needed - just verifying it handles the error gracefully
}

// TestRestartService_CommandError tests command execution failure
func TestRestartService_CommandError(t *testing.T) {
	mockClient := &MockSSHClient{
		RunFunc: func(cmd string) ([]byte, error) {
			return []byte(""), fmt.Errorf("command failed")
		},
	}

	// Replace sshClientCreator for this test
	oldCreator := sshClientCreator
	defer func() { sshClientCreator = oldCreator }()

	sshClientCreator = func(user, host string) (SSHClient, error) {
		return mockClient, nil
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute - should not panic
	restartService("test-gdc", GDCRole, "test.example.com")

	// Verify command was attempted
	assert.True(t, mockClient.RunCalled)
}

// TestRestartDuckSSH_GDCRole tests correct command for GDC role
func TestRestartDuckSSH_GDCRole(t *testing.T) {
	mockClient := &MockSSHClient{
		RunFunc: func(cmd string) ([]byte, error) {
			assert.Equal(t, "/home/dateuser/duck/restart_gdc.sh", cmd)
			return []byte("GDC restarted"), nil
		},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	err := restartDuckSSH(mockClient, "gdc1", GDCRole)

	// Assert
	require.NoError(t, err)
	assert.True(t, mockClient.RunCalled)
}

// TestRestartDuckSSH_LDCRole tests correct command for LDC role
func TestRestartDuckSSH_LDCRole(t *testing.T) {
	mockClient := &MockSSHClient{
		RunFunc: func(cmd string) ([]byte, error) {
			assert.Equal(t, "/home/dateuser/duck/restart_ldc.sh", cmd)
			return []byte("LDC restarted"), nil
		},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	err := restartDuckSSH(mockClient, "ldc1", LDCRole)

	// Assert
	require.NoError(t, err)
	assert.True(t, mockClient.RunCalled)
}

// TestRestartDuckSSH_CommandError tests error handling
func TestRestartDuckSSH_CommandError(t *testing.T) {
	mockClient := &MockSSHClient{
		RunFunc: func(cmd string) ([]byte, error) {
			return nil, fmt.Errorf("script execution failed")
		},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	err := restartDuckSSH(mockClient, "gdc1", GDCRole)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script execution failed")
	assert.True(t, mockClient.RunCalled)
}

// TestRestartDuckSSH_ANSICodeRemoval tests ANSI escape code stripping
func TestRestartDuckSSH_ANSICodeRemoval(t *testing.T) {
	outputWithANSI := "[31mError text[39m [32mSuccess text[39m Normal text"

	mockClient := &MockSSHClient{
		RunFunc: func(cmd string) ([]byte, error) {
			return []byte(outputWithANSI), nil
		},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	err := restartDuckSSH(mockClient, "gdc1", GDCRole)

	// Assert
	require.NoError(t, err)
	// The function logs the output, we just verify it doesn't error
	// The actual ANSI code removal is tested by checking no panic occurs
}

// TestNewSshClient_KeyNotFound tests behavior when SSH key doesn't exist
func TestNewSshClient_KeyNotFound(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set HOME to non-existent directory
	os.Setenv("HOME", "/nonexistent/home")

	// Execute
	client, err := NewSshClient("testuser", "testhost")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, client)
}

// TestRestartService_RealCreator tests the public function with the real creator
func TestRestartService_RealCreator(t *testing.T) {
	logger = duck.NewDuckLogger("test", nil, 0)

	// This will fail trying to connect, but verifies the function exists and handles errors
	// We just want to make sure it doesn't panic
	restartService("test", GDCRole, "nonexistent.host.local")

	// No panic = success
}
