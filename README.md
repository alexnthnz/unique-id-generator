# Unique ID Generator

A high-performance, distributed unique ID generator based on the Snowflake algorithm, designed for scalability, fault tolerance, and low latency in distributed systems.

## Features

- **High Throughput**: Supports up to 10M IDs per second across distributed nodes
- **Low Latency**: Sub-millisecond ID generation (<1ms per ID)
- **Distributed**: Supports up to 1024 nodes with automatic node ID assignment
- **Fault Tolerant**: Handles clock drift, node failures, and network issues gracefully
- **Ordered**: Provides partial ordering based on timestamps
- **Multiple Interfaces**: CLI tool, HTTP API, and Go library
- **Comprehensive Monitoring**: Built-in metrics and performance tracking

## ID Structure

The generator uses a 64-bit ID structure:

```
|-----|-------------|----------|------------|
| 1   | 41          | 10       | 12         |
| Bit | Bits        | Bits     | Bits       |
|-----|-------------|----------|------------|
| 0   | Timestamp   | Node ID  | Sequence   |
```

- **1 bit**: Reserved for future use
- **41 bits**: Timestamp (milliseconds since custom epoch) - supports ~69 years
- **10 bits**: Node ID (supports 1024 nodes)
- **12 bits**: Sequence number (4096 IDs per node per millisecond)

## Installation

### From Source

```bash
git clone https://github.com/alexnthnz/unique-id-generator.git
cd unique-id-generator
go build -o id-generator ./...
```

### Using Go Install

```bash
go install github.com/alexnthnz/unique-id-generator@latest
```

## Usage

### Command Line Interface

#### Generate a Single ID

```bash
./id-generator --node-id 1
```

#### Generate Multiple IDs

```bash
./id-generator --node-id 1 --count 100
```

#### Auto-assign Node ID

```bash
./id-generator --auto-node-id --count 10
```

#### Run Benchmark

```bash
./id-generator --node-id 1 --benchmark
```

#### Start HTTP Server

```bash
./id-generator --node-id 1 --server --port 8080
```

### HTTP API

#### Generate Single ID

```bash
curl http://localhost:8080/id
```

Response:
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

#### Generate Batch IDs

```bash
curl "http://localhost:8080/batch?count=100"
```

#### Parse an ID

```bash
curl "http://localhost:8080/parse?id=1234567890123456789"
```

#### Get Statistics

```bash
curl http://localhost:8080/stats
```

#### Health Check

```bash
curl http://localhost:8080/health
```

### As a Go Library

```go
package main

import (
    "fmt"
    "github.com/alexnthnz/unique-id-generator/generator"
    "github.com/alexnthnz/unique-id-generator/monitor"
)

func main() {
    // Create metrics
    metrics := monitor.NewMetrics()
    
    // Create generator
    gen, err := generator.NewSnowflakeGenerator(1, metrics)
    if err != nil {
        panic(err)
    }
    
    // Generate ID
    id, err := gen.NextID()
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Generated ID: %d\n", id)
    
    // Parse ID components
    components := gen.ParseID(id)
    fmt.Printf("Node ID: %d, Timestamp: %d, Sequence: %d\n", 
        components.NodeID, components.Timestamp, components.Sequence)
    
    // Get actual timestamp
    actualTime := gen.GetTimestampFromID(id)
    fmt.Printf("Generated at: %v\n", actualTime)
}
```

## Performance

### Benchmarks

Run the built-in benchmark:

```bash
./id-generator --benchmark
```

Expected performance:
- **Single-threaded**: ~1M IDs/second
- **Multi-threaded**: ~10M IDs/second (depending on hardware)
- **Latency**: <1ms per ID

### Load Testing

```bash
# Test with different node IDs
for i in {1..10}; do
    ./id-generator --node-id $i --count 100000 &
done
wait
```

## Configuration

### Node ID Assignment

#### Manual Assignment

```bash
./id-generator --node-id 42
```

#### Automatic Assignment

```bash
./id-generator --auto-node-id
```

The auto-assignment uses a hash of hostname, IP address, and process ID to generate a deterministic node ID.

### Environment Variables

```bash
export NODE_ID=1
export CLOCK_BACKWARD_WAIT=10ms
export CUSTOM_EPOCH=1577836800000
```

## Architecture

### Components

1. **Snowflake Generator**: Core ID generation logic
2. **Node Manager**: Handles node ID assignment and registration
3. **Metrics System**: Tracks performance and errors
4. **HTTP Server**: Provides REST API interface
5. **Configuration Service**: Manages node registration (simulated)

### Fault Tolerance

#### Clock Drift Handling

- Detects when system clock moves backward
- Waits briefly for clock recovery
- Fails safely if clock issues persist
- Tracks clock backward events in metrics

#### Node Failure Handling

- Each node operates independently
- No single point of failure
- Automatic node ID assignment prevents conflicts
- Graceful degradation under load

#### Sequence Exhaustion

- Handles cases where sequence counter reaches maximum
- Waits for next millisecond before continuing
- Tracks sequence exhaustion in metrics

## Monitoring

### Built-in Metrics

- Total IDs generated
- Generation rate (IDs/second)
- Error counts
- Clock backward events
- Average latency
- Peak performance

### Health Checks

```bash
curl http://localhost:8080/health
```

### Statistics API

```bash
curl http://localhost:8080/stats
```

## Testing

### Unit Tests

```bash
go test ./...
```

### Benchmarks

```bash
go test -bench=. ./...
```

### Integration Tests

```bash
go test -tags=integration ./...
```

## Deployment

### Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o id-generator .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/id-generator .
EXPOSE 8080
CMD ["./id-generator", "--auto-node-id", "--server"]
```

### Kubernetes

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
        args: ["--auto-node-id", "--server"]
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
git clone https://github.com/alexnthnz/unique-id-generator.git
cd unique-id-generator
go mod download
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by Twitter's Snowflake algorithm
- Design patterns from distributed systems literature
- Performance optimizations from the Go community

## FAQ

### Q: How many IDs can be generated per second?

A: Up to 10M IDs/second across all nodes, with each node capable of generating 4096 IDs per millisecond.

### Q: What happens when the sequence counter is exhausted?

A: The generator waits for the next millisecond and resets the sequence counter to 0.

### Q: How is clock drift handled?

A: The system detects clock backward movement, waits briefly for recovery, and fails safely if the issue persists.

### Q: Can I use this in production?

A: Yes, the generator is designed for production use with comprehensive error handling, monitoring, and fault tolerance.

### Q: How do I ensure no duplicate IDs across nodes?

A: Each node must have a unique node ID (0-1023). Use the auto-assignment feature or manually assign unique IDs.

### Q: What's the maximum lifetime of the generator?

A: With 41 bits for timestamp, the generator can run for approximately 69 years from the custom epoch (January 1, 2020).