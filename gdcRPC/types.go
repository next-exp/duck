package main

import (
	"fmt"

	duck "github.com/jmbenlloch/next_duck/pkg"
	decoder "github.com/next-exp/decoder_go/pkg"
)

// DecodedWriterEvent represents an event that has been decoded
type DecodedWriterEvent struct {
	DuckEventID int
	decodedEvt  decoder.EventType
}

// TriggerType represents different trigger types for decoded events
type TriggerType int

const (
	Trg0 TriggerType = iota
	Trg1
	Trg2
)

// String returns the string representation of the trigger type
func (t TriggerType) String() string {
	return fmt.Sprintf("%d", t)
}

// DecoderJob represents a job for the decoder worker pool
type DecoderJob struct {
	DuckEventID   int
	Data          []byte
	WritersCh     WriterChannels
	DecoderConfig duck.DecoderConfiguration
}
