package testhelpers

import (
	"context"
	"net"
	"sync"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// MockTCPConn implements a mock TCP connection for testing
type MockTCPConn struct {
	mu         sync.Mutex
	written    [][]byte
	closed     bool
	WriteError error // Return this error on Write()
}

// NewMockTCPConn creates a new mock TCP connection
func NewMockTCPConn() *MockTCPConn {
	return &MockTCPConn{
		written: make([][]byte, 0),
		closed:  false,
	}
}

// Write records data written to the connection
func (m *MockTCPConn) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.WriteError != nil {
		return 0, m.WriteError
	}

	data := make([]byte, len(b))
	copy(data, b)
	m.written = append(m.written, data)
	return len(b), nil
}

// SetWriteError sets the error to return on Write calls
func (m *MockTCPConn) SetWriteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WriteError = err
}

// Read is not implemented for this mock
func (m *MockTCPConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

// Close marks the connection as closed
func (m *MockTCPConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// LocalAddr returns a dummy address
func (m *MockTCPConn) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	return addr
}

// RemoteAddr returns a dummy address
func (m *MockTCPConn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	return addr
}

// SetDeadline does nothing in the mock
func (m *MockTCPConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline does nothing in the mock
func (m *MockTCPConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline does nothing in the mock
func (m *MockTCPConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// GetWritten returns all data written to the connection
func (m *MockTCPConn) GetWritten() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.written
}

// IsClosed returns whether the connection was closed
func (m *MockTCPConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// MockUDPConn implements a mock UDP connection for network I/O testing
type MockUDPConn struct {
	mu            sync.Mutex
	readData      [][]byte         // Packets to return on ReadFromUDP
	readAddrs     []*net.UDPAddr   // Source addresses for each packet
	readIndex     int
	readError     error            // Optional persistent error to return
	readErrorsAt  map[int]error    // Errors to return at specific indices
	closed        bool
	deadline      time.Time        // Connection deadline
	readDeadline  time.Time        // Read deadline
	writeDeadline time.Time        // Write deadline
	blocking      bool             // If true, block when no more data (default: true)
}

// NewMockUDPConn creates a new mock UDP connection with specified packets
func NewMockUDPConn(packets [][]byte, sourceIP string) *MockUDPConn {
	addrs := make([]*net.UDPAddr, len(packets))
	for i := range packets {
		addrs[i], _ = net.ResolveUDPAddr("udp", sourceIP+":12345")
	}
	return &MockUDPConn{
		readData:     packets,
		readAddrs:    addrs,
		readIndex:    0,
		readErrorsAt: make(map[int]error),
		closed:       false,
		blocking:     true,
	}
}

// NewMockUDPConnWithErrors creates a mock UDP connection that returns specific errors at given indices
func NewMockUDPConnWithErrors(packets [][]byte, sourceIP string, errorsAt map[int]error) *MockUDPConn {
	conn := NewMockUDPConn(packets, sourceIP)
	conn.readErrorsAt = errorsAt
	return conn
}

// NewMockUDPConnNonBlocking creates a mock UDP connection that returns timeout instead of blocking
func NewMockUDPConnNonBlocking(packets [][]byte, sourceIP string) *MockUDPConn {
	conn := NewMockUDPConn(packets, sourceIP)
	conn.blocking = false
	return conn
}

// NewMockUDPConnWithMixedAddrs creates a mock UDP connection with different source addresses per packet
func NewMockUDPConnWithMixedAddrs(packets [][]byte, sourceIPs []string) *MockUDPConn {
	addrs := make([]*net.UDPAddr, len(packets))
	for i := range packets {
		if i < len(sourceIPs) {
			addrs[i], _ = net.ResolveUDPAddr("udp", sourceIPs[i]+":12345")
		} else {
			addrs[i], _ = net.ResolveUDPAddr("udp", "127.0.0.1:12345")
		}
	}
	return &MockUDPConn{
		readData:     packets,
		readAddrs:    addrs,
		readIndex:    0,
		readErrorsAt: make(map[int]error),
		closed:       false,
		blocking:     true,
	}
}

// ReadFromUDP simulates reading from UDP connection
func (m *MockUDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for persistent error first
	if m.readError != nil {
		return 0, nil, m.readError
	}

	// Check for index-specific error
	if err, exists := m.readErrorsAt[m.readIndex]; exists {
		m.readIndex++
		return 0, nil, err
	}

	if m.readIndex >= len(m.readData) {
		if m.blocking {
			// Block indefinitely (simulates waiting for data)
			m.mu.Unlock()
			select {}
		}
		// Non-blocking mode: return timeout error
		return 0, nil, &timeoutError{message: "i/o timeout"}
	}

	data := m.readData[m.readIndex]
	addr := m.readAddrs[m.readIndex]
	m.readIndex++

	n := copy(b, data)
	return n, addr, nil
}

// SetReadError sets a persistent error to return on all ReadFromUDP calls
func (m *MockUDPConn) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readError = err
}

// SetReadErrorAt sets an error to return at a specific read index
func (m *MockUDPConn) SetReadErrorAt(index int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readErrorsAt[index] = err
}

// ClearReadErrors clears all read errors
func (m *MockUDPConn) ClearReadErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readError = nil
	m.readErrorsAt = make(map[int]error)
}

// GetReadIndex returns the current read position
func (m *MockUDPConn) GetReadIndex() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readIndex
}

// ResetReadIndex resets the read position to the beginning
func (m *MockUDPConn) ResetReadIndex() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readIndex = 0
}

