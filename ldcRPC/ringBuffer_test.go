package main

import (
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Tests for new RingBufferTracking (reference counting implementation)
// ============================================================================

func TestRingBufferTracking_NewRingBufferTracking(t *testing.T) {
	// Act
	tracking := NewRingBufferTracking(10)

	// Assert
	assert.NotNil(t, tracking)
	assert.Len(t, tracking.positions, 10)

	// All positions should be available (not in use)
	for i := 0; i < 10; i++ {
		assert.False(t, tracking.IsPositionInUse(i), "Position %d should be free", i)
		assert.Equal(t, int32(0), tracking.GetRefCount(i), "Position %d refCount should be 0", i)
	}
}

func TestRingBufferTracking_TryAcquirePosition_Success(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Act
	err := tracking.TryAcquirePosition(0, 100*time.Millisecond)

	// Assert
	assert.NoError(t, err)
}

func TestRingBufferTracking_TryAcquirePosition_OutOfRange(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Act
	err := tracking.TryAcquirePosition(10, 100*time.Millisecond)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestRingBufferTracking_TryAcquirePosition_NegativeIndex(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Act
	err := tracking.TryAcquirePosition(-1, 100*time.Millisecond)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestRingBufferTracking_TryAcquirePosition_BlocksWhenInUse(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Acquire and mark position 0 as in use
	err := tracking.TryAcquirePosition(0, 100*time.Millisecond)
	assert.NoError(t, err)
	tracking.MarkPositionInUse(0)

	// Act: Try to acquire same position - should timeout
	err = tracking.TryAcquirePosition(0, 50*time.Millisecond)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")

	// Cleanup
	tracking.ReleasePosition(0)
}

func TestRingBufferTracking_TryAcquirePosition_UnblocksAfterRelease(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Acquire and mark position 0 as in use
	err := tracking.TryAcquirePosition(0, 100*time.Millisecond)
	assert.NoError(t, err)
	tracking.MarkPositionInUse(0)

	// Start a goroutine that will try to acquire position 0
	acquired := make(chan bool)
	go func() {
		err := tracking.TryAcquirePosition(0, 5*time.Second)
		acquired <- (err == nil)
	}()

	// Give the goroutine time to start blocking
	time.Sleep(50 * time.Millisecond)

	// Release position 0
	tracking.ReleasePosition(0)

	// Assert: The waiting goroutine should have acquired the position
	select {
	case result := <-acquired:
		assert.True(t, result, "Should have acquired position after release")
	case <-time.After(1 * time.Second):
		t.Fatal("Goroutine did not acquire position after release")
	}
}

func TestRingBufferTracking_TryAcquirePosition_ZeroTimeout(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Act: Zero timeout means wait indefinitely (we'll use a separate goroutine)
	done := make(chan bool)
	go func() {
		err := tracking.TryAcquirePosition(0, 0)
		done <- (err == nil)
	}()

	// Assert: Should complete quickly since position is free
	select {
	case result := <-done:
		assert.True(t, result)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("TryAcquirePosition with zero timeout should complete immediately for free position")
	}
}

func TestRingBufferTracking_MarkPositionInUse(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Acquire position first
	tracking.TryAcquirePosition(0, 100*time.Millisecond)

	// Act
	tracking.MarkPositionInUse(0)

	// Assert
	assert.True(t, tracking.IsPositionInUse(0))
	assert.Equal(t, int32(1), tracking.GetRefCount(0))
}

func TestRingBufferTracking_MarkPositionInUse_OutOfRange(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Precondition: Acquire and mark a valid position
	tracking.TryAcquirePosition(5, 100*time.Millisecond)
	tracking.MarkPositionInUse(5)

	// Act: Call with out-of-range indices
	tracking.MarkPositionInUse(10)
	tracking.MarkPositionInUse(-1)

	// Assert: Valid positions should remain unchanged
	assert.Equal(t, int32(1), tracking.GetRefCount(5), "Position 5 should still be in use")
	assert.True(t, tracking.IsPositionInUse(5), "Position 5 should still be marked as in use")

	// Cleanup
	tracking.ReleasePosition(5)
}

func TestRingBufferTracking_ReleasePosition(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Acquire and mark position
	tracking.TryAcquirePosition(0, 100*time.Millisecond)
	tracking.MarkPositionInUse(0)
	assert.True(t, tracking.IsPositionInUse(0))

	// Act
	tracking.ReleasePosition(0)

	// Assert
	assert.False(t, tracking.IsPositionInUse(0))
	assert.Equal(t, int32(0), tracking.GetRefCount(0))
}

func TestRingBufferTracking_ReleasePosition_OutOfRange(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Precondition: Acquire and mark a valid position
	tracking.TryAcquirePosition(5, 100*time.Millisecond)
	tracking.MarkPositionInUse(5)
	assert.Equal(t, int32(1), tracking.GetRefCount(5))

	// Act: Call with out-of-range indices
	tracking.ReleasePosition(10)
	tracking.ReleasePosition(-1)

	// Assert: Valid positions should remain unchanged
	assert.Equal(t, int32(1), tracking.GetRefCount(5), "Position 5 should still be in use after invalid calls")

	// Cleanup: Properly release position 5
	tracking.ReleasePosition(5)
	assert.Equal(t, int32(0), tracking.GetRefCount(5), "Position 5 should be free after proper release")
}

func TestRingBufferTracking_ReleasePosition_MultipleReleases(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Acquire and mark position
	tracking.TryAcquirePosition(0, 100*time.Millisecond)
	tracking.MarkPositionInUse(0)

	// Act: Release multiple times (bug scenario)
	tracking.ReleasePosition(0)
	tracking.ReleasePosition(0) // Should be handled gracefully

	// Assert: RefCount should not go negative
	assert.Equal(t, int32(0), tracking.GetRefCount(0))
}

func TestRingBufferTracking_GetRefCount_OutOfRange(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Act & Assert
	assert.Equal(t, int32(-1), tracking.GetRefCount(10))
	assert.Equal(t, int32(-1), tracking.GetRefCount(-1))
}

func TestRingBufferTracking_IsPositionInUse(t *testing.T) {
	tracking := NewRingBufferTracking(10)

	// Initially not in use
	assert.False(t, tracking.IsPositionInUse(0))

	// Acquire but don't mark
	tracking.TryAcquirePosition(0, 100*time.Millisecond)
	assert.False(t, tracking.IsPositionInUse(0))

	// Mark as in use
	tracking.MarkPositionInUse(0)
	assert.True(t, tracking.IsPositionInUse(0))

	// Release
	tracking.ReleasePosition(0)
	assert.False(t, tracking.IsPositionInUse(0))
}

func TestRingBufferTracking_ConcurrentAcquireRelease(t *testing.T) {
	tracking := NewRingBufferTracking(100)
	numGoroutines := 10
	operationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Each goroutine works on its own set of positions
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			basePos := goroutineID * 10
			for i := 0; i < operationsPerGoroutine; i++ {
				pos := basePos + (i % 10)

				err := tracking.TryAcquirePosition(pos, 5*time.Second)
				if err == nil {
					tracking.MarkPositionInUse(pos)
					// Simulate some work
					time.Sleep(time.Microsecond)
					tracking.ReleasePosition(pos)
				}
			}
		}(g)
	}

	wg.Wait()

	// All positions should be free at the end
	for i := 0; i < 100; i++ {
		assert.False(t, tracking.IsPositionInUse(i), "Position %d should be free", i)
	}
}

func TestRingBufferTracking_ProducerConsumerPattern(t *testing.T) {
	tracking := NewRingBufferTracking(10)
	recvChannel := make(chan int, 100)

	numPackets := 50
	var wg sync.WaitGroup
	wg.Add(2)

	// Producer
	go func() {
		defer wg.Done()
		position := 0
		for i := 0; i < numPackets; i++ {
			err := tracking.TryAcquirePosition(position, 5*time.Second)
			if err != nil {
				t.Errorf("Producer failed to acquire position %d: %v", position, err)
				return
			}
			tracking.MarkPositionInUse(position)
			recvChannel <- position
			position = (position + 1) % 10
		}
		close(recvChannel)
	}()

	// Consumer
	consumedCount := 0
	go func() {
		defer wg.Done()
		for pos := range recvChannel {
			tracking.ReleasePosition(pos)
			consumedCount++
		}
	}()

	wg.Wait()

	// Assert
	assert.Equal(t, numPackets, consumedCount)

	// All positions should be free
	for i := 0; i < 10; i++ {
		assert.False(t, tracking.IsPositionInUse(i))
	}
}

func TestRingBufferTracking_BackpressureScenario(t *testing.T) {
	// Small buffer to easily test backpressure
	tracking := NewRingBufferTracking(3)

	// Fill the buffer
	for i := 0; i < 3; i++ {
		err := tracking.TryAcquirePosition(i, 100*time.Millisecond)
		assert.NoError(t, err)
		tracking.MarkPositionInUse(i)
	}

	// Try to acquire position 0 - should timeout (buffer full, wrapped)
	err := tracking.TryAcquirePosition(0, 50*time.Millisecond)
	assert.Error(t, err)

	// Release position 0
	tracking.ReleasePosition(0)

	// Now should be able to acquire position 0
	err = tracking.TryAcquirePosition(0, 100*time.Millisecond)
	assert.NoError(t, err)
}

func TestRingBufferTracking_RaceWindow_AcquireToMark(t *testing.T) {
	// This test documents and verifies the race window behavior.
	//
	// Between TryAcquirePosition() and MarkPositionInUse(), the position
	// is "acquired" (writeGate token taken) but refCount is still 0.
	// During this window, IsPositionInUse() returns FALSE.
	//
	// This is intentional design - the caller is preparing to write but
	// hasn't claimed ownership yet. This test ensures future maintainers
	// are aware of this behavior.
	tracking := NewRingBufferTracking(10)

	// Act: Acquire position (takes writeGate token)
	err := tracking.TryAcquirePosition(0, 100*time.Millisecond)
	assert.NoError(t, err)

	// Assert: During race window, IsPositionInUse returns FALSE
	// even though we hold the write token
	assert.False(t, tracking.IsPositionInUse(0),
		"Position is acquired but not marked - IsPositionInUse must return false (documented behavior)")
	assert.Equal(t, int32(0), tracking.GetRefCount(0),
		"RefCount should be 0 until MarkPositionInUse is called")

	// Complete the lifecycle
	tracking.MarkPositionInUse(0)
	assert.True(t, tracking.IsPositionInUse(0), "Should be in use after marking")
	assert.Equal(t, int32(1), tracking.GetRefCount(0), "RefCount should be 1 after marking")

	tracking.ReleasePosition(0)
	assert.False(t, tracking.IsPositionInUse(0), "Should be free after release")
}

func TestRingBufferTracking_BugRecovery_DoubleRelease(t *testing.T) {
	// This test documents a bug in the current implementation:
	// When ReleasePosition is called more times than MarkPositionInUse,
	// the recovery logic can create duplicate tokens in writeGate.

	tracking := NewRingBufferTracking(10)

	// Normal lifecycle
	err := tracking.TryAcquirePosition(0, 100*time.Millisecond)
	assert.NoError(t, err)
	tracking.MarkPositionInUse(0)
	tracking.ReleasePosition(0)

	// This is a BUG - double release (refCount goes 1 → 0 → -1)
	tracking.ReleasePosition(0)

	// After bug recovery, position should still be usable
	err = tracking.TryAcquirePosition(0, 100*time.Millisecond)
	assert.NoError(t, err, "Should recover from double release")

	// CRITICAL: Verify we didn't create duplicate tokens
	// If the bug exists, we can acquire the same position twice
	secondTry := make(chan error, 1)
	go func() {
		secondTry <- tracking.TryAcquirePosition(0, 50*time.Millisecond)
	}()

	// Should timeout because we already acquired it once
	// If this succeeds, we have duplicate tokens (BUG)
	select {
	case err := <-secondTry:
		if err == nil {
			t.Error("BUG DETECTED: Double recovery created duplicate writeGate tokens. " +
				"Position can be acquired twice without release. " +
				"See ringBuffer.go lines 90-98 for fix.")
		}
	case <-time.After(100 * time.Millisecond):
		// Expected: second acquire should timeout
	}

	// Cleanup
	tracking.MarkPositionInUse(0)
	tracking.ReleasePosition(0)
}

func TestRingBufferTracking_RingWrappingStress(t *testing.T) {
	// Stress test for ring buffer wrapping behavior
	// Verifies that state remains consistent after many full rotations
	bufferSize := 10
	numCycles := 1000 // Complete 1000 rotations around the ring
	tracking := NewRingBufferTracking(bufferSize)

	position := 0
	for i := 0; i < numCycles*bufferSize; i++ {
		// Acquire
		err := tracking.TryAcquirePosition(position, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to acquire position %d at iteration %d: %v", position, i, err)
		}

		// Mark
		tracking.MarkPositionInUse(position)

		// Verify state
		if !tracking.IsPositionInUse(position) {
			t.Fatalf("Position %d should be in use at iteration %d", position, i)
		}
		if tracking.GetRefCount(position) != 1 {
			t.Fatalf("Position %d refCount should be 1 at iteration %d, got %d",
				position, i, tracking.GetRefCount(position))
		}

		// Release
		tracking.ReleasePosition(position)

		// Verify state after release
		if tracking.IsPositionInUse(position) {
			t.Fatalf("Position %d should be free after release at iteration %d", position, i)
		}
		if tracking.GetRefCount(position) != 0 {
			t.Fatalf("Position %d refCount should be 0 after release at iteration %d, got %d",
				position, i, tracking.GetRefCount(position))
		}

		// Move to next position (with wraparound)
		position = (position + 1) % bufferSize
	}

	// Final verification: all positions should be free
	for i := 0; i < bufferSize; i++ {
		if tracking.IsPositionInUse(i) {
			t.Errorf("Position %d should be free after stress test", i)
		}
		if tracking.GetRefCount(i) != 0 {
			t.Errorf("Position %d refCount should be 0 after stress test, got %d",
				i, tracking.GetRefCount(i))
		}
	}
}

// Benchmarks for RingBufferTracking
func BenchmarkRingBufferTracking_AcquireMarkRelease(b *testing.B) {
	tracking := NewRingBufferTracking(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := i % 1000
		tracking.TryAcquirePosition(pos, 0)
		tracking.MarkPositionInUse(pos)
		tracking.ReleasePosition(pos)
	}
}

func BenchmarkRingBufferTracking_ConcurrentAccess(b *testing.B) {
	tracking := NewRingBufferTracking(1000)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pos := i % 1000
			tracking.TryAcquirePosition(pos, 0)
			tracking.MarkPositionInUse(pos)
			tracking.ReleasePosition(pos)
			i++
		}
	})
}

