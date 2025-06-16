package main

import (
	"testing"
	"time"

	"github.com/alexnthnz/unique-id-generator/generator"
	"github.com/alexnthnz/unique-id-generator/monitor"
)

func TestRunBenchmark(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := generator.NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// This should run without panicking
	runBenchmark(gen, metrics)

	// Verify that some operations were performed
	stats := metrics.GetStats()
	if stats.TotalGenerated == 0 {
		t.Error("Expected some IDs to be generated during benchmark")
	}
}

func TestHTTPServerCreation(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := generator.NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	server := NewHTTPServer(gen, metrics)
	if server == nil {
		t.Error("NewHTTPServer should not return nil")
	}

	if server.generator == nil {
		t.Error("HTTP server should have a generator")
	}

	if server.metrics == nil {
		t.Error("HTTP server should have metrics")
	}
}

func TestGeneratorConfiguration(t *testing.T) {
	metrics := monitor.NewMetrics()

	// Test valid node ID
	gen, err := generator.NewSnowflakeGenerator(42, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator with valid node ID: %v", err)
	}

	if gen.GetNodeID() != 42 {
		t.Errorf("Expected node ID 42, got %d", gen.GetNodeID())
	}

	// Test ID generation
	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if id == 0 {
		t.Error("Generated ID should not be zero")
	}

	// Test ID parsing
	components := gen.ParseID(id)
	if components.NodeID != 42 {
		t.Errorf("Parsed node ID should be 42, got %d", components.NodeID)
	}
}

func TestMetricsIntegration(t *testing.T) {
	metrics := monitor.NewMetrics()
	gen, err := generator.NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	initialStats := metrics.GetStats()

	// Generate some IDs
	for i := 0; i < 10; i++ {
		_, err := gen.NextID()
		if err != nil {
			t.Fatalf("Failed to generate ID %d: %v", i, err)
		}
	}

	finalStats := metrics.GetStats()

	if finalStats.TotalGenerated <= initialStats.TotalGenerated {
		t.Error("Total generated count should have increased")
	}

	expectedIncrease := finalStats.TotalGenerated - initialStats.TotalGenerated
	if expectedIncrease != 10 {
		t.Errorf("Expected 10 more generated IDs, got %d", expectedIncrease)
	}
}

func TestBenchmarkPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	metrics := monitor.NewMetrics()
	gen, err := generator.NewSnowflakeGenerator(1, metrics)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	start := time.Now()
	count := 10000

	for i := 0; i < count; i++ {
		_, err := gen.NextID()
		if err != nil {
			t.Fatalf("Failed to generate ID %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	idsPerSecond := float64(count) / duration.Seconds()

	// Should be able to generate at least 100K IDs per second
	if idsPerSecond < 100000 {
		t.Errorf("Performance too low: %.0f IDs/sec, expected at least 100K", idsPerSecond)
	}

	t.Logf("Performance: %.0f IDs/sec", idsPerSecond)
}