// Close marks the connection as closed
func (m *MockUDPConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// IsClosed returns whether the connection was closed
func (m *MockUDPConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// SetDeadline sets both read and write deadlines
func (m *MockUDPConn) SetDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deadline = t
	m.readDeadline = t
	m.writeDeadline = t
	return nil
}

// SetReadDeadline sets the read deadline
func (m *MockUDPConn) SetReadDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDeadline = t
	return nil
}

// SetWriteDeadline sets the write deadline
func (m *MockUDPConn) SetWriteDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeDeadline = t
	return nil
}

// LocalAddr returns a mock local address
func (m *MockUDPConn) LocalAddr() net.Addr {
	return &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 6006,
	}
}

// RemoteAddr returns a mock remote address
func (m *MockUDPConn) RemoteAddr() net.Addr {
	return &net.UDPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 12345,
	}
}

// timeoutError implements net.Error for timeout simulation
type timeoutError struct {
	message string
}

func (e *timeoutError) Error() string   { return e.message }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// MockContextBuilder helps build context.Context with common test values
type MockContextBuilder struct {
	ctx context.Context
}

// NewMockContextBuilder creates a new context builder with a background context
func NewMockContextBuilder() *MockContextBuilder {
	return &MockContextBuilder{
		ctx: context.Background(),
	}
}

// WithRunNumber adds run number to context
func (b *MockContextBuilder) WithRunNumber(n int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "runNumber", n)
	return b
}

// WithLDCConfiguration adds LDC configuration to context
func (b *MockContextBuilder) WithLDCConfiguration(c *duck.LDCConfiguration) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "ldcConfiguration", c)
	return b
}

// WithListenerCounter adds listener counter channel to context
func (b *MockContextBuilder) WithListenerCounter(ch chan string) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "listenerCounter", ch)
	return b
}

// WithListenerClosedCounter adds listener closed counter channel to context
func (b *MockContextBuilder) WithListenerClosedCounter(ch chan string) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "listenerClosedCounter", ch)
	return b
}

// WithNEnabledEquipments adds number of enabled equipments to context
func (b *MockContextBuilder) WithNEnabledEquipments(n int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "nEnabledEquipments", n)
	return b
}

// WithMetricsChBufferSize adds metrics channel buffer size to context
func (b *MockContextBuilder) WithMetricsChBufferSize(size int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "metricsChBufferSize", size)
	return b
}

// Build returns the built context
func (b *MockContextBuilder) Build() context.Context {
	return b.ctx
}

// WithCancel wraps the context with cancellation and returns the cancel function
func (b *MockContextBuilder) WithCancel() (*MockContextBuilder, context.CancelFunc) {
	ctx, cancel := context.WithCancel(b.ctx)
	b.ctx = ctx
	return b, cancel
}

// WithPacketBufferSize adds packet buffer size to context
func (b *MockContextBuilder) WithPacketBufferSize(size int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "packetBufferSize", size)
	return b
}

