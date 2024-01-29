package testhelpers

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// MockTCPConn is a mock implementation of net.Conn for testing
type MockTCPConn struct {
	net.Conn                          // Embedded to implement net.Conn interface
	ReadData   [][]byte               // Data chunks to return on Read
	ReadIndex  int                    // Current position in ReadData
	CloseCalled bool                  // Track if Close was called
	ReadDelay  time.Duration          // Optional delay before each read
	ReadError  error                  // Optional error to return on Read
	mu         sync.Mutex             // Protect concurrent access
}

// NewMockTCPConn creates a new mock TCP connection
func NewMockTCPConn(data [][]byte) *MockTCPConn {
	return &MockTCPConn{
		ReadData:  data,
		ReadIndex: 0,
	}
}

// Read reads data from the mock connection
func (m *MockTCPConn) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ReadDelay > 0 {
		time.Sleep(m.ReadDelay)
	}

	// Return error if set (persists until data is available or explicitly cleared)
	if m.ReadError != nil {
		return 0, m.ReadError
	}

	if m.ReadIndex >= len(m.ReadData) {
		return 0, io.EOF
	}

	data := m.ReadData[m.ReadIndex]
	copied := copy(b, data)
	m.ReadIndex++

	// If the buffer was smaller than the data chunk, we need to return the remaining data next time
	if copied < len(data) {
		// Insert the remaining data at the next index
		remaining := data[copied:]
		m.ReadData = append(m.ReadData[:m.ReadIndex+1], m.ReadData[m.ReadIndex+1:]...)
		m.ReadData[m.ReadIndex] = remaining
	}

	return copied, nil
}

// SetReadError sets an error to be returned on Read calls
func (m *MockTCPConn) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReadError = err
}

// Close marks the connection as closed
func (m *MockTCPConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseCalled = true
	return nil
}

// LocalAddr returns a mock local address
func (m *MockTCPConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 6005,
	}
}

// RemoteAddr returns a mock remote address
func (m *MockTCPConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.2"),
		Port: 6006,
	}
}

// SetDeadline is a no-op for the mock
func (m *MockTCPConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is a no-op for the mock
func (m *MockTCPConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is a no-op for the mock
func (m *MockTCPConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// Write is a no-op for the mock (returns success)
func (m *MockTCPConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

// MockFileCloser tracks file operations for testing
type MockFileCloser struct {
	Filename      string
	CloseCalled   bool
	WriteCalls    int
	WriteData     [][]byte
	WriteErrors   []error
	CloseError    error
	OpenError     error
	mu            sync.Mutex
}

// NewMockFileCloser creates a new mock file closer
func NewMockFileCloser(filename string) *MockFileCloser {
	return &MockFileCloser{
		Filename:    filename,
		CloseCalled: false,
		WriteCalls:  0,
		WriteData:   make([][]byte, 0),
		WriteErrors: make([]error, 0),
	}
}

// MockWrite simulates writing data to the file
func (m *MockFileCloser) MockWrite(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WriteCalls++
	m.WriteData = append(m.WriteData, make([]byte, len(data)))
	copy(m.WriteData[len(m.WriteData)-1], data)

	if len(m.WriteErrors) > 0 {
		err := m.WriteErrors[0]
		m.WriteErrors = m.WriteErrors[1:]
		return err
	}
	return nil
}

// MockClose simulates closing the file
func (m *MockFileCloser) MockClose() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseCalled = true
	return m.CloseError
}

// SetNextWriteError sets the next write to return an error
func (m *MockFileCloser) SetNextWriteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WriteErrors = append(m.WriteErrors, err)
}

// SetCloseError sets an error to return when Close is called
func (m *MockFileCloser) SetCloseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseError = err
}

// GetWriteCount returns the number of write calls
func (m *MockFileCloser) GetWriteCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.WriteCalls
}

// GetTotalBytesWritten returns the total bytes written
func (m *MockFileCloser) GetTotalBytesWritten() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := 0
	for _, data := range m.WriteData {
		total += len(data)
	}
	return total
}

// GetWrittenData returns all data that was written
func (m *MockFileCloser) GetWrittenData() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.WriteData))
	copy(result, m.WriteData)
	return result
}

// WasClosed returns true if Close was called
func (m *MockFileCloser) WasClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CloseCalled
}

