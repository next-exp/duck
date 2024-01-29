package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// UDPConn defines the interface for UDP connection operations
// This allows for easier testing with mock connections
type UDPConn interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	SetDeadline(t time.Time) error
	Close() error
}

func openEquipmentSocket(s *server, equipment duck.Equipment) (*net.UDPConn, error) {
	listenAddr := fmt.Sprintf("%s:%d", equipment.HostIP, equipment.HostPort)

	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			if err := c.Control(func(fd uintptr) {
				// Allow quick rebind after StopRun -> StartRun
				_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			}); err != nil {
				return err
			}
			return nil
		},
	}

	pc, err := lc.ListenPacket(context.Background(), "udp", listenAddr)
	if err != nil {
		return nil, err
	}
	udpConn, ok := pc.(*net.UDPConn)
	if !ok {
		pc.Close()
		return nil, fmt.Errorf("expected UDPConn")
	}

	message := fmt.Sprintf("Listening to equipment %s:%d (ID:%d)", equipment.HostIP, equipment.HostPort, equipment.ID)
	s.logger.Slog.Info(message)
	return udpConn, nil
}

func readUDPSocket(s *server, conn UDPConn, buffer *[]byte, position int, deviceIP *net.IP,
	recvChannel *chan RingBufferData, bufferTracking *RingBufferTracking, packetBufferSize int, nPacketsInBuffer int, bufferTimeout time.Duration) (int, int) {
	// Add timeout to avoid blocking indefinetely
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Calculate the buffer index from the byte position
	bufferIndex := position / packetBufferSize

	// Block until the position is available (backpressure)
	// Use a timeout to avoid blocking forever if consumer is stuck
	err := bufferTracking.TryAcquirePosition(bufferIndex, bufferTimeout)
	if err != nil {
		message := fmt.Sprintf("Buffer full, timeout waiting for position %d: %v", bufferIndex, err)
		s.logger.Slog.Error(message)
		s.metrics.networkBufferErrorCounter.Inc()
		// Return 0 to retry with the same position
		return 0, 0
	}

	size, addr, err := conn.ReadFromUDP((*buffer)[position : position+packetBufferSize])
	if err != nil {
		if !strings.Contains(err.Error(), "i/o timeout") {
			message := fmt.Errorf("error reading UDP socket: %w", err)
			s.logger.Slog.Error(message.Error())
		}
		// Release the position since we didn't write valid data
		bufferTracking.ReleasePosition(bufferIndex)
		return 0, 0
	}

	if addr.IP.Equal(*deviceIP) {
		// Mark position as in use after writing
		bufferTracking.MarkPositionInUse(bufferIndex)
		data := RingBufferData{data: (*buffer)[position : position+size], position: bufferIndex}
		*recvChannel <- data
		position += packetBufferSize
		if position >= packetBufferSize*nPacketsInBuffer {
			position = 0
		}
		return position, size
	}
	// if data is not from the device, release the position and return the same position
	bufferTracking.ReleasePosition(bufferIndex)
	return 0, 0
}

func listenEquipment(s *server, equipment duck.Equipment, conn UDPConn, recvChannel chan RingBufferData,
	bufferTracking *RingBufferTracking) {
	deviceIP := net.ParseIP(equipment.DeviceIP)

	// Read buffer configuration from context
	packetBufferSize, _ := s.getContext().Value("packetBufferSize").(int)
	nPacketsInBuffer, _ := s.getContext().Value("nPacketsInBuffer").(int)
	bufferTimeout, _ := s.getContext().Value("bufferTimeout").(int)

	// Use default of 30 seconds if not configured (0 or not set)
	if bufferTimeout <= 0 {
		bufferTimeout = 30
	}

	timeoutDuration := time.Duration(bufferTimeout) * time.Second
	buffer := make([]byte, packetBufferSize*nPacketsInBuffer)
	position := 0
	totalSize := 0
	size := 0

	for {
		select {
		case <-s.getContext().Done():
			message := fmt.Sprintf("Close equipment socket %s", equipment.DeviceIP)
			s.logger.Slog.Info(message)
			err := conn.Close()
			if err != nil {
				message := fmt.Sprintf("Error closing equipment socket %s", err.Error())
				s.logger.Slog.Error(message)
			}
			return
		default:
			position, size = readUDPSocket(s, conn, &buffer, position, &deviceIP, &recvChannel, bufferTracking, packetBufferSize, nPacketsInBuffer, timeoutDuration)
			totalSize += size
		}
	}
}

func readoutEquipment(s *server, equipment duck.Equipment, recvChannel chan RingBufferData, bufferTracking *RingBufferTracking, readyChan chan struct{}) {
	conn, err := openEquipmentSocket(s, equipment)
	if err != nil {
		s.stopOnLocalError(err.Error())
		return
	}

	// Signal that socket is ready
	if readyChan != nil {
		readyChan <- struct{}{}
	}

	listenEquipment(s, equipment, conn, recvChannel, bufferTracking)
}
