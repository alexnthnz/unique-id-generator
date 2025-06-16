package generator

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alexnthnz/unique-id-generator/monitor"
)

const (
	// Custom epoch: January 1, 2020 00:00:00 UTC
	CustomEpoch = 1577836800000 // milliseconds

	// Bit allocation for 64-bit ID
	SequenceBits = 12
	NodeIDBits   = 10
	TimestampBits = 41
	ReservedBits = 1

	// Maximum values
	MaxSequence = (1 << SequenceBits) - 1  // 4095
	MaxNodeID   = (1 << NodeIDBits) - 1    // 1023
	MaxTimestamp = (1 << TimestampBits) - 1 // ~69 years from epoch

	// Bit shifts
	NodeIDShift    = SequenceBits
	TimestampShift = SequenceBits + NodeIDBits
	ReservedShift  = SequenceBits + NodeIDBits + TimestampBits
)

var (
	ErrInvalidNodeID      = errors.New("node ID must be between 0 and 1023")
	ErrClockMovedBackward = errors.New("clock moved backward")
	ErrSequenceExhausted  = errors.New("sequence exhausted for current millisecond")
	ErrTimestampExhausted = errors.New("timestamp exhausted - epoch overflow")
)

// SnowflakeGenerator generates unique IDs using the Snowflake algorithm
type SnowflakeGenerator struct {
	nodeID            uint16
	sequence          uint16
	lastTimestamp     int64
	mutex             sync.Mutex
	metrics           *monitor.Metrics
	clockBackwardWait time.Duration
}

// IDComponents represents the components of a generated ID
type IDComponents struct {
	ID        uint64
	Timestamp int64
	NodeID    uint16
	Sequence  uint16
	Reserved  uint8
}

// NewSnowflakeGenerator creates a new Snowflake ID generator
func NewSnowflakeGenerator(nodeID uint16, metrics *monitor.Metrics) (*SnowflakeGenerator, error) {
	if nodeID > MaxNodeID {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidNodeID, nodeID)
	}

	if metrics == nil {
		metrics = monitor.NewMetrics()
	}

	return &SnowflakeGenerator{
		nodeID:            nodeID,
		sequence:          0,
		lastTimestamp:     -1,
		metrics:           metrics,
		clockBackwardWait: 10 * time.Millisecond, // Max wait time for clock backward
	}, nil
}

// NextID generates the next unique ID
func (g *SnowflakeGenerator) NextID() (uint64, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	timestamp := g.getCurrentTimestamp()
	
	// Handle clock moving backward
	if timestamp < g.lastTimestamp {
		g.metrics.IncrementClockBackward()
		
		// Wait for a short time to see if clock recovers
		time.Sleep(g.clockBackwardWait)
		timestamp = g.getCurrentTimestamp()
		
		if timestamp < g.lastTimestamp {
			return 0, fmt.Errorf("%w: current=%d, last=%d", 
				ErrClockMovedBackward, timestamp, g.lastTimestamp)
		}
	}

	// Check for timestamp exhaustion
	if timestamp > MaxTimestamp {
		return 0, ErrTimestampExhausted
	}

	// Handle sequence management
	if timestamp == g.lastTimestamp {
		// Same millisecond, increment sequence
		g.sequence = (g.sequence + 1) & MaxSequence
		if g.sequence == 0 {
			// Sequence exhausted, wait for next millisecond
			timestamp = g.waitNextMillisecond(timestamp)
		}
	} else {
		// New millisecond, reset sequence
		g.sequence = 0
	}

	g.lastTimestamp = timestamp

	// Construct the ID
	id := g.constructID(timestamp, g.nodeID, g.sequence)
	
	g.metrics.IncrementGenerated()
	return id, nil
}

// BatchNextID generates multiple IDs efficiently
func (g *SnowflakeGenerator) BatchNextID(count int) ([]uint64, error) {
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}

	ids := make([]uint64, 0, count)
	
	for i := 0; i < count; i++ {
		id, err := g.NextID()
		if err != nil {
			return ids, err // Return partial results with error
		}
		ids = append(ids, id)
	}
	
	return ids, nil
}

// ParseID parses an ID into its components
func (g *SnowflakeGenerator) ParseID(id uint64) IDComponents {
	return IDComponents{
		ID:        id,
		Reserved:  uint8((id >> ReservedShift) & 1),
		Timestamp: int64((id >> TimestampShift) & MaxTimestamp),
		NodeID:    uint16((id >> NodeIDShift) & MaxNodeID),
		Sequence:  uint16(id & MaxSequence),
	}
}

// GetTimestampFromID extracts timestamp from ID and converts to actual time
func (g *SnowflakeGenerator) GetTimestampFromID(id uint64) time.Time {
	timestamp := int64((id >> TimestampShift) & MaxTimestamp)
	return time.Unix(0, (timestamp+CustomEpoch)*int64(time.Millisecond))
}

// constructID builds the 64-bit ID from components
func (g *SnowflakeGenerator) constructID(timestamp int64, nodeID uint16, sequence uint16) uint64 {
	return (uint64(timestamp) << TimestampShift) |
		(uint64(nodeID) << NodeIDShift) |
		uint64(sequence)
}

// getCurrentTimestamp returns current timestamp in milliseconds since custom epoch
func (g *SnowflakeGenerator) getCurrentTimestamp() int64 {
	return time.Now().UnixNano()/int64(time.Millisecond) - CustomEpoch
}

// waitNextMillisecond waits until the next millisecond
func (g *SnowflakeGenerator) waitNextMillisecond(lastTimestamp int64) int64 {
	timestamp := g.getCurrentTimestamp()
	for timestamp <= lastTimestamp {
		time.Sleep(100 * time.Microsecond) // Small sleep to avoid busy waiting
		timestamp = g.getCurrentTimestamp()
	}
	return timestamp
}

// GetNodeID returns the node ID of this generator
func (g *SnowflakeGenerator) GetNodeID() uint16 {
	return g.nodeID
}

// GetStats returns current statistics
func (g *SnowflakeGenerator) GetStats() monitor.Stats {
	return g.metrics.GetStats()
}

// SetClockBackwardWait sets the maximum wait time for clock backward recovery
func (g *SnowflakeGenerator) SetClockBackwardWait(duration time.Duration) {
	g.clockBackwardWait = duration
} 