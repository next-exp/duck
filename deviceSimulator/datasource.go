package main

// SourceState represents the current state of a data source
type SourceState int

const (
	SourceStateStopped SourceState = iota
	SourceStateRunning
	SourceStatePaused
)

func (s SourceState) String() string {
	switch s {
	case SourceStateStopped:
		return "stopped"
	case SourceStateRunning:
		return "running"
	case SourceStatePaused:
		return "paused"
	default:
		return "unknown"
	}
}

// SourceMetrics holds common metrics for all data sources
type SourceMetrics struct {
	EventsSent     uint64
	PacketsSent    uint64
	BytesSent      uint64
	Errors         uint64
	ErrorsInjected uint64 // Generate mode only
}

// DataSource is the interface that both Replayer and Generator implement
type DataSource interface {
	// Start begins the data source operation
	Start() error

	// Stop stops the data source operation
	Stop() error

	// GetState returns the current state of the data source
	GetState() SourceState

	// GetMetrics returns the current metrics
	GetMetrics() SourceMetrics

	// GetStatus returns a human-readable status string
	GetStatus() string
}
