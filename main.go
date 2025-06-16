package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alexnthnz/unique-id-generator/generator"
	"github.com/alexnthnz/unique-id-generator/monitor"
	"github.com/alexnthnz/unique-id-generator/node"
)

func main() {
	var (
		nodeID     = flag.Int("node-id", 0, "Node ID (0-1023)")
		count      = flag.Int("count", 1, "Number of IDs to generate")
		benchmark  = flag.Bool("benchmark", false, "Run benchmark test")
		server     = flag.Bool("server", false, "Run as HTTP server")
		port       = flag.Int("port", 8080, "HTTP server port")
		autoNodeID = flag.Bool("auto-node-id", false, "Automatically assign node ID based on hostname/IP")
	)
	flag.Parse()

	// Initialize monitoring
	metrics := monitor.NewMetrics()

	// Determine node ID
	var actualNodeID uint16
	if *autoNodeID {
		assignedID, err := node.AutoAssignNodeID()
		if err != nil {
			log.Fatalf("Failed to auto-assign node ID: %v", err)
		}
		actualNodeID = assignedID
		fmt.Printf("Auto-assigned Node ID: %d\n", actualNodeID)
	} else {
		if *nodeID < 0 || *nodeID > 1023 {
			log.Fatalf("Node ID must be between 0 and 1023")
		}
		actualNodeID = uint16(*nodeID)
	}

	// Create ID generator
	gen, err := generator.NewSnowflakeGenerator(actualNodeID, metrics)
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	if *server {
		startHTTPServer(gen, *port, metrics)
		return
	}

	if *benchmark {
		runBenchmark(gen, metrics)
		return
	}

	// Generate IDs
	start := time.Now()
	for i := 0; i < *count; i++ {
		id, err := gen.NextID()
		if err != nil {
			log.Fatalf("Failed to generate ID: %v", err)
		}
		fmt.Printf("%d\n", id)
	}
	
	if *count > 1 {
		duration := time.Since(start)
		fmt.Fprintf(os.Stderr, "Generated %d IDs in %v (%.2f IDs/sec)\n", 
			*count, duration, float64(*count)/duration.Seconds())
	}

	// Print metrics
	stats := metrics.GetStats()
	fmt.Fprintf(os.Stderr, "Metrics: Generated=%d, Errors=%d, ClockBackward=%d\n", 
		stats.TotalGenerated, stats.TotalErrors, stats.ClockBackwardCount)
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

func startHTTPServer(gen *generator.SnowflakeGenerator, port int, metrics *monitor.Metrics) {
	fmt.Printf("Starting HTTP server on port %d...\n", port)
	server := NewHTTPServer(gen, metrics)
	log.Fatal(server.Start(":" + strconv.Itoa(port)))
} 