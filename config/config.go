package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the configuration for the unique ID generator
type Config struct {
	// Core generator settings
	NodeID            uint16        `json:"node_id" yaml:"node_id"`
	CustomEpoch       int64         `json:"custom_epoch" yaml:"custom_epoch"`
	ClockBackwardWait time.Duration `json:"clock_backward_wait" yaml:"clock_backward_wait"`

	// HTTP Server settings
	ServerEnabled      bool          `json:"server_enabled" yaml:"server_enabled"`
	ServerPort         int           `json:"server_port" yaml:"server_port"`
	ServerReadTimeout  time.Duration `json:"server_read_timeout" yaml:"server_read_timeout"`
	ServerWriteTimeout time.Duration `json:"server_write_timeout" yaml:"server_write_timeout"`
	ServerIdleTimeout  time.Duration `json:"server_idle_timeout" yaml:"server_idle_timeout"`
	MaxHeaderBytes     int           `json:"max_header_bytes" yaml:"max_header_bytes"`

	// Rate limiting
	RateLimitEnabled bool    `json:"rate_limit_enabled" yaml:"rate_limit_enabled"`
	RateLimitRPS     float64 `json:"rate_limit_rps" yaml:"rate_limit_rps"`
	RateLimitBurst   int     `json:"rate_limit_burst" yaml:"rate_limit_burst"`

	// Batch settings
	MaxBatchSize     int `json:"max_batch_size" yaml:"max_batch_size"`
	DefaultBatchSize int `json:"default_batch_size" yaml:"default_batch_size"`

	// Node management
	AutoNodeID   bool   `json:"auto_node_id" yaml:"auto_node_id"`
	NodeIDSource string `json:"node_id_source" yaml:"node_id_source"` // "auto", "env", "config"

	// Monitoring
	MetricsEnabled     bool `json:"metrics_enabled" yaml:"metrics_enabled"`
	HealthCheckEnabled bool `json:"health_check_enabled" yaml:"health_check_enabled"`

	// Performance tuning
	WorkerPoolSize    int `json:"worker_pool_size" yaml:"worker_pool_size"`
	ChannelBufferSize int `json:"channel_buffer_size" yaml:"channel_buffer_size"`
	GCPercent         int `json:"gc_percent" yaml:"gc_percent"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		NodeID:            0,
		CustomEpoch:       1577836800000, // January 1, 2020 00:00:00 UTC
		ClockBackwardWait: 10 * time.Millisecond,

		ServerEnabled:      false,
		ServerPort:         8080,
		ServerReadTimeout:  10 * time.Second,
		ServerWriteTimeout: 10 * time.Second,
		ServerIdleTimeout:  60 * time.Second,
		MaxHeaderBytes:     1 << 20, // 1 MB

		RateLimitEnabled: true,
		RateLimitRPS:     1000.0,
		RateLimitBurst:   100,

		MaxBatchSize:     10000,
		DefaultBatchSize: 10,

		AutoNodeID:   false,
		NodeIDSource: "config",

		MetricsEnabled:     true,
		HealthCheckEnabled: true,

		WorkerPoolSize:    4,
		ChannelBufferSize: 1000,
		GCPercent:         100,
	}
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(filename string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := DefaultConfig()

	if val := os.Getenv("NODE_ID"); val != "" {
		if nodeID, err := strconv.ParseUint(val, 10, 16); err == nil {
			config.NodeID = uint16(nodeID)
		}
	}

	if val := os.Getenv("AUTO_NODE_ID"); val != "" {
		config.AutoNodeID = val == "true" || val == "1"
	}

	if val := os.Getenv("SERVER_ENABLED"); val != "" {
		config.ServerEnabled = val == "true" || val == "1"
	}

	if val := os.Getenv("SERVER_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil && port > 0 && port <= 65535 {
			config.ServerPort = port
		}
	}

	if val := os.Getenv("RATE_LIMIT_RPS"); val != "" {
		if rps, err := strconv.ParseFloat(val, 64); err == nil && rps > 0 {
			config.RateLimitRPS = rps
		}
	}

	if val := os.Getenv("RATE_LIMIT_BURST"); val != "" {
		if burst, err := strconv.Atoi(val); err == nil && burst > 0 {
			config.RateLimitBurst = burst
		}
	}

	if val := os.Getenv("MAX_BATCH_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			config.MaxBatchSize = size
		}
	}

	if val := os.Getenv("METRICS_ENABLED"); val != "" {
		config.MetricsEnabled = val == "true" || val == "1"
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.NodeID > 1023 {
		return fmt.Errorf("node_id must be between 0 and 1023, got %d", c.NodeID)
	}

	if c.CustomEpoch <= 0 {
		return fmt.Errorf("custom_epoch must be positive, got %d", c.CustomEpoch)
	}

	if c.ClockBackwardWait < 0 {
		return fmt.Errorf("clock_backward_wait must be non-negative, got %v", c.ClockBackwardWait)
	}

	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("server_port must be between 1 and 65535, got %d", c.ServerPort)
	}

	if c.ServerReadTimeout <= 0 {
		return fmt.Errorf("server_read_timeout must be positive, got %v", c.ServerReadTimeout)
	}

	if c.ServerWriteTimeout <= 0 {
		return fmt.Errorf("server_write_timeout must be positive, got %v", c.ServerWriteTimeout)
	}

	if c.RateLimitRPS <= 0 {
		return fmt.Errorf("rate_limit_rps must be positive, got %f", c.RateLimitRPS)
	}

	if c.RateLimitBurst <= 0 {
		return fmt.Errorf("rate_limit_burst must be positive, got %d", c.RateLimitBurst)
	}

	if c.MaxBatchSize <= 0 || c.MaxBatchSize > 100000 {
		return fmt.Errorf("max_batch_size must be between 1 and 100000, got %d", c.MaxBatchSize)
	}

	if c.DefaultBatchSize <= 0 || c.DefaultBatchSize > c.MaxBatchSize {
		return fmt.Errorf("default_batch_size must be between 1 and max_batch_size (%d), got %d",
			c.MaxBatchSize, c.DefaultBatchSize)
	}

	if c.WorkerPoolSize <= 0 || c.WorkerPoolSize > 1000 {
		return fmt.Errorf("worker_pool_size must be between 1 and 1000, got %d", c.WorkerPoolSize)
	}

	if c.ChannelBufferSize < 0 {
		return fmt.Errorf("channel_buffer_size must be non-negative, got %d", c.ChannelBufferSize)
	}

	validNodeIDSources := map[string]bool{"auto": true, "env": true, "config": true}
	if !validNodeIDSources[c.NodeIDSource] {
		return fmt.Errorf("node_id_source must be one of: auto, env, config, got %s", c.NodeIDSource)
	}

	return nil
}

// SaveToFile saves the configuration to a JSON file
func (c *Config) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filename, err)
	}

	return nil
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}

// Merge merges another configuration into this one (other takes precedence)
func (c *Config) Merge(other *Config) {
	if other.NodeID != 0 {
		c.NodeID = other.NodeID
	}
	if other.CustomEpoch != 0 {
		c.CustomEpoch = other.CustomEpoch
	}
	if other.ClockBackwardWait != 0 {
		c.ClockBackwardWait = other.ClockBackwardWait
	}
	if other.ServerPort != 0 {
		c.ServerPort = other.ServerPort
	}
	if other.RateLimitRPS != 0 {
		c.RateLimitRPS = other.RateLimitRPS
	}
	if other.RateLimitBurst != 0 {
		c.RateLimitBurst = other.RateLimitBurst
	}
	if other.MaxBatchSize != 0 {
		c.MaxBatchSize = other.MaxBatchSize
	}
	if other.DefaultBatchSize != 0 {
		c.DefaultBatchSize = other.DefaultBatchSize
	}
	if other.WorkerPoolSize != 0 {
		c.WorkerPoolSize = other.WorkerPoolSize
	}

	// Boolean fields are merged if explicitly set
	c.ServerEnabled = other.ServerEnabled
	c.RateLimitEnabled = other.RateLimitEnabled
	c.AutoNodeID = other.AutoNodeID
	c.MetricsEnabled = other.MetricsEnabled
	c.HealthCheckEnabled = other.HealthCheckEnabled

	// String fields
	if other.NodeIDSource != "" {
		c.NodeIDSource = other.NodeIDSource
	}
}