// ============================================================================
// Data Integrity Tests (Integration Tests with Actual Byte Buffer)
// ============================================================================

// TestRingBuffer_DataIntegrity is an integration test that verifies RingBufferTracking
// correctly manages access to an actual byte buffer with producer/consumer pattern.
//
// This test addresses the gap in the original test suite, which only tested the
// synchronization primitives without verifying they protect actual data correctly.
func TestRingBuffer_DataIntegrity(t *testing.T) {
	bufferSize := 10
	slotSize := 128 // bytes per slot

	// Create the actual data buffer
	buffer := make([]byte, bufferSize*slotSize)
	tracking := NewRingBufferTracking(bufferSize)

	numPackets := 10000 // Large number to catch race conditions
	var wg sync.WaitGroup
	wg.Add(2)

	// Error channel for both goroutines
	errors := make(chan error, 100)

	// Producer: Writes unique, verifiable patterns to buffer slots
	go func() {
		defer wg.Done()
		position := 0

		for sequence := 0; sequence < numPackets; sequence++ {
			// Try to acquire position with backpressure
			err := tracking.TryAcquirePosition(position, 5*time.Second)
			if err != nil {
				errors <- assert.AnError
				return
			}

			// Write data to buffer slot
			slotOffset := position * slotSize
			slot := buffer[slotOffset : slotOffset+slotSize]

			// Write unique pattern: sequence number + position
			// This makes each packet uniquely identifiable
			binary.BigEndian.PutUint64(slot[0:8], uint64(sequence))     // Sequence number
			binary.BigEndian.PutUint64(slot[8:16], uint64(position))    // Position
			binary.BigEndian.PutUint64(slot[16:24], uint64(slotSize))   // Slot size (sanity check)
			binary.BigEndian.PutUint64(slot[24:32], 0xDEADBEEFCAFE1234) // Magic number

			// Mark as ready for consumption
			tracking.MarkPositionInUse(position)

			position = (position + 1) % bufferSize
		}
	}()

	// Consumer: Reads from buffer, verifies pattern integrity, releases positions
	consumedCount := 0
	expectedSequence := 0

	go func() {
		defer wg.Done()
		position := 0

		for consumedCount < numPackets {
			// Wait for data to be available
			// In real system, this would be signaled by producer
			// Here we poll with a small delay
			for !tracking.IsPositionInUse(position) {
				time.Sleep(10 * time.Microsecond)
			}

			// Read data from buffer slot
			slotOffset := position * slotSize
			slot := buffer[slotOffset : slotOffset+slotSize]

			// Verify data integrity
			sequence := binary.BigEndian.Uint64(slot[0:8])
			pos := binary.BigEndian.Uint64(slot[8:16])
			size := binary.BigEndian.Uint64(slot[16:24])
			magic := binary.BigEndian.Uint64(slot[24:32])

			// Assertions for data integrity
			if sequence != uint64(expectedSequence) {
				t.Errorf("Sequence mismatch at position %d: expected %d, got %d",
					position, expectedSequence, sequence)
				errors <- assert.AnError
				return
			}

			if pos != uint64(position) {
				t.Errorf("Position mismatch: expected %d, got %d", position, pos)
				errors <- assert.AnError
				return
			}

			if size != uint64(slotSize) {
				t.Errorf("Size mismatch: expected %d, got %d", slotSize, size)
				errors <- assert.AnError
				return
			}

			if magic != 0xDEADBEEFCAFE1234 {
				t.Errorf("Magic number corrupted at position %d: got %016X", position, magic)
				errors <- assert.AnError
				return
			}

			// Release position for reuse
			tracking.ReleasePosition(position)

			consumedCount++
			expectedSequence++
			position = (position + 1) % bufferSize
		}
	}()

	// Wait for both goroutines to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			t.Fatalf("Test failed with errors")
		}
	}

	// Final assertions
	assert.Equal(t, numPackets, consumedCount, "Should consume all packets")

	// All positions should be free at the end
	for i := 0; i < bufferSize; i++ {
		assert.False(t, tracking.IsPositionInUse(i), "Position %d should be free", i)
		assert.Equal(t, int32(0), tracking.GetRefCount(i), "Position %d refCount should be 0", i)
	}
}

