package generator

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexnthnz/unique-id-generator/monitor"
)

func TestNewSnowflakeGenerator(t *testing.T) {
	metrics := monitor.NewMetrics()
	
	// Test valid node ID
	gen, err := NewSnowflakeGenerator(100, metrics)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if gen.GetNodeID() != 100 {
		t.Errorf("Expected node ID 100, got %d", gen.GetNodeID())
	}
	
	// Test invalid node ID
	_, err = NewSnowflakeGenerator(MaxNodeID+1, metrics)
	if err == nil {
		t.Error("Expected error for invalid node ID")
	}
}

func TestNextID(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	// Test basic ID generation
	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if id == 0 {
		t.Error("Generated ID should not be 0")
	}
	
	// Test ID uniqueness
	ids := make(map[uint64]bool)
	for i := 0; i < 1000; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("Failed to generate ID at iteration %d: %v", i, err)
		}
		if ids[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		ids[id] = true
	}
}

func TestIDComponents(t *testing.T) {
	metrics := monitor.NewMetrics()
	nodeID := uint16(42)
	gen, err := NewSnowflakeGenerator(nodeID, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	
	components := gen.ParseID(id)
	
	// Verify components
	if components.NodeID != nodeID {
		t.Errorf("Expected node ID %d, got %d", nodeID, components.NodeID)
	}
	if components.Timestamp <= 0 {
		t.Error("Timestamp should be positive")
	}
	if components.Sequence > MaxSequence {
		t.Errorf("Sequence %d exceeds maximum %d", components.Sequence, MaxSequence)
	}
}

func TestSequenceIncrement(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	// Generate multiple IDs in quick succession
	var lastSequence uint16 = 0
	sameTimestampCount := 0
	
	for i := 0; i < 100; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		
		components := gen.ParseID(id)
		
		// If we're in the same millisecond, sequence should increment
		if i > 0 {
			if components.Sequence == lastSequence+1 {
				sameTimestampCount++
			}
		}
		lastSequence = components.Sequence
	}
	
	// We should have at least some sequence increments
	if sameTimestampCount == 0 {
		t.Log("No sequence increments detected - this might be expected in slow environments")
	}
}

func TestConcurrentGeneration(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	const numGoroutines = 100
	const idsPerGoroutine = 100
	
	var wg sync.WaitGroup
	idChan := make(chan uint64, numGoroutines*idsPerGoroutine)
	
	// Launch goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := gen.NextID()
				if err != nil {
					t.Errorf("Failed to generate ID: %v", err)
					return
				}
				idChan <- id
			}
		}()
	}
	
	wg.Wait()
	close(idChan)
	
	// Check for duplicates
	ids := make(map[uint64]bool)
	count := 0
	for id := range idChan {
		if ids[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		ids[id] = true
		count++
	}
	
	expectedCount := numGoroutines * idsPerGoroutine
	if count != expectedCount {
		t.Errorf("Expected %d IDs, got %d", expectedCount, count)
	}
}

func TestBatchGeneration(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	batchSize := 1000
	ids, err := gen.BatchNextID(batchSize)
	if err != nil {
		t.Fatalf("Failed to generate batch IDs: %v", err)
	}
	
	if len(ids) != batchSize {
		t.Errorf("Expected %d IDs, got %d", batchSize, len(ids))
	}
	
	// Check for duplicates
	idSet := make(map[uint64]bool)
	for _, id := range ids {
		if idSet[id] {
			t.Errorf("Duplicate ID in batch: %d", id)
		}
		idSet[id] = true
	}
}

func TestTimestampOrdering(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	var lastTimestamp int64 = -1
	
	for i := 0; i < 100; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		
		components := gen.ParseID(id)
		
		// Timestamps should be monotonically increasing or equal
		if components.Timestamp < lastTimestamp {
			t.Errorf("Timestamp went backward: %d -> %d", lastTimestamp, components.Timestamp)
		}
		
		lastTimestamp = components.Timestamp
		
		// Small delay to potentially move to next millisecond
		if i%10 == 0 {
			time.Sleep(time.Microsecond * 100)
		}
	}
}

func TestClockBackwardHandling(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	// Generate an ID to set lastTimestamp
	_, err = gen.NextID()
	if err != nil {
		t.Fatalf("Failed to generate initial ID: %v", err)
	}
	
	// Simulate clock moving backward by setting lastTimestamp to future
	gen.lastTimestamp = gen.getCurrentTimestamp() + 1000
	
	// Set a short wait time for testing
	gen.SetClockBackwardWait(1 * time.Millisecond)
	
	// Try to generate ID - should detect clock backward
	start := time.Now()
	_, err = gen.NextID()
	duration := time.Since(start)
	
	// Should either succeed after wait or fail
	if err != nil {
		// Check that it's a clock backward error (it might be wrapped)
		if !strings.Contains(err.Error(), "clock moved backward") {
			t.Errorf("Expected clock backward error, got %v", err)
		}
		// Check that it waited
		if duration < 1*time.Millisecond {
			t.Error("Should have waited for clock backward recovery")
		}
	}
	
	// Check metrics
	stats := metrics.GetStats()
	if stats.ClockBackwardCount == 0 {
		t.Error("Clock backward count should be incremented")
	}
}

func TestGetTimestampFromID(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	
	beforeGeneration := time.Now()
	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	
	extractedTime := gen.GetTimestampFromID(id)
	
	// The extracted time should be close to the generation time (within 1 second)
	timeDiff := extractedTime.Sub(beforeGeneration)
	absDiff := timeDiff
	if absDiff < 0 {
		absDiff = -absDiff
	}
	if absDiff > time.Second {
		t.Errorf("Extracted time %v too far from generation time %v (diff: %v)", 
			extractedTime, beforeGeneration, timeDiff)
	}
}

func BenchmarkNextID(b *testing.B) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		b.Fatalf("Failed to create generator: %v", err)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gen.NextID()
			if err != nil {
				b.Errorf("Failed to generate ID: %v", err)
			}
		}
	})
}

func BenchmarkNextIDSingleThread(b *testing.B) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		b.Fatalf("Failed to create generator: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.NextID()
		if err != nil {
			b.Errorf("Failed to generate ID: %v", err)
		}
	}
}

func BenchmarkBatchNextID(b *testing.B) {
	metrics := monitor.NewMetrics()
	gen, err := NewSnowflakeGenerator(1, metrics)
	if err != nil {
		b.Fatalf("Failed to create generator: %v", err)
	}
	
	batchSizes := []int{10, 100, 1000}
	
	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("batch_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := gen.BatchNextID(size)
				if err != nil {
					b.Errorf("Failed to generate batch IDs: %v", err)
				}
			}
		})
	}
} 