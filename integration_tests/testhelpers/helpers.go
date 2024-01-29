package testhelpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// GetProjectRoot returns the project root directory
func GetProjectRoot() string {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}

	// Walk up until we find go.mod
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return wd
		}
		dir = parent
	}
}

// ReplaceInConfig replaces a key-value pair in YAML config string
// This is a simple implementation for testing purposes
func ReplaceInConfig(config, key, value string) (string, error) {
	// Try simple string replacement for "key: value" pattern
	// This is a basic implementation - for complex cases, use proper YAML parsing
	searchPattern := fmt.Sprintf("%s:", key)

	lines := strings.Split(config, "\n")
	for i, line := range lines {
		if strings.Contains(line, searchPattern) && strings.Contains(line, ":") {
			// Replace the value after the colon
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				// Preserve indentation
				indent := ""
				for _, c := range parts[0] {
					if c == ' ' || c == '\t' {
						indent += string(c)
					} else {
						break
					}
				}
				lines[i] = fmt.Sprintf("%s%s: %s", indent, key, value)
				return strings.Join(lines, "\n"), nil
			}
		}
	}

	// If not found as exact match, try to replace within lines
	for i, line := range lines {
		if strings.Contains(line, searchPattern) {
			// Found a line with this key, try to replace value
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				lines[i] = fmt.Sprintf("%s: %s", parts[0], value)
				return strings.Join(lines, "\n"), nil
			}
		}
	}

	// Key not found - return original and note it
	return config, fmt.Errorf("key %s not found in config", key)
}

// prepareTestConfig copies the config file to a temp directory to avoid modifying git-controlled files
// Returns the path to the temp config file which should be used instead of the original
func prepareTestConfig(t *testing.T, configFilename string) string {
	// Read original config
	b, err := os.ReadFile(configFilename)
	require.NoError(t, err, "Failed to read config file: %s", configFilename)

	// Create temp directory for test configs
	tempDir := filepath.Join(os.TempDir(), "duck-test-configs", fmt.Sprintf("%d", time.Now().UnixNano()))
	err = os.MkdirAll(tempDir, 0755)
	require.NoError(t, err, "Failed to create temp config directory")

	// Write config to temp location
	tempConfigPath := filepath.Join(tempDir, filepath.Base(configFilename))
	err = os.WriteFile(tempConfigPath, b, 0644)
	require.NoError(t, err, "Failed to write temp config file")

	// Register cleanup to remove temp directory after test
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempConfigPath
}