// WithNPacketsInBuffer adds number of packets in buffer to context
func (b *MockContextBuilder) WithNPacketsInBuffer(n int) *MockContextBuilder {
	b.ctx = context.WithValue(b.ctx, "nPacketsInBuffer", n)
	return b
}

// MockPinger implements a mock pinger for testing without network access
type MockPinger struct {
	mu            sync.Mutex
	PingResults   map[string]bool // IP -> success
	PingErrors    map[string]error // IP -> error on NewPinger
	RunErrors     map[string]error // IP -> error on Run
	PacketsRecv   map[string]int   // IP -> packets received count
	DefaultRecv   int              // Default packets received if not specified
}

// NewMockPinger creates a new mock pinger with all pings succeeding by default
func NewMockPinger() *MockPinger {
	return &MockPinger{
		PingResults: make(map[string]bool),
		PingErrors:  make(map[string]error),
		RunErrors:   make(map[string]error),
		PacketsRecv: make(map[string]int),
		DefaultRecv: 2, // Default success
	}
}

// SetPingSuccess sets whether ping to an IP should succeed
func (m *MockPinger) SetPingSuccess(ip string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PingResults[ip] = success
	if success {
		m.PacketsRecv[ip] = 2
	} else {
		m.PacketsRecv[ip] = 0
	}
}

// SetPingError sets an error to return when creating a pinger for an IP
func (m *MockPinger) SetPingError(ip string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PingErrors[ip] = err
}

// SetRunError sets an error to return when running ping for an IP
func (m *MockPinger) SetRunError(ip string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunErrors[ip] = err
}

// SetPacketsRecv sets the number of packets received for an IP
func (m *MockPinger) SetPacketsRecv(ip string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PacketsRecv[ip] = count
}

// GetPingError returns the error for creating a pinger for an IP
func (m *MockPinger) GetPingError(ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.PingErrors[ip]
}

// GetRunError returns the error for running ping for an IP
func (m *MockPinger) GetRunError(ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.RunErrors[ip]
}

// GetPacketsRecv returns the packets received count for an IP
func (m *MockPinger) GetPacketsRecv(ip string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if count, ok := m.PacketsRecv[ip]; ok {
		return count
	}
	return m.DefaultRecv
}

// MockPingerFactory creates mock pingers for testing
type MockPingerFactory struct {
	mock     *MockPinger
	adapters []*MockPingerAdapter // Track created adapters
}

// NewMockPingerFactory creates a new mock pinger factory
func NewMockPingerFactory(mock *MockPinger) *MockPingerFactory {
	return &MockPingerFactory{mock: mock, adapters: make([]*MockPingerAdapter, 0)}
}

// GetLastCreatedAdapter returns the most recently created adapter
func (f *MockPingerFactory) GetLastCreatedAdapter() *MockPingerAdapter {
	if len(f.adapters) == 0 {
		return nil
	}
	return f.adapters[len(f.adapters)-1]
}

// NewPinger creates a new mock pinger for the given IP address
func (f *MockPingerFactory) NewPinger(ip string) (any, error) {
	if err := f.mock.GetPingError(ip); err != nil {
		return nil, err
	}
	adapter := &MockPingerAdapter{mock: f.mock, ip: ip}
	f.adapters = append(f.adapters, adapter) // Track adapter
	return adapter, nil
}

// MockPingerAdapter adapts MockPinger to implement the Pinger interface
type MockPingerAdapter struct {
	mock    *MockPinger
	ip      string
	count   int           // Track SetCount calls
	timeout time.Duration // Track SetTimeout calls
}

// SetCount tracks the count value for verification
func (a *MockPingerAdapter) SetCount(count int) {
	a.count = count
}

// SetTimeout tracks the timeout value for verification
func (a *MockPingerAdapter) SetTimeout(timeout time.Duration) {
	a.timeout = timeout
}

// Run simulates running the ping and returns any configured error
func (a *MockPingerAdapter) Run() error {
	return a.mock.GetRunError(a.ip)
}

// PacketsReceived returns the configured number of packets received
func (a *MockPingerAdapter) PacketsReceived() int {
	return a.mock.GetPacketsRecv(a.ip)
}

// GetCount returns the count value set by SetCount
func (a *MockPingerAdapter) GetCount() int {
	return a.count
}

// GetTimeout returns the timeout value set by SetTimeout
func (a *MockPingerAdapter) GetTimeout() time.Duration {
	return a.timeout
}
