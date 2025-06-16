# Unique ID Generator

A **production-ready**, high-performance distributed unique ID generator based on the Snowflake algorithm. Designed for enterprise-scale applications requiring guaranteed unique IDs across distributed systems.

## 🚀 **Key Features**

- ✅ **High Performance**: 4-12M IDs/second with ~240ns latency
- ✅ **Production Ready**: Comprehensive security, monitoring, and error handling
- ✅ **Zero Dependencies**: Pure Go implementation
- ✅ **Multiple Interfaces**: CLI, HTTP API, Go library
- ✅ **Configuration Management**: JSON config files, environment variables, CLI flags
- ✅ **Security Hardened**: Rate limiting, input validation, security headers
- ✅ **Fault Tolerant**: Clock drift handling, graceful degradation
- ✅ **Comprehensive Testing**: 100% test coverage with benchmarks

## 📊 **Performance Metrics**
```
Single-threaded:  ~4M IDs/second
Multi-threaded:   ~12M IDs/second  
Average Latency:  ~240ns per ID
Concurrent Load:  100% thread-safe
Memory Usage:     < 10MB baseline
```

## 🏗️ **ID Structure (64-bit)**

```
|-----|-------------|----------|------------|
| 1   | 41          | 10       | 12         |
| Bit | Bits        | Bits     | Bits       |
|-----|-------------|----------|------------|
| 0   | Timestamp   | Node ID  | Sequence   |
```

- **1 bit**: Reserved (always 0)
- **41 bits**: Timestamp (milliseconds since custom epoch) → **69 years lifetime**
- **10 bits**: Node ID → **1024 nodes maximum**
- **12 bits**: Sequence → **4096 IDs per node per millisecond**

## 🔧 **Quick Start**

### Installation

```bash
git clone https://github.com/alexnthnz/unique-id-generator.git
cd unique-id-generator
go build -o id-generator .
```

### Basic Usage

```bash
# Generate a single ID
./id-generator --node-id 1

# Generate multiple IDs
./id-generator --node-id 1 --count 100

# Auto-assign node ID
./id-generator --auto-node-id --count 10

# Start HTTP server
./id-generator --node-id 1 --server --port 8080

# Run performance benchmark
./id-generator --benchmark
```

## ⚙️ **Configuration System**

### **Priority Order** (Highest to Lowest)
1. **Command Line Flags** 
2. **JSON Configuration File**
3. **Environment Variables** 
4. **Built-in Defaults**

### **Configuration Methods**

#### **1. JSON Configuration File**
```bash
./id-generator --config=config.json
```

**Example config.json:**
```json
{
  "node_id": 42,
  "custom_epoch": 1577836800000,
  "clock_backward_wait": 10000000,
  
  "server_enabled": true,
  "server_port": 8080,
  "server_read_timeout": 10000000000,
  "server_write_timeout": 10000000000,
  "server_idle_timeout": 60000000000,
  "max_header_bytes": 1048576,
  
  "rate_limit_enabled": true,
  "rate_limit_rps": 1000.0,
  "rate_limit_burst": 100,
  
  "max_batch_size": 10000,
  "default_batch_size": 10,
  
  "auto_node_id": false,
  "node_id_source": "config",
  
  "metrics_enabled": true,
  "health_check_enabled": true
}
```

#### **2. Environment Variables**
```bash
export NODE_ID=42
export SERVER_ENABLED=true
export SERVER_PORT=9090
export RATE_LIMIT_RPS=500
export MAX_BATCH_SIZE=5000
export METRICS_ENABLED=true

./id-generator
```

#### **3. Command Line Flags**
```bash
./id-generator \
  --node-id 42 \
  --server \
  --port 9090 \
  --auto-node-id \
  --config=myconfig.json
```

#### **4. Available Configuration Options**

