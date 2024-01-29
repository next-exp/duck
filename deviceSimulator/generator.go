package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

// Generator handles the generation of random test events
type Generator struct {
	config   GeneratorConfig
	sender   *UDPSender
	limiter  *RateLimiter
	state    SourceState
	stateMu  sync.RWMutex
	stopCh   chan struct{}
	doneCh   chan struct{}

	// Metrics
	eventsSent     uint64
	currentEventID uint32
	errorsInjected uint64
	startTime      time.Time
}

// NewGenerator creates a new generator with the given configuration
func NewGenerator(config GeneratorConfig) (*Generator, error) {
	// Validate configuration
	if config.PacketsPerEvent <= 0 {
		config.PacketsPerEvent = 10 // Default
	}
	if config.PacketSize <= 0 {
		config.PacketSize = 7992 // Default for NEXT-100
	}
	if config.RateHz <= 0 {
		config.RateHz = 1.0 // Default to 1 Hz
	}

	return &Generator{
		config: config,
		state:  SourceStateStopped,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}, nil
}

// Start begins the event generation
func (g *Generator) Start() error {
	g.stateMu.Lock()
	if g.state != SourceStateStopped {
		g.stateMu.Unlock()
		return fmt.Errorf("generator is already running")
	}
	g.state = SourceStateRunning
	g.stateMu.Unlock()

	// Create UDP sender
	sender, err := NewUDPSender(g.config.Equipment, g.config.PacketSize)
	if err != nil {
		return fmt.Errorf("failed to create UDP sender: %w", err)
	}
	g.sender = sender

	// Create rate limiter with absolute rate in Hz
	g.limiter = NewRateLimiter(g.config.RateHz)

	// Start generator in background
	go g.generateLoop()

	log.Printf("Started generator for equipment %d: rate=%.1f Hz, packets=%d, size=%d, error_rate=%.2f%%",
		g.config.EquipmentID, g.config.RateHz, g.config.PacketsPerEvent,
		g.config.PacketSize, g.config.ErrorInjectionRate*100)
	return nil
}

// Stop stops the event generation
func (g *Generator) Stop() error {
	g.stateMu.Lock()
	defer g.stateMu.Unlock()

	if g.state == SourceStateStopped {
		return fmt.Errorf("generator is not running")
	}

	close(g.stopCh)
	g.state = SourceStateStopped

	// Wait for generator to finish
	<-g.doneCh

	// Close resources
	if g.sender != nil {
		g.sender.Close()
	}

	log.Println("Generator stopped")
	return nil
}

// GetState returns the current generator state (implements DataSource interface)
func (g *Generator) GetState() SourceState {
	g.stateMu.RLock()
	defer g.stateMu.RUnlock()
	return g.state
}

// GetMetrics returns generator statistics (implements DataSource interface)
func (g *Generator) GetMetrics() SourceMetrics {
	senderMetrics := SenderMetrics{}
	if g.sender != nil {
		senderMetrics = g.sender.GetMetrics()
	}
	return SourceMetrics{
		EventsSent:     g.eventsSent,
		PacketsSent:    senderMetrics.PacketsSent,
		BytesSent:      senderMetrics.BytesSent,
		Errors:         senderMetrics.Errors,
		ErrorsInjected: g.errorsInjected,
	}
}

// GetStatus returns a human-readable status string (implements DataSource interface)
func (g *Generator) GetStatus() string {
	state := g.GetState()
	metrics := g.GetMetrics()

	elapsed := time.Since(g.startTime)
	eventsPerSec := float64(0)
	if elapsed.Seconds() > 0 {
		eventsPerSec = float64(metrics.EventsSent) / elapsed.Seconds()
	}

	return fmt.Sprintf(
		"State: %s | Events: %d | Current: %d | Rate: %.1f evt/s | Packets: %d | Bytes: %d | Errors: %d | Injected: %d",
		state.String(), metrics.EventsSent, g.currentEventID, eventsPerSec,
		metrics.PacketsSent, metrics.BytesSent, metrics.Errors, metrics.ErrorsInjected,
	)
}

// generateLoop is the main generation loop
func (g *Generator) generateLoop() {
	defer close(g.doneCh)

	g.startTime = time.Now()
	g.currentEventID = 1

	for {
		// Check for stop signal
		select {
		case <-g.stopCh:
			return
		default:
		}

		// Rate limiting
		g.limiter.Wait()

		// Determine if we should inject an error in this event
		injectError := rand.Float64() < g.config.ErrorInjectionRate

		// Send event
		err := g.sendEvent(int(g.currentEventID), injectError)
		if err != nil {
			log.Printf("Error sending event %d: %v", g.currentEventID, err)
			return
		}

		if injectError {
			g.errorsInjected++
		}

		g.eventsSent++
		g.currentEventID++

		// Log progress every 100 events
		if g.eventsSent%100 == 0 {
			log.Printf("Generated %d events (current: %d)", g.eventsSent, g.currentEventID-1)
		}

		// Check max events limit
		if g.config.MaxEvents > 0 && int(g.eventsSent) >= g.config.MaxEvents {
			log.Printf("Reached max events limit (%d)", g.config.MaxEvents)
			g.stateMu.Lock()
			g.state = SourceStateStopped
			g.stateMu.Unlock()
			return
		}
	}
}

// sendEvent generates and sends a single event with random data
func (g *Generator) sendEvent(eventID int, injectError bool) error {
	nPackets := g.config.PacketsPerEvent
	packetSize := g.config.PacketSize
	packetErrorIdx := rand.IntN(nPackets)

	for packetIdx := 0; packetIdx < nPackets; packetIdx++ {
		// Create packet with test data
		packet := make([]byte, packetSize)

		// Fill with random value (avoiding 0xfa which is the end marker)
		value := rand.IntN(254) + 1
		if value == 0xfa {
			value++
		}
		for i := 0; i < packetSize; i++ {
			packet[i] = byte(value)
		}

		// Add packet header: [seq counter BE 4 bytes][event ID LE 4 bytes]
		binary.BigEndian.PutUint32(packet[0:4], uint32(packetIdx))

		// Inject error by replacing sequence counter with 0xdeadbeef
		if injectError && packetIdx == packetErrorIdx {
			binary.BigEndian.PutUint32(packet[0:4], uint32(0xdeadbeef))
		}

		// Event ID in little endian
		binary.LittleEndian.PutUint32(packet[4:8], uint32(eventID))

		// Send packet
		if err := g.sender.SendRawPacket(packet); err != nil {
			return fmt.Errorf("failed to send packet %d/%d: %w", packetIdx+1, nPackets, err)
		}
	}

	// Send end-of-event marker
	if err := g.sender.SendEndMarker(); err != nil {
		return fmt.Errorf("failed to send end marker: %w", err)
	}

	return nil
}
