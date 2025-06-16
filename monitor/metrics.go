package monitor

import (
	"sync/atomic"
	"time"
)

// Stats represents the current statistics
type Stats struct {
	TotalGenerated      uint64
	TotalErrors         uint64
	ClockBackwardCount  uint64
	SequenceExhausted   uint64
	LastGeneratedTime   time.Time
	AverageLatency      time.Duration
	PeakIDsPerSecond    float64
}

// Metrics tracks various metrics for the ID generator
type Metrics struct {
	totalGenerated     uint64
	totalErrors        uint64
	clockBackwardCount uint64
	sequenceExhausted  uint64
	startTime          time.Time
	lastResetTime      time.Time
	
	// For calculating rates
	lastGeneratedCount uint64
	lastMeasureTime    time.Time
	peakRate          float64
	
	// For latency tracking
	totalLatency      uint64 // nanoseconds
	latencyCount      uint64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	now := time.Now()
	return &Metrics{
		startTime:       now,
		lastResetTime:   now,
		lastMeasureTime: now,
	}
}

// IncrementGenerated increments the generated counter
func (m *Metrics) IncrementGenerated() {
	atomic.AddUint64(&m.totalGenerated, 1)
}

// IncrementErrors increments the error counter
func (m *Metrics) IncrementErrors() {
	atomic.AddUint64(&m.totalErrors, 1)
}

// IncrementClockBackward increments the clock backward counter
func (m *Metrics) IncrementClockBackward() {
	atomic.AddUint64(&m.clockBackwardCount, 1)
}

// IncrementSequenceExhausted increments the sequence exhausted counter
func (m *Metrics) IncrementSequenceExhausted() {
	atomic.AddUint64(&m.sequenceExhausted, 1)
}

// RecordLatency records the latency for ID generation
func (m *Metrics) RecordLatency(duration time.Duration) {
	atomic.AddUint64(&m.totalLatency, uint64(duration.Nanoseconds()))
	atomic.AddUint64(&m.latencyCount, 1)
}

// GetStats returns current statistics
func (m *Metrics) GetStats() Stats {
	totalGenerated := atomic.LoadUint64(&m.totalGenerated)
	totalErrors := atomic.LoadUint64(&m.totalErrors)
	clockBackwardCount := atomic.LoadUint64(&m.clockBackwardCount)
	sequenceExhausted := atomic.LoadUint64(&m.sequenceExhausted)
	totalLatency := atomic.LoadUint64(&m.totalLatency)
	latencyCount := atomic.LoadUint64(&m.latencyCount)
	
	var avgLatency time.Duration
	if latencyCount > 0 {
		avgLatency = time.Duration(totalLatency / latencyCount)
	}
	
	// Calculate current rate
	now := time.Now()
	currentRate := m.calculateCurrentRate(totalGenerated, now)
	if currentRate > m.peakRate {
		m.peakRate = currentRate
	}
	
	return Stats{
		TotalGenerated:     totalGenerated,
		TotalErrors:        totalErrors,
		ClockBackwardCount: clockBackwardCount,
		SequenceExhausted:  sequenceExhausted,
		LastGeneratedTime:  now,
		AverageLatency:     avgLatency,
		PeakIDsPerSecond:   m.peakRate,
	}
}

// calculateCurrentRate calculates the current generation rate
func (m *Metrics) calculateCurrentRate(currentTotal uint64, now time.Time) float64 {
	timeDiff := now.Sub(m.lastMeasureTime).Seconds()
	if timeDiff < 1.0 {
		return 0 // Don't calculate for less than 1 second
	}
	
	countDiff := currentTotal - m.lastGeneratedCount
	rate := float64(countDiff) / timeDiff
	
	m.lastGeneratedCount = currentTotal
	m.lastMeasureTime = now
	
	return rate
}

// Reset resets all counters
func (m *Metrics) Reset() {
	atomic.StoreUint64(&m.totalGenerated, 0)
	atomic.StoreUint64(&m.totalErrors, 0)
	atomic.StoreUint64(&m.clockBackwardCount, 0)
	atomic.StoreUint64(&m.sequenceExhausted, 0)
	atomic.StoreUint64(&m.totalLatency, 0)
	atomic.StoreUint64(&m.latencyCount, 0)
	
	now := time.Now()
	m.lastResetTime = now
	m.lastMeasureTime = now
	m.lastGeneratedCount = 0
	m.peakRate = 0
}

// GetUptime returns the uptime since creation
func (m *Metrics) GetUptime() time.Duration {
	return time.Since(m.startTime)
}

// GetTimeSinceReset returns time since last reset
func (m *Metrics) GetTimeSinceReset() time.Duration {
	return time.Since(m.lastResetTime)
}

// PerformanceTracker helps track performance over time windows
type PerformanceTracker struct {
	windowSize time.Duration
	samples    []sample
	index      int
	full       bool
}

type sample struct {
	timestamp time.Time
	count     uint64
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(windowSize time.Duration, sampleCount int) *PerformanceTracker {
	return &PerformanceTracker{
		windowSize: windowSize,
		samples:    make([]sample, sampleCount),
	}
}

// Record records a new sample
func (pt *PerformanceTracker) Record(count uint64) {
	pt.samples[pt.index] = sample{time.Now(), count}
	pt.index = (pt.index + 1) % len(pt.samples)
	if pt.index == 0 {
		pt.full = true
	}
}

// GetRate returns the current rate over the window
func (pt *PerformanceTracker) GetRate() float64 {
	if !pt.full && pt.index < 2 {
		return 0
	}
	
	var oldest, newest sample
	if pt.full {
		oldest = pt.samples[pt.index]
		newest = pt.samples[(pt.index-1+len(pt.samples))%len(pt.samples)]
	} else {
		oldest = pt.samples[0]
		newest = pt.samples[pt.index-1]
	}
	
	timeDiff := newest.timestamp.Sub(oldest.timestamp).Seconds()
	if timeDiff <= 0 {
		return 0
	}
	
	countDiff := newest.count - oldest.count
	return float64(countDiff) / timeDiff
} 