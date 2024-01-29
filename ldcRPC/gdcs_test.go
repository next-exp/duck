package main

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/ldcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

func TestGdcConnectionPool_EmptyList_ReturnsEmptyConnections(t *testing.T) {
	testhelpers.SetupTest()
	// Setup
	gdcs := []duck.GDCConfiguration{}

	// Act
	connections, err := gdcGDCConnetionPool(gdcs)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, connections)
}

func TestGdcConnectionPool_AllDisabled_ReturnsEmpty(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - create disabled GDCs
	gdcs := testhelpers.NewTestGDCConfigurationList(3, false)

	// Act
	connections, err := gdcGDCConnetionPool(gdcs)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, connections)
}

func TestGdcConnectionPool_ValidConnection_ReturnsConnection(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - start a local TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	// Get the dynamically assigned port
	port := listener.Addr().(*net.TCPAddr).Port

	// Accept connections in background
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
	}()

	// Create GDC config pointing to our listener
	gdcs := []duck.GDCConfiguration{
		testhelpers.NewTestGDCConfigurationWithPort(1, "testgdc", port, true),
	}

	// Act
	connections, err := gdcGDCConnetionPool(gdcs)

	// Assert
	require.NoError(t, err)
	require.Len(t, connections, 1)

	// Cleanup
	for _, conn := range connections {
		conn.Close()
	}
}

func TestGdcConnectionPool_ConnectionFailure_ReturnsErrorWithHost(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - use an invalid IP/port that won't connect
	gdcs := []duck.GDCConfiguration{
		testhelpers.NewTestGDCConfigurationWithPort(1, "failgdc", 1, true), // Port 1 typically refuses connections
	}

	// Act
	connections, err := gdcGDCConnetionPool(gdcs)

	// Assert
	require.Error(t, err)
	assert.Nil(t, connections)
	assert.Contains(t, err.Error(), "testhost") // Error should contain host name (from fixture)
}

func TestGdcConnectionPool_MixedEnabledDisabled_OnlyConnectsEnabled(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - start local TCP listeners
	listener1, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener1.Close()

	listener2, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener2.Close()

	port1 := listener1.Addr().(*net.TCPAddr).Port
	port2 := listener2.Addr().(*net.TCPAddr).Port

	// Accept connections in background
	go func() {
		conn, _ := listener1.Accept()
		if conn != nil {
			defer conn.Close()
		}
	}()
	go func() {
		conn, _ := listener2.Accept()
		if conn != nil {
			defer conn.Close()
		}
	}()

	// Create mixed GDC configs - 2 enabled, 1 disabled
	gdcs := []duck.GDCConfiguration{
		testhelpers.NewTestGDCConfigurationWithPort(1, "gdc1", port1, true),
		testhelpers.NewTestGDCConfigurationWithPort(2, "gdc2", 99999, false), // Disabled, won't connect
		testhelpers.NewTestGDCConfigurationWithPort(3, "gdc3", port2, true),
	}

	// Act
	connections, err := gdcGDCConnetionPool(gdcs)

	// Assert
	require.NoError(t, err)
	assert.Len(t, connections, 2) // Only 2 enabled GDCs should connect

	// Cleanup
	for _, conn := range connections {
		conn.Close()
	}
}

func TestGdcConnection_ValidAddress_ReturnsConnection(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - start a local TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Accept connections in background
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			defer conn.Close()
		}
	}()

	gdcConfig := duck.GDCConfiguration{
		IP:   "127.0.0.1",
		Port: port,
	}

	// Act
	conn, err := gdcConnection(gdcConfig)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Verify connection can be used (it's a *net.TCPConn)
	assert.NotNil(t, conn.LocalAddr())
	assert.NotNil(t, conn.RemoteAddr())

	// Cleanup
	conn.Close()
}