// MockContextBuilder helps build test contexts with all required values
type MockContextBuilder struct {
	ctx context.Context
}

// NewMockContextBuilder creates a new context builder
func NewMockContextBuilder() *MockContextBuilder {
	baseCtx, cancel := context.WithCancel(context.Background())
	// Store cancel function in context for testing
	baseCtx = context.WithValue(baseCtx, "cancelCtx", cancel)

	return &MockContextBuilder{
		ctx: baseCtx,
	}
}

// WithRunNumber sets the run number in the context
func (b *MockContextBuilder) WithRunNumber(runNumber int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "runNumber", runNumber)
	return b
}

// WithGDCConfiguration sets the GDC configuration in the context
func (b *MockContextBuilder) WithGDCConfiguration(config *duck.GDCConfiguration) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "gdcConfiguration", config)
	return b
}

// WithExperiment sets the experiment name in the context
func (b *MockContextBuilder) WithExperiment(experiment string) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "experiment", experiment)
	return b
}

// WithMaxFilesize sets the maximum file size in the context
func (b *MockContextBuilder) WithMaxFilesize(size int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "maxFilesize", size)
	return b
}

// WithDecoderConfig sets the decoder configuration in the context
func (b *MockContextBuilder) WithDecoderConfig(config duck.DecoderConfiguration) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "decoderConfig", config)
	return b
}

// WithNGDCs sets the number of GDCs in the context
func (b *MockContextBuilder) WithNGDCs(n int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "nGDCs", n)
	return b
}

// WithWriteOutput sets the write output flag in the context
func (b *MockContextBuilder) WithWriteOutput(enabled bool) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "writeOutput", enabled)
	return b
}

// WithDecode sets the decode flag in the context
func (b *MockContextBuilder) WithDecode(enabled bool) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "decode", enabled)
	return b
}

// WithDecoderWorkers sets the number of decoder workers in the context
func (b *MockContextBuilder) WithDecoderWorkers(n int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "decoderWorkers", n)
	return b
}

// WithChannelSizes sets various channel buffer sizes in the context
func (b *MockContextBuilder) WithChannelSizes(metrics, tcpConn, writer, decoder int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "metricsChBufferSize", metrics)
	b.ctx = context.WithValue(b.ctx, "tcpConnChBufferSize", tcpConn)
	b.ctx = context.WithValue(b.ctx, "writerChBufferSize", writer)
	b.ctx = context.WithValue(b.ctx, "decoderChBufferSize", decoder)
	return b
}

// WithWriterChBufferSize sets the writer channel buffer size in the context
func (b *MockContextBuilder) WithWriterChBufferSize(size int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "writerChBufferSize", size)
	return b
}

// WithDecoderChBufferSize sets the decoder channel buffer size in the context
func (b *MockContextBuilder) WithDecoderChBufferSize(size int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "decoderChBufferSize", size)
	return b
}

// WithMetricsChBufferSize sets the metrics channel buffer size in the context
func (b *MockContextBuilder) WithMetricsChBufferSize(size int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "metricsChBufferSize", size)
	return b
}

// WithTimeout adds a timeout to the context
func (b *MockContextBuilder) WithTimeout(timeout time.Duration) (*MockContextBuilder, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(b.ctx, timeout)
	b.ctx = ctx
	return b, cancel
}

// Build returns the constructed context
func (b *MockContextBuilder) Build() context.Context {
	return b.ctx
}

// Cancel returns the cancel function for the context
func (b *MockContextBuilder) Cancel() context.CancelFunc {
	cancel, _ := b.ctx.Value("cancelCtx").(context.CancelFunc)
	return cancel
}

// CreateTestContext is a convenience function to create a fully populated test context
func CreateTestContext(gdcConfig *duck.GDCConfiguration, runNumber int) context.Context {
	return NewMockContextBuilder().
		WithRunNumber(runNumber).
		WithGDCConfiguration(gdcConfig).
		WithExperiment("next100").
		WithMaxFilesize(1000000000). // 1GB
		WithNGDCs(1).
		WithWriteOutput(true).
		WithDecode(false).
		WithDecoderWorkers(1).
		WithChannelSizes(100, 100, 100, 100).
		WithDecoderConfig(duck.DecoderConfiguration{
			WriteData:     false,
			SplitTrigger:  false,
			ExtTrigger:    0,
			TrgCode1:      0,
			TrgCode2:      1,
		}).
		Build()
}