// TestRingBuffer_DataIntegrity_BoundaryConditions tests the critical wraparound
// scenario where the producer wraps from the last position back to the first.
func TestRingBuffer_DataIntegrity_BoundaryConditions(t *testing.T) {
	bufferSize := 5
	slotSize := 32

	buffer := make([]byte, bufferSize*slotSize)
	tracking := NewRingBufferTracking(bufferSize)

	// Test multiple wraparounds
	numWraps := 10
	numPackets := bufferSize * numWraps

	var wg sync.WaitGroup
	wg.Add(2)

	errors := make(chan error, 10)

	// Producer
	go func() {
		defer wg.Done()
		position := 0

		for sequence := 0; sequence < numPackets; sequence++ {
			err := tracking.TryAcquirePosition(position, 2*time.Second)
			if err != nil {
				errors <- assert.AnError
				return
			}

			// Write sequence number at wraparound points for verification
			slotOffset := position * slotSize
			binary.BigEndian.PutUint64(buffer[slotOffset:slotOffset+8], uint64(sequence))

			tracking.MarkPositionInUse(position)
			position = (position + 1) % bufferSize
		}
	}()

	// Consumer
	go func() {
		defer wg.Done()
		position := 0
		expectedSeq := 0

		for i := 0; i < numPackets; i++ {
			// Wait for data
			for !tracking.IsPositionInUse(position) {
				time.Sleep(time.Microsecond)
			}

			// Verify sequence
			slotOffset := position * slotSize
			sequence := binary.BigEndian.Uint64(buffer[slotOffset : slotOffset+8])

			if sequence != uint64(expectedSeq) {
				t.Errorf("Boundary test: sequence mismatch at position %d: expected %d, got %d",
					position, expectedSeq, sequence)
				errors <- assert.AnError
				return
			}

			tracking.ReleasePosition(position)
			expectedSeq++
			position = (position + 1) % bufferSize
		}
	}()

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatal("Boundary test failed")
		}
	}
}

