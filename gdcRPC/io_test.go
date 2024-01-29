//go:build nohdf5

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"
)

// Helper function to create a test server with context
func createTestServerForIO(t *testing.T, runNumber int, gdcID int, host string, experiment string) *server {
	gdcConfig := testhelpers.NewTestGDCConfiguration(gdcID, host)
	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", runNumber)
	ctx = context.WithValue(ctx, "gdcConfiguration", &gdcConfig)
	ctx = context.WithValue(ctx, "experiment", experiment)
	cancelCtx := func() {}

	return &server{
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
}

// TestGetDecodedOutputFilename_ValidInput tests filename generation with valid inputs
func TestGetDecodedOutputFilename_ValidInput(t *testing.T) {
	// Setup
	s := createTestServerForIO(t, 123, 1, "gdc1", "next100")

	tests := []struct {
		name     string
		subrun   int
		trigger  int
		path     string
		expected string
	}{
		{
			name:     "Basic filename",
			subrun:   0,
			trigger:  1,
			path:     "/data/output",
			expected: "/data/output/run_123_0000_ldc1_trg1.waveforms.h5",
		},
		{
			name:     "Subrun 5",
			subrun:   5,
			trigger:  2,
			path:     "/tmp/output",
			expected: "/tmp/output/run_123_0005_ldc1_trg2.waveforms.h5",
		},
		{
			name:     "Subrun 9999",
			subrun:   9999,
			trigger:  0,
			path:     "/data",
			expected: "/data/run_123_9999_ldc1_trg0.waveforms.h5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getDecodedOutputFilename(s, tt.subrun, tt.trigger, tt.path)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetDecodedOutputFilename_MissingRunNumber tests error handling when runNumber is missing
func TestGetDecodedOutputFilename_MissingRunNumber(t *testing.T) {
	// Setup - context without runNumber
	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	ctx := context.Background()
	ctx = context.WithValue(ctx, "gdcConfiguration", &gdcConfig)
	cancelCtx := func() {}
	s := &server{
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	// Note: runNumber is NOT set

	// Act
	result, err := getDecodedOutputFilename(s, 0, 1, "/data")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, result)
}

// TestGetDecodedOutputFilename_MissingGDCConfiguration tests error handling when gdcConfiguration is missing
func TestGetDecodedOutputFilename_MissingGDCConfiguration(t *testing.T) {
	// Setup - context without gdcConfiguration
	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	cancelCtx := func() {}
	s := &server{
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	// Note: gdcConfiguration is NOT set

	// Act
	result, err := getDecodedOutputFilename(s, 0, 1, "/data")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, result)
}

// TestGetDecodedOutputFilename_GDCID tests that GDC ID is correctly included in filename
func TestGetDecodedOutputFilename_GDCID(t *testing.T) {
	// Setup
	s1 := createTestServerForIO(t, 123, 1, "gdc1", "next100")
	s2 := createTestServerForIO(t, 123, 5, "gdc5", "next100")

	// Act
	result1, err1 := getDecodedOutputFilename(s1, 0, 1, "/data")
	result2, err2 := getDecodedOutputFilename(s2, 0, 1, "/data")

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Contains(t, result1, "ldc1_")
	assert.Contains(t, result2, "ldc5_")
}

// TestOpenFile_SuccessfulCreation tests that files are created successfully
func TestOpenFile_SuccessfulCreation(t *testing.T) {
	// Setup
	s := createTestServerForIO(t, 123, 1, "testhost", "next100")

	// Use temp directory
	tempDir := t.TempDir()

	// Act
	file, err := openFile(s, 0, tempDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, file)

	// Verify file exists
	filename := fmt.Sprintf("%s/run_123.testhost.next100.00000.rd", tempDir)
	_, statErr := os.Stat(filename)
	assert.NoError(t, statErr, "File should exist")

	// Cleanup
	file.Close()
}

// TestOpenFile_SubrunNumber tests that subrun number is correctly included in filename
func TestOpenFile_SubrunNumber(t *testing.T) {
	// Setup
	s := createTestServerForIO(t, 456, 1, "testhost", "next100")

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		subrun      int
		expectedExt string
	}{
		{"Subrun 0", 0, "00000.rd"},
		{"Subrun 1", 1, "00001.rd"},
		{"Subrun 999", 999, "00999.rd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			file, err := openFile(s, tt.subrun, tempDir)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, file)

			// Verify filename has correct subrun
			filename := fmt.Sprintf("%s/run_456.testhost.next100.%s", tempDir, tt.expectedExt)
			_, statErr := os.Stat(filename)
			assert.NoError(t, statErr, "File should exist with correct subrun number")

			file.Close()
		})
	}
}

// TestOpenFile_MissingRunNumber tests error handling when runNumber is missing
func TestOpenFile_MissingRunNumber(t *testing.T) {
	// Setup - context without runNumber
	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "testhost")
	ctx := context.Background()
	ctx = context.WithValue(ctx, "gdcConfiguration", &gdcConfig)
	ctx = context.WithValue(ctx, "experiment", "next100")
	cancelCtx := func() {}
	s := &server{
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	// Note: runNumber is NOT set

	tempDir := t.TempDir()

	// Act
	file, err := openFile(s, 0, tempDir)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, file)
}

// TestOpenFile_MissingGDCConfiguration tests error handling when gdcConfiguration is missing
func TestOpenFile_MissingGDCConfiguration(t *testing.T) {
	// Setup - context without gdcConfiguration
	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "experiment", "next100")
	cancelCtx := func() {}
	s := &server{
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	// Note: gdcConfiguration is NOT set

	tempDir := t.TempDir()

	// Act
	file, err := openFile(s, 0, tempDir)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, file)
}

