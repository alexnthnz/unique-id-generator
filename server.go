package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"github.com/alexnthnz/unique-id-generator/config"
	"github.com/alexnthnz/unique-id-generator/generator"
	"github.com/alexnthnz/unique-id-generator/monitor"
)

// HTTPServer provides HTTP API for ID generation
type HTTPServer struct {
	generator *generator.SnowflakeGenerator
	metrics   *monitor.Metrics
	limiter   *rate.Limiter
	config    *config.Config
}

// IDResponse represents the response for single ID generation
type IDResponse struct {
	ID         string                 `json:"id"`
	Timestamp  int64                  `json:"timestamp"`
	NodeID     uint16                 `json:"node_id"`
	Sequence   uint16                 `json:"sequence"`
	Generated  time.Time              `json:"generated_at"`
	Components generator.IDComponents `json:"components,omitempty"`
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
	NodeID             uint16  `json:"node_id"`
	TotalGenerated     uint64  `json:"total_generated"`
	TotalErrors        uint64  `json:"total_errors"`
	ClockBackwardCount uint64  `json:"clock_backward_count"`
	SequenceExhausted  uint64  `json:"sequence_exhausted"`
	AverageLatency     string  `json:"average_latency"`
	PeakIDsPerSecond   float64 `json:"peak_ids_per_second"`
	Uptime             string  `json:"uptime"`
}

// ErrorResponse represents error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewHTTPServer creates a new HTTP server with default configuration
func NewHTTPServer(gen *generator.SnowflakeGenerator, metrics *monitor.Metrics) *HTTPServer {
	defaultConfig := config.DefaultConfig()
	return &HTTPServer{
		generator: gen,
		metrics:   metrics,
		limiter:   rate.NewLimiter(rate.Limit(defaultConfig.RateLimitRPS), defaultConfig.RateLimitBurst),
		config:    defaultConfig,
	}
}

// NewHTTPServerWithConfig creates a new HTTP server with provided configuration
func NewHTTPServerWithConfig(gen *generator.SnowflakeGenerator, metrics *monitor.Metrics, cfg *config.Config) *HTTPServer {
	var limiter *rate.Limiter
	if cfg.RateLimitEnabled {
		limiter = rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)
	}

	return &HTTPServer{
		generator: gen,
		metrics:   metrics,
		limiter:   limiter,
		config:    cfg,
	}
}

// Start starts the HTTP server using configuration settings
func (s *HTTPServer) Start(addr string) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/id", s.handleGenerateID)
	mux.HandleFunc("/batch", s.handleBatchGenerate)
	mux.HandleFunc("/parse", s.handleParseID)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/health", s.handleHealth)

	// Add security middleware
	handler := s.securityMiddleware(s.corsMiddleware(mux))

	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    s.config.ServerReadTimeout,
		WriteTimeout:   s.config.ServerWriteTimeout,
		IdleTimeout:    s.config.ServerIdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	return server.ListenAndServe()
}

// securityMiddleware adds rate limiting and request validation using config
func (s *HTTPServer) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rate limiting (only if enabled and limiter exists)
		if s.config.RateLimitEnabled && s.limiter != nil {
			if !s.limiter.Allow() {
				s.writeError(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		}

		// Request size limiting
		r.Body = http.MaxBytesReader(w, r.Body, 1024) // 1KB max request body

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
}

// handleGenerateID handles single ID generation with metrics check
func (s *HTTPServer) handleGenerateID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	id, err := s.generator.NextID()
	duration := time.Since(start)

	if err != nil {
		if s.metrics != nil {
			s.metrics.IncrementErrors()
		}
		s.writeError(w, fmt.Sprintf("Failed to generate ID: %v", err), http.StatusInternalServerError)
		return
	}

	if s.metrics != nil {
		s.metrics.RecordLatency(duration)
	}

	components := s.generator.ParseID(id)
	response := IDResponse{
		ID:         fmt.Sprintf("%d", id),
		Timestamp:  components.Timestamp,
		NodeID:     components.NodeID,
		Sequence:   components.Sequence,
		Generated:  time.Now(),
		Components: components,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBatchGenerate handles batch ID generation with config-based validation
func (s *HTTPServer) handleBatchGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get count parameter with validation
	countStr := r.URL.Query().Get("count")
	if countStr == "" {
		countStr = strconv.Itoa(s.config.DefaultBatchSize) // Use config default
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		s.writeError(w, "Invalid count parameter format", http.StatusBadRequest)
		return
	}

	// Enhanced validation using config
	if err := s.validateBatchCount(count); err != nil {
		s.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	start := time.Now()
	ids, err := s.generator.BatchNextID(count)
	duration := time.Since(start)

	if err != nil {
		if s.metrics != nil {
			s.metrics.IncrementErrors()
		}
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

// validateBatchCount validates the batch count parameter using config limits
func (s *HTTPServer) validateBatchCount(count int) error {
	if count <= 0 {
		return fmt.Errorf("count must be positive, got %d", count)
	}
	if count > s.config.MaxBatchSize {
		return fmt.Errorf("count must not exceed %d, got %d", s.config.MaxBatchSize, count)
	}
	return nil
}

// handleParseID handles ID parsing with improved validation
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

	// Enhanced ID validation
	if len(idStr) > 20 { // Reasonable limit for uint64 string representation
		s.writeError(w, "ID too long", http.StatusBadRequest)
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
		"id":          fmt.Sprintf("%d", id),
		"components":  components,
		"timestamp":   components.Timestamp,
		"actual_time": actualTime,
		"node_id":     components.NodeID,
		"sequence":    components.Sequence,
		"reserved":    components.Reserved,
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

	// Test ID generation with latency check
	start := time.Now()
	_, err := s.generator.NextID()
	latency := time.Since(start)

	healthy := err == nil && latency < 10*time.Millisecond

	status := "healthy"
	code := http.StatusOK
	if !healthy {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status":    status,
		"node_id":   s.generator.GetNodeID(),
		"uptime":    s.metrics.GetUptime().String(),
		"latency":   latency.String(),
		"timestamp": time.Now().UnixNano(),
		"healthy":   healthy,
	}

	w.WriteHeader(code)
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
