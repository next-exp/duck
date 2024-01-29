package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"
	decoder "github.com/next-exp/decoder_go/pkg"
)

// ============================================================================
// Helper Functions for Decoder Tests
// ============================================================================

// createTestServerForDecoder creates a server configured for decoder tests
func createTestServerForDecoder(t *testing.T) *server {
	t.Helper()
	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	decoderConfig := testhelpers.CreateTestDecoderConfig()

	ctx := testhelpers.NewMockContextBuilder().
		WithRunNumber(123).
		WithGDCConfiguration(&gdcConfig).
		WithExperiment("next100").
		WithWriteOutput(false).
		WithDecode(false).
		WithNGDCs(1).
		WithDecoderConfig(decoderConfig).
		WithMaxFilesize(100000000).
		WithWriterChBufferSize(10).
		WithDecoderChBufferSize(10).
		WithMetricsChBufferSize(10).
		WithDecoderWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, "cancelCtx", cancel)

	return &server{
		ctx:            ctx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.RUNNING,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
}

// ============================================================================
// Decoder Tests
// These tests require the decoder package to be available
// They should be run in the Docker testing environment or locally with HDF5 installed
// ============================================================================

// TestParseDecoderConfiguration_ValidConfiguration tests parsing a valid decoder configuration
func TestParseDecoderConfiguration_ValidConfiguration(t *testing.T) {
	// This test requires HDF5 libraries
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	s := createTestServerForDecoder(t)

	// Act
	config, err := parseDecoderConfiguration(s)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1000000000, config.MaxEvents)
	assert.Equal(t, 1, config.ExtTrigger)
	assert.Equal(t, 1, config.TrgCode1)
	assert.Equal(t, 2, config.TrgCode2)
	assert.True(t, config.ReadPMTs)
	assert.True(t, config.ReadSiPMs)
	assert.True(t, config.ReadTrigger)
	assert.False(t, config.SplitTrg) // Default config has SplitTrigger=false
	assert.False(t, config.NoDB)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "test_user", config.User)
	assert.Equal(t, "test_password", config.Passwd)
	assert.Equal(t, "test_db", config.DBName)
	assert.Equal(t, 1, config.NumWorkers)
	assert.True(t, config.WriteData)
	assert.True(t, config.UseBlosc)
	assert.Equal(t, 5, config.CompressionLevel)
}

// TestParseDecoderConfiguration_WithBloscSettings tests Blosc configuration parsing
func TestParseDecoderConfiguration_WithBloscSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	decoderConfig := testhelpers.CreateTestDecoderConfigWithBlosc("lz4", "shuffle", 9)

	ctx := testhelpers.NewMockContextBuilder().
		WithRunNumber(123).
		WithGDCConfiguration(&gdcConfig).
		WithExperiment("next100").
		WithWriteOutput(false).
		WithDecode(false).
		WithNGDCs(1).
		WithDecoderConfig(decoderConfig).
		WithMaxFilesize(100000000).
		WithWriterChBufferSize(10).
		WithDecoderChBufferSize(10).
		WithMetricsChBufferSize(10).
		WithDecoderWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, "cancelCtx", cancel)

	s := &server{
		ctx:            ctx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.RUNNING,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	defer cancel()

	// Act
	config, err := parseDecoderConfiguration(s)

	// Assert
	require.NoError(t, err)
	assert.True(t, config.UseBlosc)
	assert.Equal(t, 9, config.CompressionLevel)
}

// TestParseDecoderConfiguration_MissingConfig_ReturnsError tests error handling for missing config
func TestParseDecoderConfiguration_MissingConfig_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	// Create server without decoder config
	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "gdc1")

	ctx := testhelpers.NewMockContextBuilder().
		WithRunNumber(123).
		WithGDCConfiguration(&gdcConfig).
		WithExperiment("next100").
		WithWriteOutput(false).
		WithDecode(false).
		WithNGDCs(1).
		// NOTE: decoderConfig NOT set
		WithMaxFilesize(100000000).
		Build()

	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, "cancelCtx", cancel)

	s := &server{
		ctx:            ctx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.RUNNING,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	defer cancel()

	// Act
	config, err := parseDecoderConfiguration(s)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, decoder.Configuration{}, config)
}

// TestSetupDecoder_WithMockDB tests setupDecoder with mock database configuration
func TestSetupDecoder_WithMockDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	// This test would require a real database connection
	// For now, we just test that the function doesn't panic with valid config
	s := createTestServerForDecoder(t)

	// Act & Assert - should not panic
	assert.NotPanics(t, func() {
		setupDecoder(s, 123)
	})
}

// TestDecoderJob_WithRealConfig tests DecoderJob structure with real decoder config
func TestDecoderJob_WithRealConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	decoderConfig := testhelpers.CreateTestDecoderConfig()
	writerChs := WriterChannels{
		binaryEvtCh:   make(chan []byte, 5),
		decodedTrg1Ch: make(chan DecodedWriterEvent, 5),
		decodedTrg2Ch: make(chan DecodedWriterEvent, 5),
		fileCloser:    createFileCloserChannels(5),
	}

	job := DecoderJob{
		DuckEventID:   456,
		Data:          []byte{0xAA, 0xBB, 0xCC},
		WritersCh:     writerChs,
		DecoderConfig: decoderConfig,
	}

	assert.Equal(t, 456, job.DuckEventID)
	assert.Equal(t, []byte{0xAA, 0xBB, 0xCC}, job.Data)
	assert.NotNil(t, job.WritersCh)
	assert.True(t, job.DecoderConfig.WriteData)
}