// TestOpenFile_MissingExperiment tests error handling when experiment is missing
func TestOpenFile_MissingExperiment(t *testing.T) {
	// Setup - context without experiment
	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "testhost")
	ctx := context.Background()
	ctx = context.WithValue(ctx, "runNumber", 123)
	ctx = context.WithValue(ctx, "gdcConfiguration", &gdcConfig)
	cancelCtx := func() {}
	s := &server{
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	// Note: experiment is NOT set

	tempDir := t.TempDir()

	// Act
	file, err := openFile(s, 0, tempDir)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, file)
}

// TestWriteBinaryData_Success tests successful binary data write
func TestWriteBinaryData_Success(t *testing.T) {
	// Setup
	s := createTestServerForIO(t, 0, 1, "testhost", "next100")
	tempFile, err := os.CreateTemp(t.TempDir(), "test-*.rd")
	require.NoError(t, err)
	defer tempFile.Close()

	testData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	wrapper := &osFileWrapper{file: tempFile}

	// Act
	err = writeBinaryData(s, testData, wrapper)

	// Assert
	require.NoError(t, err)

	// Verify data was written
	tempFile.Close()
	data, err := os.ReadFile(tempFile.Name())
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

// TestWriteBinaryData_EmptyData tests writing empty data
func TestWriteBinaryData_EmptyData(t *testing.T) {
	// Setup
	s := createTestServerForIO(t, 0, 1, "testhost", "next100")
	tempFile, err := os.CreateTemp(t.TempDir(), "test-*.rd")
	require.NoError(t, err)
	defer tempFile.Close()

	emptyData := []byte{}
	wrapper := &osFileWrapper{file: tempFile}

	// Act
	err = writeBinaryData(s, emptyData, wrapper)

	// Assert
	require.NoError(t, err)

	// Verify file is empty
	tempFile.Close()
	stat, err := os.Stat(tempFile.Name())
	require.NoError(t, err)
	assert.Equal(t, int64(0), stat.Size())
}

// TestWriteBinaryData_LargeData tests writing large data
func TestWriteBinaryData_LargeData(t *testing.T) {
	// Setup
	s := createTestServerForIO(t, 0, 1, "testhost", "next100")
	tempFile, err := os.CreateTemp(t.TempDir(), "test-*.rd")
	require.NoError(t, err)
	defer tempFile.Close()

	// Create 1MB of test data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	wrapper := &osFileWrapper{file: tempFile}

	// Act
	err = writeBinaryData(s, largeData, wrapper)

	// Assert
	require.NoError(t, err)

	// Verify all data was written
	tempFile.Close()
	stat, err := os.Stat(tempFile.Name())
	require.NoError(t, err)
	assert.Equal(t, int64(1024*1024), stat.Size())
}

// TestStopOnIOError_CancelsContext tests that IO errors cancel the context
func TestStopOnIOError_CancelsContext(t *testing.T) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	cancelCtx := func() {}
	s := &server{
		ctx:            context.WithValue(ctx, "cancelCtx", cancel),
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}

	testError := fmt.Errorf("test IO error")

	// Act
	stopOnIOError(s, testError)

	// Assert - context should be cancelled
	select {
	case <-s.getContext().Done():
		// Expected - context was cancelled
	default:
		t.Fatal("Context should be cancelled after IO error")
	}
}

