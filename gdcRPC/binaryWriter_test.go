//go:build nohdf5

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"
)

// Helper function to create a test server with context
func createTestServerWithContext(t *testing.T, runNumber int, gdcID int, host string, experiment string) *server {
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

// TestBinaryWriter_ChannelClosed_Cleanup tests that binaryWriter cleans up when channel is closed
func TestBinaryWriter_ChannelClosed_Cleanup(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 123, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 10)

	done := make(chan bool, 1)

	// Act - start binaryWriter in goroutine
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write some data
	testData := []byte{0x01, 0x02, 0x03, 0x04}
	dataCh <- testData

	// Give writer time to process
	time.Sleep(50 * time.Millisecond)

	// Close channel to signal writer to stop
	close(dataCh)

	// Assert - writer should exit
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to exit after channel close")
	}

	// Verify file was created and data was written
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "One file should be created")

	// Verify file content
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

// mockErrorWriter is a mock implementation of fileWriter that always returns an error on Write
type mockErrorWriter struct {
	writeCount int
	closeCount int
}

func (m *mockErrorWriter) Write(data []byte) (int, error) {
	m.writeCount++
	return 0, fmt.Errorf("simulated write error")
}

func (m *mockErrorWriter) Close() error {
	m.closeCount++
	return nil
}

// mockSuccessWriter is a mock implementation of fileWriter that succeeds on Write
type mockSuccessWriter struct {
	writeCount int
	closeCount int
	bytesWritten int
}

func (m *mockSuccessWriter) Write(data []byte) (int, error) {
	m.writeCount++
	m.bytesWritten += len(data)
	return len(data), nil
}

func (m *mockSuccessWriter) Close() error {
	m.closeCount++
	return nil
}

// TestBinaryWriter_WriteError_HandlesGracefully tests error handling during write
func TestBinaryWriter_WriteError_HandlesGracefully(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 0, 1, "testhost", "next100")
	dataCh := make(chan []byte, 10)
	done := make(chan bool, 1)

	// Create mock writer that will return an error on Write
	mockWriter := &mockErrorWriter{}

	// Act - start binaryWriterWithFile with mock writer
	go func() {
		binaryWriterWithFile(s, dataCh, mockWriter)
		done <- true
	}()

	// Write data - this will fail with the mock error
	testData := []byte{0x01, 0x02, 0x03, 0x04}
	dataCh <- testData

	// Assert - writer should exit due to write error
	select {
	case <-done:
		// Expected - writer exited on error
		assert.Equal(t, 1, mockWriter.writeCount, "Write should have been called once")
		assert.Equal(t, 1, mockWriter.closeCount, "Close should be called even on error (defer ensures cleanup)")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to exit after write error")
	}

	// Close channel to clean up
	close(dataCh)
}

// TestBinaryWriter_ContextCancellation_StopsGracefully tests that context cancellation stops the writer
func TestBinaryWriter_ContextCancellation_StopsGracefully(t *testing.T) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	s := createTestServerWithContext(t, 0, 1, "testhost", "next100")
	s.ctx = ctx
	s.cancelCtx = cancel

	dataCh := make(chan []byte, 10)
	done := make(chan bool, 1)

	// Create mock writer that succeeds to track writes
	mockWriter := &mockSuccessWriter{}

	// Act - start binaryWriterWithFile
	go func() {
		binaryWriterWithFile(s, dataCh, mockWriter)
		done <- true
	}()

	// Write some data successfully
	testData := []byte{0x01, 0x02, 0x03, 0x04}
	dataCh <- testData

	// Give writer time to process
	time.Sleep(10 * time.Millisecond)

	// Cancel context - writer should exit gracefully
	cancel()

	// Assert - writer should exit
	select {
	case <-done:
		// Expected - writer exited on context cancellation
		assert.Equal(t, 1, mockWriter.writeCount, "Write should have been called once")
		assert.Equal(t, 1, mockWriter.closeCount, "Close should be called on context cancellation")
		assert.Equal(t, len(testData), mockWriter.bytesWritten, "Data should be written successfully")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to exit after context cancellation")
	}

	// Close channel to clean up
	close(dataCh)
}

// TestBinaryWriter_MultipleEvents tests writing multiple events
func TestBinaryWriter_MultipleEvents(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 456, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 100)

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write multiple events
	numEvents := 10
	testData := make([][]byte, numEvents)
	for i := 0; i < numEvents; i++ {
		testData[i] = make([]byte, 100)
		for j := range testData[i] {
			testData[i][j] = byte((i + j) % 256)
		}
		dataCh <- testData[i]
	}

	// Give writer time to process
	time.Sleep(100 * time.Millisecond)

	// Close channel
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "One file should be created")

	// Verify file content
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)

	expectedSize := numEvents * 100
	assert.Equal(t, expectedSize, len(data), "All events should be written")
}

// TestBinaryWriter_EmptyChannel tests writer with no data
func TestBinaryWriter_EmptyChannel(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 789, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 10)

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Give writer time to create file
	time.Sleep(50 * time.Millisecond)

	// Close channel without writing any data
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify file was created (even if empty)
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "File should be created even with no data")

	// Verify file is empty
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, 0, len(data), "File should be empty")
}