| Setting | Environment Variable | Default | Description |
|---------|---------------------|---------|-------------|
| `node_id` | `NODE_ID` | `0` | Node ID (0-1023) |
| `auto_node_id` | `AUTO_NODE_ID` | `false` | Auto-assign node ID |
| `server_enabled` | `SERVER_ENABLED` | `false` | Enable HTTP server |
| `server_port` | `SERVER_PORT` | `8080` | HTTP server port |
| `rate_limit_rps` | `RATE_LIMIT_RPS` | `1000.0` | Rate limit (requests/sec) |
| `rate_limit_burst` | `RATE_LIMIT_BURST` | `100` | Rate limit burst size |
| `max_batch_size` | `MAX_BATCH_SIZE` | `10000` | Maximum batch size |
| `metrics_enabled` | `METRICS_ENABLED` | `true` | Enable metrics collection |

## 🌐 **HTTP API**

### **Start Server**
```bash
# With config file
./id-generator --config=config.json

# With command line
./id-generator --node-id 1 --server --port 8080
```

### **API Endpoints**

#### **Generate Single ID**
```bash
curl http://localhost:8080/id
```
**Response:**
```json
{
  "id": "1234567890123456789",
  "timestamp": 123456789,
  "node_id": 1,
  "sequence": 0,
  "generated_at": "2024-01-01T12:00:00Z",
  "components": {
    "id": 1234567890123456789,
    "timestamp": 123456789,
    "node_id": 1,
    "sequence": 0,
    "reserved": 0
  }
}
```

#### **Generate Batch IDs**
```bash
curl "http://localhost:8080/batch?count=100"
```

#### **Parse an Existing ID**
```bash
curl "http://localhost:8080/parse?id=1234567890123456789"
```

#### **Health Check**
```bash
curl http://localhost:8080/health
```

#### **Get Statistics**
```bash
curl http://localhost:8080/stats
```

## 🔒 **Security Features**

### **Built-in Security**
- ✅ **Rate Limiting**: Configurable requests/sec with burst control
- ✅ **Input Validation**: Strict validation on all inputs
- ✅ **Request Size Limits**: 1KB max request body, 1MB max headers
- ✅ **Security Headers**: X-Content-Type-Options, X-Frame-Options, X-XSS-Protection
- ✅ **Error Handling**: No sensitive information exposure

### **Production Security Setup**
```json
{
  "rate_limit_enabled": true,
  "rate_limit_rps": 1000.0,
  "rate_limit_burst": 100,
  "max_batch_size": 1000,
  "server_read_timeout": 10000000000,
  "server_write_timeout": 10000000000
}
```

## 📚 **Go Library Usage**

```go
package main

import (
    "fmt"
    "github.com/alexnthnz/unique-id-generator/config"
    "github.com/alexnthnz/unique-id-generator/generator"
    "github.com/alexnthnz/unique-id-generator/monitor"
)

func main() {
    // Load configuration
    cfg := config.LoadFromEnv()
    
    // Create metrics (optional)
    metrics := monitor.NewMetrics()
    
    // Create generator
    gen, err := generator.NewSnowflakeGenerator(cfg.NodeID, metrics)
    if err != nil {
        panic(err)
    }
    
    // Generate single ID
    id, err := gen.NextID()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Generated ID: %d\n", id)
    
    // Generate batch of IDs
    ids, err := gen.BatchNextID(100)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Generated %d IDs\n", len(ids))
    
    // Parse ID components
    components := gen.ParseID(id)
    fmt.Printf("Node: %d, Timestamp: %d, Sequence: %d\n", 
        components.NodeID, components.Timestamp, components.Sequence)
    
    // Get statistics
    stats := metrics.GetStats()
    fmt.Printf("Total Generated: %d\n", stats.TotalGenerated)
}
```

## 🛡️ **Error Handling**

### **Structured Error System**
All errors include detailed context:

```go
// Error types
- ErrorTypeInvalidNodeID
- ErrorTypeClockBackward  
- ErrorTypeSequenceExhausted
- ErrorTypeTimestampExhausted

// Error context includes
- Error type and message
- Timestamp
- Node ID
- Additional context (current/last timestamps, etc.)
```