func TestGdcConnectionPool_PartialFailure_ClosesEstablishedConnections(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - start a valid listener (first connection will succeed)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	validPort := listener.Addr().(*net.TCPAddr).Port

	// Track accepted connections to verify they're closed
	acceptedConns := make(chan net.Conn, 2)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			acceptedConns <- conn
		}
	}()

	// Setup config: Valid -> Fail (port 1 will refuse)
	gdcs := []duck.GDCConfiguration{
		testhelpers.NewTestGDCConfigurationWithPort(1, "valid", validPort, true),
		testhelpers.NewTestGDCConfigurationWithPort(2, "fail", 1, true),
	}

	// Act
	connections, err := gdcGDCConnetionPool(gdcs)

	// Assert - should fail on second connection
	require.Error(t, err)
	assert.Nil(t, connections)
	assert.Contains(t, err.Error(), "connection refused")

	// Verify the valid connection was closed
	select {
	case conn := <-acceptedConns:
		// Connection was accepted, now verify it gets closed
		defer conn.Close()

		// Set a read deadline to prevent hanging
		conn.SetReadDeadline(time.Now().Add(time.Second))

		// Read should return EOF when the other side closes the connection
		buf := make([]byte, 1)
		_, err := conn.Read(buf)
		assert.Equal(t, io.EOF, err, "Expected connection to be closed by pool (EOF), but got %v", err)
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for connection to be accepted")
	}
}

func TestGdcConnection_DNSError_ReturnsDNSErrorType(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - use an invalid IP address that will fail DNS resolution
	gdcConfig := duck.GDCConfiguration{
		Host: "test-gdc-dns",
		IP:   "invalid-ip-address-that-will-not-resolve",
		Port: 9999,
	}

	// Act
	conn, err := gdcConnection(gdcConfig)

	// Assert
	require.Error(t, err)
	assert.Nil(t, conn)

	// Verify it's a DNSError
	dnsErr, ok := err.(*DNSError)
	require.True(t, ok, "Error should be of type DNSError")
	assert.Equal(t, "test-gdc-dns", dnsErr.Host)
	assert.Equal(t, "invalid-ip-address-that-will-not-resolve", dnsErr.IP)
	assert.Equal(t, "dns", dnsErr.Type())
	assert.NotNil(t, dnsErr.Unwrap())
}

func TestGdcConnection_ConnectionRefused_ReturnsRefusedErrorType(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - use port 1 which typically refuses connections
	gdcConfig := duck.GDCConfiguration{
		Host: "test-gdc-refused",
		IP:   "127.0.0.1",
		Port: 1, // Port 1 typically refuses connections
	}

	// Act
	conn, err := gdcConnection(gdcConfig)

	// Assert
	require.Error(t, err)
	assert.Nil(t, conn)

	// Verify it's a ConnectionRefusedError
	refusedErr, ok := err.(*ConnectionRefusedError)
	require.True(t, ok, "Error should be of type ConnectionRefusedError")
	assert.Equal(t, "test-gdc-refused", refusedErr.Host)
	assert.Equal(t, "127.0.0.1", refusedErr.IP)
	assert.Equal(t, 1, refusedErr.Port)
	assert.Equal(t, "refused", refusedErr.Type())
	assert.NotNil(t, refusedErr.Unwrap())
}

func TestGdcConnection_Timeout_ReturnsTimeoutErrorType(t *testing.T) {
	testhelpers.SetupTest()
	// Setup - use a non-routable IP that will cause a timeout
	// 192.0.2.1 is in TEST-NET-1 (RFC 5737) and should never route
	gdcConfig := duck.GDCConfiguration{
		Host: "test-gdc-timeout",
		IP:   "192.0.2.1",
		Port: 9999,
	}

	// Act with a deadline to prevent test from hanging forever
	done := make(chan struct {
		conn *net.TCPConn
		err  error
	}, 1)
	go func() {
		conn, err := gdcConnection(gdcConfig)
		done <- struct {
			conn *net.TCPConn
			err  error
		}{conn, err}
	}()

	// Assert - should get an error within reasonable time
	select {
	case result := <-done:
		require.Error(t, result.err)
		assert.Nil(t, result.conn)

		// Verify it's a TimeoutError
		timeoutErr, ok := result.err.(*TimeoutError)
		require.True(t, ok, "Error should be of type TimeoutError")
		assert.Equal(t, "test-gdc-timeout", timeoutErr.Host)
		assert.Equal(t, "192.0.2.1", timeoutErr.IP)
		assert.Equal(t, 9999, timeoutErr.Port)
		assert.Equal(t, "timeout", timeoutErr.Type())
		assert.NotNil(t, timeoutErr.Unwrap())
	case <-time.After(10 * time.Second):
		t.Fatal("Connection did not timeout within expected time")
	}
}