// TestBinaryWriter_LargeEventData tests writing large events
func TestBinaryWriter_LargeEventData(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 999, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 10)

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write large event (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	dataCh <- largeData

	// Give writer time to process
	time.Sleep(100 * time.Millisecond)

	// Close channel
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "File should be created")

	// Verify file content
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, len(largeData), len(data), "All data should be written")
	assert.Equal(t, largeData, data, "Data should match")
}

// TestBinaryWriter_SubrunNumber tests that subrun number is included in filename
func TestBinaryWriter_SubrunNumber(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 111, 1, "testhost", "next100")

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		subrun      int
		expectedExt string
	}{
		{"Subrun 0", 0, "00000.rd"},
		{"Subrun 5", 5, "00005.rd"},
		{"Subrun 123", 123, "00123.rd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataCh := make(chan []byte, 10)
			done := make(chan bool, 1)

			// Act - start binaryWriter
			go func() {
				binaryWriter(s, dataCh, tt.subrun, tempDir)
				done <- true
			}()

			// Write some data
			dataCh <- []byte{0x01, 0x02}
			time.Sleep(50 * time.Millisecond)

			// Close channel
			close(dataCh)

			// Wait for writer to finish
			select {
			case <-done:
				// Expected
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout waiting for binaryWriter to finish")
			}

			// Verify filename
			expectedFilename := filepath.Join(tempDir, "run_111.testhost.next100."+tt.expectedExt)
			_, err := os.Stat(expectedFilename)
			assert.NoError(t, err, "File with correct subrun number should exist")
		})
	}
}

// TestBinaryWriter_ConcurrentWrites tests concurrent write operations
func TestBinaryWriter_ConcurrentWrites(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 222, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 1000)

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write data from multiple goroutines
	numGoroutines := 10
	writesPerGoroutine := 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				testData := make([]byte, 50)
				testData[0] = byte(id)
				testData[1] = byte(j)
				dataCh <- testData
			}
		}(i)
	}

	// Wait for all goroutines to finish sending data
	wg.Wait()

	// Give writer time to process
	time.Sleep(200 * time.Millisecond)

	// Close channel
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "File should be created")

	// Verify file size
	expectedSize := numGoroutines * writesPerGoroutine * 50
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, expectedSize, len(data), "All data should be written")
}

// TestBinaryWriter_FilenameFormat tests that filename format is correct
func TestBinaryWriter_FilenameFormat(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 333, 5, "myhost", "testexp")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 10)

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write some data
	dataCh <- []byte{0x01}
	time.Sleep(50 * time.Millisecond)

	// Close channel
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify filename format
	expectedFilename := filepath.Join(tempDir, "run_333.testhost.testexp.00000.rd")
	_, err := os.Stat(expectedFilename)
	assert.NoError(t, err, "File with correct format should exist")
}

// TestBinaryWriter_ChannelBlocking tests that writer blocks when channel is full
func TestBinaryWriter_ChannelBlocking(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 444, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 5) // Small buffer to test blocking

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write more data than channel buffer size
	numWrites := 20
	for i := 0; i < numWrites; i++ {
		testData := make([]byte, 100)
		testData[0] = byte(i)
		dataCh <- testData
	}

	// Give writer time to process
	time.Sleep(100 * time.Millisecond)

	// Close channel
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify all data was written
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "File should be created")

	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	expectedSize := numWrites * 100
	assert.Equal(t, expectedSize, len(data), "All data should be written")
}

// TestBinaryWriter_VaryingDataSizes tests writing events of varying sizes
func TestBinaryWriter_VaryingDataSizes(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 555, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 100)

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write events of varying sizes
	sizes := []int{10, 100, 1000, 5000, 100, 50}
	totalSize := 0
	for _, size := range sizes {
		testData := make([]byte, size)
		dataCh <- testData
		totalSize += size
	}

	// Give writer time to process
	time.Sleep(100 * time.Millisecond)

	// Close channel
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify file size
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "File should be created")

	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, totalSize, len(data), "All data should be written")
}

// TestBinaryWriter_ZeroByteEvent tests handling of zero-byte events
func TestBinaryWriter_ZeroByteEvent(t *testing.T) {
	// Setup
	s := createTestServerWithContext(t, 666, 1, "testhost", "next100")

	tempDir := t.TempDir()
	dataCh := make(chan []byte, 10)

	done := make(chan bool, 1)

	// Act - start binaryWriter
	go func() {
		binaryWriter(s, dataCh, 0, tempDir)
		done <- true
	}()

	// Write zero-byte event
	dataCh <- []byte{}

	// Write normal event
	dataCh <- []byte{0x01, 0x02, 0x03}

	// Give writer time to process
	time.Sleep(50 * time.Millisecond)

	// Close channel
	close(dataCh)

	// Wait for writer to finish
	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for binaryWriter to finish")
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.rd"))
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "File should be created")

	// Verify file content (should be 3 bytes)
	data, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, 3, len(data), "Zero-byte event should be handled correctly")
}
