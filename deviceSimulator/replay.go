package main

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// ReplayState is an alias for SourceState for backward compatibility
type ReplayState = SourceState

// State constants for backward compatibility
const (
	StateStopped = SourceStateStopped
	StateRunning = SourceStateRunning
	StatePaused  = SourceStatePaused
)

// ReplayConfig holds configuration for the replay
type ReplayConfig struct {
	FilePath    string
	EquipmentID uint32
	Equipment   duck.Equipment
	RateHz      float64 // Absolute event rate in Hz (events per second). 0 or negative = no rate limiting
	LoopMode    bool
	MaxEvents   int
	PacketSize  int
}

// Replayer handles the replay of events from a file
type Replayer struct {
	config   ReplayConfig
	parser   *FileParser
	sender   *UDPSender
	limiter  *RateLimiter
	state    ReplayState
	stateMu  sync.RWMutex
	stopCh   chan struct{}
	pauseCh  chan struct{}
	resumeCh chan struct{}
	doneCh   chan struct{}

	// Metrics
	eventsReplayed uint64
	currentEventID uint32
	startTime      time.Time
}

// NewReplayer creates a new replayer with the given configuration
func NewReplayer(config ReplayConfig) (*Replayer, error) {
	// Validate configuration
	if config.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	if config.PacketSize == 0 {
		config.PacketSize = DefaultPacketSize
	}
	// Note: RateHz can be 0 or negative to disable rate limiting

	return &Replayer{
		config:   config,
		state:    StateStopped,
		stopCh:   make(chan struct{}),
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
		doneCh:   make(chan struct{}),
	}, nil
}

// Start begins the replay
func (r *Replayer) Start() error {
	r.stateMu.Lock()
	if r.state != StateStopped {
		r.stateMu.Unlock()
		return fmt.Errorf("replay is already running")
	}
	r.state = StateRunning
	r.stateMu.Unlock()

	// Open and parse file
	parser, err := NewFileParser(r.config.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	r.parser = parser

	// Create UDP sender
	sender, err := NewUDPSender(r.config.Equipment, r.config.PacketSize)
	if err != nil {
		parser.Close()
		return fmt.Errorf("failed to create UDP sender: %w", err)
	}
	r.sender = sender

	// Create rate limiter with absolute rate in Hz
	r.limiter = NewRateLimiter(r.config.RateHz)

	// Start replay in background
	go r.replayLoop()

	log.Printf("Started replay of file %s for equipment %d", r.config.FilePath, r.config.EquipmentID)
	return nil
}

// Stop stops the replay
func (r *Replayer) Stop() error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	if r.state == StateStopped {
		return fmt.Errorf("replay is not running")
	}

	close(r.stopCh)
	r.state = StateStopped

	// Wait for replay to finish
	<-r.doneCh

	// Close resources
	if r.sender != nil {
		r.sender.Close()
	}
	if r.parser != nil {
		r.parser.Close()
	}

	log.Println("Replay stopped")
	return nil
}

// Pause pauses the replay
func (r *Replayer) Pause() error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	if r.state != StateRunning {
		return fmt.Errorf("replay is not running")
	}

	r.state = StatePaused
	close(r.pauseCh)
	log.Println("Replay paused")
	return nil
}

// Resume resumes a paused replay
func (r *Replayer) Resume() error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	if r.state != StatePaused {
		return fmt.Errorf("replay is not paused")
	}

	r.state = StateRunning
	close(r.resumeCh)
	log.Println("Replay resumed")
	return nil
}

// GetState returns the current replay state
func (r *Replayer) GetState() ReplayState {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	return r.state
}

// GetMetrics returns replay statistics as SourceMetrics (implements DataSource interface)
func (r *Replayer) GetMetrics() SourceMetrics {
	senderMetrics := SenderMetrics{}
	if r.sender != nil {
		senderMetrics = r.sender.GetMetrics()
	}
	return SourceMetrics{
		EventsSent:     r.eventsReplayed,
		PacketsSent:    senderMetrics.PacketsSent,
		BytesSent:      senderMetrics.BytesSent,
		Errors:         senderMetrics.Errors,
		ErrorsInjected: 0, // Replay mode doesn't inject errors
	}
}

// GetDetailedMetrics returns detailed replay statistics for backward compatibility
func (r *Replayer) GetDetailedMetrics() (uint64, uint32, SenderMetrics) {
	senderMetrics := SenderMetrics{}
	if r.sender != nil {
		senderMetrics = r.sender.GetMetrics()
	}
	return r.eventsReplayed, r.currentEventID, senderMetrics
}

// replayLoop is the main replay loop
func (r *Replayer) replayLoop() {
	defer close(r.doneCh)

	r.startTime = time.Now()

	for {
		// Stream events one at a time instead of loading all into memory
		eventsInFile := uint64(0)

		for {
			// Check for stop signal
			select {
			case <-r.stopCh:
				return
			case <-r.pauseCh:
				// Wait for resume or stop
				select {
				case <-r.resumeCh:
					// Reset channels for next pause
					r.pauseCh = make(chan struct{})
					r.resumeCh = make(chan struct{})
				case <-r.stopCh:
					return
				}
			default:
			}

			// Parse next event from file (streaming, no bulk load)
			event, err := r.parser.parseNextEvent()
			if err == io.EOF {
				// Reached end of file
				break
			}
			if err != nil {
				log.Printf("Error parsing event: %v", err)
				return
			}

			eventsInFile++

			// Check if we should send this event (based on equipment ID)
			equipmentData, ok := event.Equipments[r.config.EquipmentID]
			if !ok {
				// No data for this equipment in this event, skip
				// This event is discarded immediately, no memory retained
				continue
			}

			// Rate limiting using absolute rate in Hz (ignores file timestamps)
			r.limiter.Wait()

			// Send event
			err = r.sender.SendEvent(event.EventID, equipmentData)
			if err != nil {
				log.Printf("Error sending event %d: %v", event.EventID, err)
				// Continue with next event
			} else {
				r.eventsReplayed++
				r.currentEventID = event.EventID
			}

			// Check max events limit
			if r.config.MaxEvents > 0 && int(r.eventsReplayed) >= r.config.MaxEvents {
				log.Printf("Reached max events limit (%d)", r.config.MaxEvents)
				return
			}
		}

		if eventsInFile == 0 {
			log.Println("No events found in file")
			return
		}

		log.Printf("Streamed %d events from file (sent %d for equipment %d)",
			eventsInFile, r.eventsReplayed, r.config.EquipmentID)

		// Check loop mode
		if !r.config.LoopMode {
			log.Println("Replay completed (loop mode disabled)")
			return
		}

		log.Println("Looping back to start of file")

		// Reset parser to beginning of file
		if err := r.parser.Reset(); err != nil {
			log.Printf("Error resetting parser: %v", err)
			return
		}
	}
}

// GetStatus returns a human-readable status string (implements DataSource interface)
func (r *Replayer) GetStatus() string {
	state := r.GetState()
	metrics := r.GetMetrics()

	elapsed := time.Since(r.startTime)
	eventsPerSec := float64(0)
	if elapsed.Seconds() > 0 {
		eventsPerSec = float64(metrics.EventsSent) / elapsed.Seconds()
	}

	return fmt.Sprintf(
		"State: %s | Events: %d | Current: %d | Rate: %.1f evt/s | Packets: %d | Bytes: %d | Errors: %d",
		state.String(), metrics.EventsSent, r.currentEventID, eventsPerSec,
		metrics.PacketsSent, metrics.BytesSent, metrics.Errors,
	)
}