// TestRingBuffer_DataIntegrity_SlowConsumer tests backpressure behavior.
// The consumer deliberately lags behind, forcing the producer to block.
func TestRingBuffer_DataIntegrity_SlowConsumer(t *testing.T) {
	bufferSize := 3
	slotSize := 32

	buffer := make([]byte, bufferSize*slotSize)
	tracking := NewRingBufferTracking(bufferSize)

	numPackets := 20
	var wg sync.WaitGroup
	wg.Add(2)

	errors := make(chan error, 10)

	// Producer: Fast producer
	producedCount := 0
	go func() {
		defer wg.Done()
		position := 0

		for producedCount < numPackets {
			// This should block when buffer is full (backpressure)
			err := tracking.TryAcquirePosition(position, 2*time.Second)
			if err != nil {
				errors <- assert.AnError
				return
			}

			slotOffset := position * slotSize
			binary.BigEndian.PutUint64(buffer[slotOffset:slotOffset+8], uint64(producedCount))

			tracking.MarkPositionInUse(position)
			producedCount++
			position = (position + 1) % bufferSize
		}
	}()

	// Consumer: Slow consumer with random delays
	consumedCount := 0
	go func() {
		defer wg.Done()
		position := 0
		expectedSeq := 0

		for consumedCount < numPackets {
			// Wait for data
			for !tracking.IsPositionInUse(position) {
				time.Sleep(100 * time.Microsecond)
			}

			// Verify data
			slotOffset := position * slotSize
			sequence := binary.BigEndian.Uint64(buffer[slotOffset : slotOffset+8])

			if sequence != uint64(expectedSeq) {
				t.Errorf("Slow consumer: sequence mismatch at position %d", position)
				errors <- assert.AnError
				return
			}

			// Simulate slow processing
			time.Sleep(10 * time.Millisecond)

			tracking.ReleasePosition(position)
			consumedCount++
			expectedSeq++
			position = (position + 1) % bufferSize
		}
	}()

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatal("Slow consumer test failed")
		}
	}

	assert.Equal(t, numPackets, producedCount, "All packets should be produced")
	assert.Equal(t, numPackets, consumedCount, "All packets should be consumed")
}

// Benchmark for the full producer-consumer data flow
func BenchmarkRingBuffer_DataIntegrity(b *testing.B) {
	bufferSize := 100
	slotSize := 128

	buffer := make([]byte, bufferSize*slotSize)
	tracking := NewRingBufferTracking(bufferSize)

	b.ResetTimer()

	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(1)

		// Producer
		go func(seq int) {
			defer wg.Done()
			pos := seq % bufferSize

			tracking.TryAcquirePosition(pos, 0)
			slotOffset := pos * slotSize
			binary.BigEndian.PutUint64(buffer[slotOffset:slotOffset+8], uint64(seq))
			tracking.MarkPositionInUse(pos)

			// Immediate consumer
			tracking.ReleasePosition(pos)
		}(i)
	}

	wg.Wait()
}