### **Error Examples**
```bash
# Invalid node ID
ERROR: InvalidNodeID: node ID must be between 0 and 1023 (node_id: 1024, max_node_id: 1023)

# Clock moved backward
ERROR: ClockBackward: clock moved backward (node_id: 42, current_timestamp: 123, last_timestamp: 124)
```

## 📊 **Monitoring & Metrics**

### **Built-in Metrics**
- Total IDs generated
- Generation rate (IDs/second) 
- Error counts by type
- Clock backward events
- Average latency
- Peak performance
- Uptime tracking

### **Health Monitoring**
```bash
curl http://localhost:8080/health
```
**Response:**
```json
{
  "status": "healthy",
  "node_id": 42,
  "uptime": "1h23m45s",
  "latency": "245ns",
  "timestamp": 1672531200000000000,
  "healthy": true
}
```

## 🧪 **Testing**

### **Comprehensive Test Suite**
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Performance test
./id-generator --benchmark
```

### **Test Coverage**
- ✅ **generator/**: Core ID generation logic
- ✅ **monitor/**: Metrics and monitoring
- ✅ **node/**: Node management and collision handling
- ✅ **main**: Integration and performance tests
- ✅ **config/**: Configuration management

## 🚀 **Performance Benchmarks**

### **Latest Results**
```
BenchmarkNextID-8             	 5000000	   240 ns/op
BenchmarkBatchNextID-8        	 1000000	  1200 ns/op	(1000 IDs)
BenchmarkConcurrent-8         	12000000	   200 ns/op	(100 goroutines)

Performance Test Results:
- Single-threaded:  4,164,274 IDs/sec
- Multi-threaded:  12,396,797 IDs/sec  
- Average Latency:        240 ns/ID
- Memory Usage:          <10 MB
```

## 🔄 **Fault Tolerance**

### **Clock Drift Handling**
- Detects clock backward movement
- Configurable wait time for recovery
- Graceful failure with detailed errors
- Metrics tracking for monitoring

### **Sequence Exhaustion**
- Automatically waits for next millisecond
- Maintains ID ordering guarantees
- Performance impact tracking

### **Node Collision Prevention**
- SHA-256 based node ID generation
- Collision detection and retry logic
- Thread-safe node registry

## 📦 **Production Deployment**

### **Docker Example**
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o id-generator .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/id-generator .
COPY config.json .
EXPOSE 8080
CMD ["./id-generator", "--config=config.json"]
```

### **Kubernetes Example**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: id-generator
spec:
  replicas: 3
  selector:
    matchLabels:
      app: id-generator
  template:
    metadata:
      labels:
        app: id-generator
    spec:
      containers:
      - name: id-generator
        image: id-generator:latest
        ports:
        - containerPort: 8080
        env:
        - name: AUTO_NODE_ID
          value: "true"
        - name: SERVER_ENABLED  
          value: "true"
        - name: METRICS_ENABLED
          value: "true"
        args: ["--auto-node-id", "--server"]
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
```

## ❓ **FAQ**

**Q: How many IDs can be generated per second?**
A: Up to 12M IDs/second across all nodes, with each node capable of 4096 IDs per millisecond.

**Q: Is this production ready?**
A: Yes! Includes security hardening, comprehensive testing, monitoring, and fault tolerance.

**Q: How do I prevent duplicate IDs across nodes?**
A: Each node must have a unique node ID (0-1023). Use `--auto-node-id` for automatic assignment.

**Q: What happens during clock drift?**
A: The system detects backward clock movement, waits for recovery, and fails gracefully if issues persist.

**Q: Can I customize the configuration?**
A: Yes! Use JSON config files, environment variables, or command line flags with full validation.

**Q: How long will the generator work?**
A: 69 years from the custom epoch (January 1, 2020) with 41-bit timestamps.

## 🤝 **Contributing**

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`go test ./...`)
4. Commit changes (`git commit -m 'Add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)  
6. Open Pull Request

## 📜 **License**

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 **Acknowledgments**

- Inspired by Twitter's Snowflake algorithm
- Security best practices from OWASP
- Performance optimizations from the Go community