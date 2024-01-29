package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

const (
	// defaultGdcTimeout is the default timeout for GDC connections
	defaultGdcTimeout = 5 * time.Second
)

// GDCConnectionError is the base interface for all GDC connection errors
type GDCConnectionError interface {
	error
	GDC() string  // Return the GDC host that failed
	Type() string // Return error type: "dns", "refused", "timeout"
}

// DNSError represents a DNS resolution failure
type DNSError struct {
	Host string
	IP   string
	Err  error
}

func (e *DNSError) Error() string {
	return fmt.Sprintf("DNS error for GDC %s (IP: %s): %s", e.Host, e.IP, e.Err.Error())
}

func (e *DNSError) GDC() string {
	return e.Host
}

func (e *DNSError) Type() string {
	return "dns"
}

func (e *DNSError) Unwrap() error {
	return e.Err
}

// ConnectionRefusedError represents a connection refused (port not listening)
type ConnectionRefusedError struct {
	Host string
	IP   string
	Port int
	Err  error
}

func (e *ConnectionRefusedError) Error() string {
	return fmt.Sprintf("connection refused for GDC %s (%s:%d): %s", e.Host, e.IP, e.Port, e.Err.Error())
}

func (e *ConnectionRefusedError) GDC() string {
	return e.Host
}

func (e *ConnectionRefusedError) Type() string {
	return "refused"
}

func (e *ConnectionRefusedError) Unwrap() error {
	return e.Err
}

// TimeoutError represents a connection timeout
type TimeoutError struct {
	Host string
	IP   string
	Port int
	Err  error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout connecting to GDC %s (%s:%d): %s", e.Host, e.IP, e.Port, e.Err.Error())
}

func (e *TimeoutError) GDC() string {
	return e.Host
}

func (e *TimeoutError) Type() string {
	return "timeout"
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

func gdcGDCConnetionPool(gdcs []duck.GDCConfiguration) ([]net.Conn, error) {
	nGDCs := len(gdcs)
	gdcConnections := make([]net.Conn, 0)
	for i := 0; i < nGDCs; i++ {
		gdc := gdcs[i]
		if gdc.Enabled {
			gdcConn, err := gdcConnection(gdc)
			if err != nil {
				// Close any established connections before returning
				for _, conn := range gdcConnections {
					conn.Close()
				}
				// Return the typed error directly - it already contains GDC context
				return nil, err
			}
			gdcConnections = append(gdcConnections, gdcConn)
		}
	}
	return gdcConnections, nil
}

func gdcConnection(gdcConfiguration duck.GDCConfiguration) (*net.TCPConn, error) {
	// Resolve the string address to a TCP address
	address := fmt.Sprintf("%s:%d", gdcConfiguration.IP, gdcConfiguration.Port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	if err != nil {
		// DNS resolution failure
		return nil, &DNSError{
			Host: gdcConfiguration.Host,
			IP:   gdcConfiguration.IP,
			Err:  err,
		}
	}

	// Connect to the address with tcp using a timeout
	dialer := &net.Dialer{
		Timeout: defaultGdcTimeout,
	}
	conn, err := dialer.Dial("tcp", tcpAddr.String())
	if err != nil {
		// Determine error type and return appropriate typed error
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				return nil, &TimeoutError{
					Host: gdcConfiguration.Host,
					IP:   gdcConfiguration.IP,
					Port: gdcConfiguration.Port,
					Err:  err,
				}
			}
		}

		// Check for connection refused
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "connect: connection refused") {
			return nil, &ConnectionRefusedError{
				Host: gdcConfiguration.Host,
				IP:   gdcConfiguration.IP,
				Port: gdcConfiguration.Port,
				Err:  err,
			}
		}

		// Other errors - wrap with context
		return nil, fmt.Errorf("error connecting to GDC %s (%s:%d): %w", gdcConfiguration.Host, gdcConfiguration.IP, gdcConfiguration.Port, err)
	}

	return conn.(*net.TCPConn), nil
}