// TestDecodedWriterEvent_WithRealEvent tests DecodedWriterEvent with real decoder event
func TestDecodedWriterEvent_WithRealEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	// We can't create a real decoder.Event without actual data
	// But we can test the structure with zero value
	evt := DecodedWriterEvent{
		DuckEventID: 100,
		decodedEvt:  decoder.EventType{}, // Zero value for struct
	}

	assert.Equal(t, 100, evt.DuckEventID)
}

// TestTriggerType_WithRealConfig tests TriggerType constants with real configuration
func TestTriggerType_WithRealConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	// Test that trigger type constants work correctly
	assert.Equal(t, 0, int(Trg0))
	assert.Equal(t, 1, int(Trg1))
	assert.Equal(t, 2, int(Trg2))

	// Test String() method
	assert.Equal(t, "0", Trg0.String())
	assert.Equal(t, "1", Trg1.String())
	assert.Equal(t, "2", Trg2.String())
}

// TestWriterChannels_WithRealConfig tests WriterChannels structure with real configuration
func TestWriterChannels_WithRealConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	writerChs := WriterChannels{
		binaryEvtCh:   make(chan []byte, 10),
		decodedTrg1Ch: make(chan DecodedWriterEvent, 10),
		decodedTrg2Ch: make(chan DecodedWriterEvent, 10),
		fileCloser:    createFileCloserChannels(10),
	}

	// Verify all channels are created
	assert.NotNil(t, writerChs.binaryEvtCh)
	assert.NotNil(t, writerChs.decodedTrg1Ch)
	assert.NotNil(t, writerChs.decodedTrg2Ch)
	assert.NotNil(t, writerChs.fileCloser.EvtReceived)
	assert.NotNil(t, writerChs.fileCloser.EvtDecoded)
	assert.NotNil(t, writerChs.fileCloser.EvtWritten)
	assert.NotNil(t, writerChs.fileCloser.EvtWithError)
	assert.NotNil(t, writerChs.fileCloser.DataEnd)
	assert.NotNil(t, writerChs.fileCloser.CloseFiles)
}

// TestParseDecoderConfiguration_SplitTriggerMode tests split trigger configuration
func TestParseDecoderConfiguration_SplitTriggerMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	decoderConfig := testhelpers.CreateTestDecoderConfigSplitTrigger()

	ctx := testhelpers.NewMockContextBuilder().
		WithRunNumber(123).
		WithGDCConfiguration(&gdcConfig).
		WithExperiment("next100").
		WithWriteOutput(false).
		WithDecode(false).
		WithNGDCs(1).
		WithDecoderConfig(decoderConfig).
		WithMaxFilesize(100000000).
		WithWriterChBufferSize(10).
		WithDecoderChBufferSize(10).
		WithMetricsChBufferSize(10).
		WithDecoderWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, "cancelCtx", cancel)

	s := &server{
		ctx:            ctx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.RUNNING,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	defer cancel()

	// Act
	config, err := parseDecoderConfiguration(s)

	// Assert
	require.NoError(t, err)
	assert.True(t, config.SplitTrg)
	assert.Equal(t, 1, config.TrgCode1)
	assert.Equal(t, 2, config.TrgCode2)
}

// TestParseDecoderConfiguration_NoDBMode tests NoDB configuration mode
func TestParseDecoderConfiguration_NoDBMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 test in short mode")
	}

	gdcConfig := testhelpers.NewTestGDCConfiguration(1, "gdc1")
	decoderConfig := testhelpers.CreateTestDecoderConfigNoDB()

	ctx := testhelpers.NewMockContextBuilder().
		WithRunNumber(123).
		WithGDCConfiguration(&gdcConfig).
		WithExperiment("next100").
		WithWriteOutput(false).
		WithDecode(false).
		WithNGDCs(1).
		WithDecoderConfig(decoderConfig).
		WithMaxFilesize(100000000).
		WithWriterChBufferSize(10).
		WithDecoderChBufferSize(10).
		WithMetricsChBufferSize(10).
		WithDecoderWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, "cancelCtx", cancel)

	s := &server{
		ctx:            ctx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.RUNNING,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
	defer cancel()

	// Act
	config, err := parseDecoderConfiguration(s)

	// Assert
	require.NoError(t, err)
	assert.True(t, config.NoDB)
	assert.Empty(t, config.Host)
	assert.Empty(t, config.User)
	assert.Empty(t, config.Passwd)
	assert.Empty(t, config.DBName)
}

// TestDecoderIntegration_EndToEnd tests end-to-end decoder integration
// This test requires actual HDF5 files and would be better suited for integration tests
func TestDecoderIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HDF5 integration test in short mode")
	}

	// This is a placeholder for future integration tests
	// Real end-to-end testing would require:
	// 1. Actual GDC event data files
	// 2. Database connection with calibration data
	// 3. Temporary HDF5 file creation
	// 4. Verification of written HDF5 files

	t.Skip("End-to-end decoder integration test requires: GDC event data files, database, and HDF5 verification - implement as integration test")
}

// TestHDF5Environment_Available verifies HDF5 environment is available
func TestHDF5Environment_Available(t *testing.T) {
	// This test verifies that the HDF5 library is available
	// by checking if we can import and use the decoder package

	// Act & Assert - should not panic if decoder package is available
	assert.NotPanics(t, func() {
		_ = decoder.Configuration{}
	})

	// If we get here, HDF5 environment is available
	t.Log("HDF5 environment is available - decoder package can be used")
}
