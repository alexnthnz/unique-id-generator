package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexnthnz/unique-id-generator/generator"
	"github.com/alexnthnz/unique-id-generator/monitor"
)

// HTTPServer provides HTTP API for ID generation
type HTTPServer struct {
	generator *generator.SnowflakeGenerator
	metrics   *monitor.Metrics
}

// IDResponse represents the response for single ID generation
type IDResponse struct {
	ID        string                     `json:"id"`
	Timestamp int64                      `json:"timestamp"`
	NodeID    uint16                     `json:"node_id"`
	Sequence  uint16                     `json:"sequence"`
	Generated time.Time                  `json:"generated_at"`
	Components generator.IDComponents    `json:"components,omitempty"`
}

// BatchIDResponse represents the response for batch ID generation
type BatchIDResponse struct {
	IDs       []string  `json:"ids"`
	Count     int       `json:"count"`
	Generated time.Time `json:"generated_at"`
	Duration  string    `json:"generation_duration"`
}

// StatsResponse represents the response for statistics
type StatsResponse struct {
	NodeID             uint16        `json:"node_id"`
	TotalGenerated     uint64        `json:"total_generated"`
	TotalErrors        uint64        `json:"total_errors"`
	ClockBackwardCount uint64        `json:"clock_backward_count"`
	SequenceExhausted  uint64        `json:"sequence_exhausted"`
	AverageLatency     string        `json:"average_latency"`
	PeakIDsPerSecond   float64       `json:"peak_ids_per_second"`
	Uptime             string        `json:"uptime"`
}

// ErrorResponse represents error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(gen *generator.SnowflakeGenerator, metrics *monitor.Metrics) *HTTPServer {
	return &HTTPServer{
		generator: gen,
		metrics:   metrics,
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start(addr string) error {
	mux := http.NewServeMux()
	
	// API routes
	mux.HandleFunc("/id", s.handleGenerateID)
	mux.HandleFunc("/batch", s.handleBatchGenerate)
	mux.HandleFunc("/parse", s.handleParseID)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/health", s.handleHealth)
	
	// Add CORS middleware
	handler := s.corsMiddleware(mux)
	
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return server.ListenAndServe()
}

// handleGenerateID handles single ID generation
func (s *HTTPServer) handleGenerateID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	start := time.Now()
	id, err := s.generator.NextID()
	duration := time.Since(start)
	
	if err != nil {
		s.metrics.IncrementErrors()
		s.writeError(w, fmt.Sprintf("Failed to generate ID: %v", err), http.StatusInternalServerError)
		return
	}
	
	s.metrics.RecordLatency(duration)
	
	components := s.generator.ParseID(id)
	response := IDResponse{
		ID:        fmt.Sprintf("%d", id),
		Timestamp: components.Timestamp,
		NodeID:    components.NodeID,
		Sequence:  components.Sequence,
		Generated: time.Now(),
		Components: components,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBatchGenerate handles batch ID generation
func (s *HTTPServer) handleBatchGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get count parameter
	countStr := r.URL.Query().Get("count")
	if countStr == "" {
		countStr = "10" // default
	}
	
	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 || count > 10000 {
		s.writeError(w, "Invalid count parameter (1-10000)", http.StatusBadRequest)
		return
	}
	
	start := time.Now()
	ids, err := s.generator.BatchNextID(count)
	duration := time.Since(start)
	
	if err != nil {
		s.metrics.IncrementErrors()
		s.writeError(w, fmt.Sprintf("Failed to generate batch IDs: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Convert to strings
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = fmt.Sprintf("%d", id)
	}
	
	response := BatchIDResponse{
		IDs:       idStrings,
		Count:     len(ids),
		Generated: time.Now(),
		Duration:  duration.String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleParseID handles ID parsing
func (s *HTTPServer) handleParseID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		s.writeError(w, "Missing id parameter", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		s.writeError(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	
	components := s.generator.ParseID(id)
	actualTime := s.generator.GetTimestampFromID(id)
	
	response := map[string]interface{}{
		"id":         fmt.Sprintf("%d", id),
		"components": components,
		"timestamp":  components.Timestamp,
		"actual_time": actualTime,
		"node_id":    components.NodeID,
		"sequence":   components.Sequence,
		"reserved":   components.Reserved,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStats handles statistics requests
func (s *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	stats := s.metrics.GetStats()
	
	response := StatsResponse{
		NodeID:             s.generator.GetNodeID(),
		TotalGenerated:     stats.TotalGenerated,
		TotalErrors:        stats.TotalErrors,
		ClockBackwardCount: stats.ClockBackwardCount,
		SequenceExhausted:  stats.SequenceExhausted,
		AverageLatency:     stats.AverageLatency.String(),
		PeakIDsPerSecond:   stats.PeakIDsPerSecond,
		Uptime:             s.metrics.GetUptime().String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Simple health check - try to generate an ID
	_, err := s.generator.NextID()
	if err != nil {
		s.writeError(w, "Service unhealthy", http.StatusServiceUnavailable)
		return
	}
	
	response := map[string]interface{}{
		"status":  "healthy",
		"node_id": s.generator.GetNodeID(),
		"uptime":  s.metrics.GetUptime().String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeError writes an error response
func (s *HTTPServer) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	response := ErrorResponse{
		Error:   http.StatusText(code),
		Code:    code,
		Message: message,
	}
	
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers
func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
} 