// TestStopOnIOError_MissingCancelFunc tests graceful handling when cancelCtx is missing
func TestStopOnIOError_MissingCancelFunc(t *testing.T) {
	// Setup - context without cancelCtx
	ctx := context.Background()
	cancelCtx := func() {}
	s := &server{
		ctx:            ctx,
		cancelCtx:      cancelCtx,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}

	testError := fmt.Errorf("test IO error")

	// Act - should not panic
	assert.NotPanics(t, func() {
		stopOnIOError(s, testError)
	})
}

// TestCreateFile_ComprehensivePath tests file creation with various path configurations
func TestCreateFile_ComprehensivePath(t *testing.T) {
	// Setup
	s := createTestServerForIO(t, 789, 1, "testhost", "next100")

	// Create nested directory structure
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "subdir1", "subdir2")
	err := os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	// Act
	file, err := openFile(s, 0, nestedDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, file)

	expectedPath := filepath.Join(nestedDir, "run_789.testhost.next100.00000.rd")
	_, statErr := os.Stat(expectedPath)
	assert.NoError(t, statErr, "File should exist in nested directory")

	file.Close()
}

// TestOpenFile_GDCConfigurationHost tests that GDC host name is used in filename
func TestOpenFile_GDCConfigurationHost(t *testing.T) {
	// Setup
	gdcConfig1 := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	gdcConfig1.Host = "host1"
	ctx1 := context.Background()
	ctx1 = context.WithValue(ctx1, "runNumber", 100)
	ctx1 = context.WithValue(ctx1, "gdcConfiguration", &gdcConfig1)
	ctx1 = context.WithValue(ctx1, "experiment", "test")
	cancelCtx1 := func() {}
	s1 := &server{
		ctx:            ctx1,
		cancelCtx:      cancelCtx1,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}

	gdcConfig2 := testhelpers.NewTestGDCConfiguration(2, "gdc2")
	gdcConfig2.Host = "host2"
	ctx2 := context.Background()
	ctx2 = context.WithValue(ctx2, "runNumber", 100)
	ctx2 = context.WithValue(ctx2, "gdcConfiguration", &gdcConfig2)
	ctx2 = context.WithValue(ctx2, "experiment", "test")
	cancelCtx2 := func() {}
	s2 := &server{
		ctx:            ctx2,
		cancelCtx:      cancelCtx2,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}

	tempDir := t.TempDir()

	// Act
	file1, err1 := openFile(s1, 0, tempDir)
	file2, err2 := openFile(s2, 0, tempDir)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	defer file1.Close()
	defer file2.Close()

	// Verify both files exist with correct hostnames
	_, statErr1 := os.Stat(filepath.Join(tempDir, "run_100.host1.test.00000.rd"))
	_, statErr2 := os.Stat(filepath.Join(tempDir, "run_100.host2.test.00000.rd"))
	assert.NoError(t, statErr1, "File with host1 should exist")
	assert.NoError(t, statErr2, "File with host2 should exist")
}
