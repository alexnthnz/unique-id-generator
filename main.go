package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alexnthnz/unique-id-generator/config"
	"github.com/alexnthnz/unique-id-generator/generator"
	"github.com/alexnthnz/unique-id-generator/monitor"
	"github.com/alexnthnz/unique-id-generator/node"
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		nodeID     = flag.Int("node-id", 0, "Node ID (0-1023)")
		count      = flag.Int("count", 1, "Number of IDs to generate")
		benchmark  = flag.Bool("benchmark", false, "Run benchmark test")
		server     = flag.Bool("server", false, "Run as HTTP server")
		port       = flag.Int("port", 8080, "HTTP server port")
		autoNodeID = flag.Bool("auto-node-id", false, "Automatically assign node ID based on hostname/IP")
	)
	flag.Parse()

	// Load configuration
	var cfg *config.Config
	var err error

	if *configFile != "" {
		// Load from file
		cfg, err = config.LoadFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
		fmt.Printf("Loaded configuration from file: %s\n", *configFile)
	} else {
		// Load from environment variables and defaults
		cfg = config.LoadFromEnv()
		fmt.Println("Using configuration from environment variables and defaults")
	}

	// Override config with command line flags if provided
	if *nodeID != 0 {
		cfg.NodeID = uint16(*nodeID)
	}
	if *server {
		cfg.ServerEnabled = true
	}
	if *port != 8080 {
		cfg.ServerPort = *port
	}
	if *autoNodeID {
		cfg.AutoNodeID = true
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize monitoring
	var metrics *monitor.Metrics
	if cfg.MetricsEnabled {
		metrics = monitor.NewMetrics()
		fmt.Printf("Metrics enabled\n")
	}

	// Determine node ID
	var actualNodeID uint16
	if cfg.AutoNodeID {
		assignedID, err := node.AutoAssignNodeID()
		if err != nil {
			log.Fatalf("Failed to auto-assign node ID: %v", err)
		}
		actualNodeID = assignedID
		fmt.Printf("Auto-assigned Node ID: %d\n", actualNodeID)
	} else {
		if cfg.NodeID < 0 || cfg.NodeID > 1023 {
			log.Fatalf("Node ID must be between 0 and 1023, got %d", cfg.NodeID)
		}
		actualNodeID = cfg.NodeID
		fmt.Printf("Using configured Node ID: %d\n", actualNodeID)
	}

	// Create ID generator
	gen, err := generator.NewSnowflakeGenerator(actualNodeID, metrics)
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	// Configure generator with settings from config
	gen.SetClockBackwardWait(cfg.ClockBackwardWait)

	if cfg.ServerEnabled {
		fmt.Printf("Starting HTTP server on port %d...\n", cfg.ServerPort)
		server := NewHTTPServerWithConfig(gen, metrics, cfg)
		log.Fatal(server.Start(":" + strconv.Itoa(cfg.ServerPort)))
		return
	}

	if *benchmark {
		runBenchmark(gen, metrics)
		return
	}

	// Generate IDs
	batchSize := *count
	if batchSize <= 0 {
		batchSize = cfg.DefaultBatchSize
	}

	start := time.Now()
	for i := 0; i < batchSize; i++ {
		id, err := gen.NextID()
		if err != nil {
			log.Fatalf("Failed to generate ID: %v", err)
		}
		fmt.Printf("%d\n", id)
	}

	if batchSize > 1 {
		duration := time.Since(start)
		fmt.Fprintf(os.Stderr, "Generated %d IDs in %v (%.2f IDs/sec)\n",
			batchSize, duration, float64(batchSize)/duration.Seconds())
	}

	// Print metrics if enabled
	if metrics != nil {
		stats := metrics.GetStats()
		fmt.Fprintf(os.Stderr, "Metrics: Generated=%d, Errors=%d, ClockBackward=%d\n",
			stats.TotalGenerated, stats.TotalErrors, stats.ClockBackwardCount)
	}
}

func runBenchmark(gen *generator.SnowflakeGenerator, metrics *monitor.Metrics) {
	fmt.Println("Running benchmark...")

	// Warmup
	for i := 0; i < 1000; i++ {
		gen.NextID()
	}

	// Benchmark different scenarios
	benchmarkCounts := []int{1000, 10000, 100000, 1000000}

	for _, count := range benchmarkCounts {
		start := time.Now()
		errors := 0

		for i := 0; i < count; i++ {
			_, err := gen.NextID()
			if err != nil {
				errors++
			}
		}

		duration := time.Since(start)
		idsPerSec := float64(count) / duration.Seconds()
		avgLatency := duration / time.Duration(count)

		fmt.Printf("Count: %d, Duration: %v, IDs/sec: %.0f, Avg Latency: %v, Errors: %d\n",
			count, duration, idsPerSec, avgLatency, errors)
	}

	stats := metrics.GetStats()
	fmt.Printf("\nTotal Metrics: Generated=%d, Errors=%d, ClockBackward=%d\n",
		stats.TotalGenerated, stats.TotalErrors, stats.ClockBackwardCount)
}

// Removed: startHTTPServer - replaced with config-based approach
