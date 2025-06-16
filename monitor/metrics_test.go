package monitor

import (
	"sync"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}
	
	// Check initial state
	stats := m.GetStats()
	if stats.TotalGenerated != 0 {
		t.Errorf("Expected TotalGenerated to be 0, got %d", stats.TotalGenerated)
	}
	if stats.TotalErrors != 0 {
		t.Errorf("Expected TotalErrors to be 0, got %d", stats.TotalErrors)
	}
	if stats.ClockBackwardCount != 0 {
		t.Errorf("Expected ClockBackwardCount to be 0, got %d", stats.ClockBackwardCount)
	}
}

func TestIncrementGenerated(t *testing.T) {
	m := NewMetrics()
	
	// Test single increment
	m.IncrementGenerated()
	stats := m.GetStats()
	if stats.TotalGenerated != 1 {
		t.Errorf("Expected TotalGenerated to be 1, got %d", stats.TotalGenerated)
	}
	
	// Test multiple increments
	for i := 0; i < 99; i++ {
		m.IncrementGenerated()
	}
	stats = m.GetStats()
	if stats.TotalGenerated != 100 {
		t.Errorf("Expected TotalGenerated to be 100, got %d", stats.TotalGenerated)
	}
}

func TestIncrementErrors(t *testing.T) {
	m := NewMetrics()
	
	m.IncrementErrors()
	stats := m.GetStats()
	if stats.TotalErrors != 1 {
		t.Errorf("Expected TotalErrors to be 1, got %d", stats.TotalErrors)
	}
}

func TestIncrementClockBackward(t *testing.T) {
	m := NewMetrics()
	
	m.IncrementClockBackward()
	stats := m.GetStats()
	if stats.ClockBackwardCount != 1 {
		t.Errorf("Expected ClockBackwardCount to be 1, got %d", stats.ClockBackwardCount)
	}
}

func TestIncrementSequenceExhausted(t *testing.T) {
	m := NewMetrics()
	
	m.IncrementSequenceExhausted()
	stats := m.GetStats()
	if stats.SequenceExhausted != 1 {
		t.Errorf("Expected SequenceExhausted to be 1, got %d", stats.SequenceExhausted)
	}
}

func TestRecordLatency(t *testing.T) {
	m := NewMetrics()
	
	// Record a latency
	latency := 5 * time.Millisecond
	m.RecordLatency(latency)
	
	stats := m.GetStats()
	if stats.AverageLatency != latency {
		t.Errorf("Expected AverageLatency to be %v, got %v", latency, stats.AverageLatency)
	}
	
	// Record another latency and check average
	latency2 := 3 * time.Millisecond
	m.RecordLatency(latency2)
	
	stats = m.GetStats()
	expectedAvg := (latency + latency2) / 2
	if stats.AverageLatency != expectedAvg {
		t.Errorf("Expected AverageLatency to be %v, got %v", expectedAvg, stats.AverageLatency)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := NewMetrics()
	
	const numGoroutines = 100
	const incrementsPerGoroutine = 100
	
	var wg sync.WaitGroup
	
	// Test concurrent increments
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				m.IncrementGenerated()
				m.IncrementErrors()
				m.IncrementClockBackward()
				m.RecordLatency(time.Microsecond)
			}
		}()
	}
	
	wg.Wait()
	
	stats := m.GetStats()
	expected := uint64(numGoroutines * incrementsPerGoroutine)
	
	if stats.TotalGenerated != expected {
		t.Errorf("Expected TotalGenerated to be %d, got %d", expected, stats.TotalGenerated)
	}
	if stats.TotalErrors != expected {
		t.Errorf("Expected TotalErrors to be %d, got %d", expected, stats.TotalErrors)
	}
	if stats.ClockBackwardCount != expected {
		t.Errorf("Expected ClockBackwardCount to be %d, got %d", expected, stats.ClockBackwardCount)
	}
}

func TestReset(t *testing.T) {
	m := NewMetrics()
	
	// Add some metrics
	m.IncrementGenerated()
	m.IncrementErrors()
	m.IncrementClockBackward()
	m.RecordLatency(time.Millisecond)
	
	// Verify metrics are set
	stats := m.GetStats()
	if stats.TotalGenerated == 0 || stats.TotalErrors == 0 || stats.ClockBackwardCount == 0 {
		t.Error("Metrics should be non-zero before reset")
	}
	
	// Reset and verify
	m.Reset()
	stats = m.GetStats()
	
	if stats.TotalGenerated != 0 {
		t.Errorf("Expected TotalGenerated to be 0 after reset, got %d", stats.TotalGenerated)
	}
	if stats.TotalErrors != 0 {
		t.Errorf("Expected TotalErrors to be 0 after reset, got %d", stats.TotalErrors)
	}
	if stats.ClockBackwardCount != 0 {
		t.Errorf("Expected ClockBackwardCount to be 0 after reset, got %d", stats.ClockBackwardCount)
	}
	if stats.AverageLatency != 0 {
		t.Errorf("Expected AverageLatency to be 0 after reset, got %v", stats.AverageLatency)
	}
}

func TestGetUptime(t *testing.T) {
	m := NewMetrics()
	
	// Small delay to ensure uptime is non-zero
	time.Sleep(time.Millisecond)
	
	uptime := m.GetUptime()
	if uptime <= 0 {
		t.Error("Uptime should be positive")
	}
	if uptime > time.Second {
		t.Error("Uptime should be less than a second for new metrics")
	}
}

func TestPerformanceTracker(t *testing.T) {
	tracker := NewPerformanceTracker(time.Second, 10)
	
	if tracker == nil {
		t.Fatal("NewPerformanceTracker returned nil")
	}
	
	// Test initial rate
	rate := tracker.GetRate()
	if rate != 0 {
		t.Errorf("Expected initial rate to be 0, got %f", rate)
	}
	
	// Record some samples
	tracker.Record(100)
	time.Sleep(10 * time.Millisecond)
	tracker.Record(200)
	
	rate = tracker.GetRate()
	if rate < 0 {
		t.Errorf("Rate should be non-negative, got %f", rate)
	}
}

func TestPerformanceTrackerFullWindow(t *testing.T) {
	tracker := NewPerformanceTracker(time.Second, 3)
	
	// Fill the tracker beyond its capacity
	for i := 0; i < 5; i++ {
		tracker.Record(uint64(i * 100))
		time.Sleep(time.Millisecond)
	}
	
	rate := tracker.GetRate()
	if rate < 0 {
		t.Errorf("Rate should be non-negative even with full window, got %f", rate)
	}
}

// Benchmark tests
func BenchmarkIncrementGenerated(b *testing.B) {
	m := NewMetrics()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.IncrementGenerated()
		}
	})
}

func BenchmarkRecordLatency(b *testing.B) {
	m := NewMetrics()
	latency := time.Microsecond
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.RecordLatency(latency)
		}
	})
}

func BenchmarkGetStats(b *testing.B) {
	m := NewMetrics()
	
	// Pre-populate some metrics
	m.IncrementGenerated()
	m.IncrementErrors()
	m.RecordLatency(time.Microsecond)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetStats()
	}
} 