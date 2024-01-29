package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

type RingBufferData struct {
	data     []byte
	position int
}

// PositionState tracks the state of a single buffer position.
//
// State transitions:
//
//	Free (ref=0, gate full) → Acquired (ref=0, gate empty) → InUse (ref=1, gate empty)
//	InUse → Free (ref=0, gate full)
type PositionState struct {
	refCount  int32         // 0 = free, >0 = in use (atomic operations only)
	writeGate chan struct{} // Semaphore: blocks writer when in use (buffered, size=1)
}

// RingBufferTracking manages concurrent access to a shared ring buffer.
//
// It coordinates between producers (who write to buffer positions) and consumers
// (who read and process the data). The tracking mechanism ensures that producers
// cannot overwrite data that hasn't been consumed yet (backpressure).
//
// Architecture notes:
//   - Single producer, single consumer (current ldcRPC usage)
//   - refCount transitions between 0 and 1 only (binary state)
//   - Implementation supports refCount > 1 for multiple readers, but this is not used
//
// Lifecycle for a buffer position:
//  1. Producer calls TryAcquirePosition() - blocks until position is free
//  2. Producer writes data to the buffer slot
//  3. Producer calls MarkPositionInUse() - signals data is ready
//  4. Consumer reads and processes the data
//  5. Consumer calls ReleasePosition() - makes position available for reuse
type RingBufferTracking struct {
	positions []PositionState
}

// NewRingBufferTracking creates a new tracking structure for nPositions
func NewRingBufferTracking(nPositions int) *RingBufferTracking {
	positions := make([]PositionState, nPositions)
	for i := range positions {
		positions[i].writeGate = make(chan struct{}, 1)
		positions[i].writeGate <- struct{}{} // Pre-fill: position is available
	}
	return &RingBufferTracking{
		positions: positions,
	}
}

// TryAcquirePosition blocks until the position is available for writing.
//
// This method acquires exclusive write access to a buffer position by taking a token
// from the writeGate semaphore. It does NOT modify refCount - that happens in
// MarkPositionInUse().
//
// IMPORTANT - Race Window Documentation:
// There is a window between TryAcquirePosition() and MarkPositionInUse() where
// IsPositionInUse() returns FALSE, even though the position has been acquired
// for writing. This is intentional design - the caller is preparing to write but
// hasn't claimed ownership yet. Callers must NOT use IsPositionInUse() during
// this window to check if they can write.
//
// Correct usage pattern:
//  1. TryAcquirePosition - wait for write permission
//  2. [Write data to buffer]
//  3. MarkPositionInUse - claim ownership (enables IsPositionInUse checks)
//  4. Consumer processes data
//  5. Consumer calls ReleasePosition
//
// Parameters:
//   - position: Index in the ring buffer [0, nPositions)
//   - timeout: Maximum time to wait. 0 means wait indefinitely.
//
// Returns error if:
//   - position is out of range
//   - timeout expires before position becomes available
func (r *RingBufferTracking) TryAcquirePosition(position int, timeout time.Duration) error {
	if position < 0 || position >= len(r.positions) {
		return fmt.Errorf("position %d out of range [0, %d)", position, len(r.positions))
	}

	pos := &r.positions[position]

	if timeout == 0 {
		// Wait indefinitely
		<-pos.writeGate
		return nil
	}

	// Wait with timeout
	select {
	case <-pos.writeGate:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for buffer position %d (refCount: %d)", position, atomic.LoadInt32(&pos.refCount))
	}
}

// MarkPositionInUse increments the reference count after writing to the position.
//
// Must be called after TryAcquirePosition succeeds and data is written to signal
// that the position now contains valid data for consumers to read.
//
// In the current ldcRPC architecture:
//   - This should be called exactly once per acquire (refCount: 0 → 1)
//   - The consumer will later call ReleasePosition() (refCount: 1 → 0)
//
// Note: This method silently returns for out-of-range positions (defensive programming).
// Consider changing to return an error if stricter validation is needed.
func (r *RingBufferTracking) MarkPositionInUse(position int) {
	if position < 0 || position >= len(r.positions) {
		return
	}

	pos := &r.positions[position]
	atomic.AddInt32(&pos.refCount, 1)
}

// ReleasePosition decrements the reference count and signals writers if count reaches 0.
// Must be called by the consumer when done processing data from this position.
// When refCount reaches 0, the position is made available for writers again.
//
// In the current ldcRPC architecture:
//   - This should be called exactly once per MarkPositionInUse (refCount: 1 → 0)
//   - Calling this without a prior MarkPositionInUse indicates a bug
//
// Note: This method silently returns for out-of-range positions (defensive programming).
func (r *RingBufferTracking) ReleasePosition(position int) {
	if position < 0 || position >= len(r.positions) {
		return
	}

	pos := &r.positions[position]
	newCount := atomic.AddInt32(&pos.refCount, -1)

	if newCount == 0 {
		// Position is free, signal writers
		select {
		case pos.writeGate <- struct{}{}:
		default:
			// Gate already signaled (shouldn't happen in normal operation)
		}
	} else if newCount < 0 {
		// This indicates a bug - more releases than marks
		atomic.StoreInt32(&pos.refCount, 0)
		// Try to restore gate if needed
		select {
		case pos.writeGate <- struct{}{}:
		default:
		}
	}
}

// GetRefCount returns the current reference count for a position (for monitoring)
func (r *RingBufferTracking) GetRefCount(position int) int32 {
	if position < 0 || position >= len(r.positions) {
		return -1
	}
	return atomic.LoadInt32(&r.positions[position].refCount)
}

// IsPositionInUse returns true if the position has active references
func (r *RingBufferTracking) IsPositionInUse(position int) bool {
	return r.GetRefCount(position) > 0
}
