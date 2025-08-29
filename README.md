# Timeout Server

A high-performance Go server that simulates configurable timeout scenarios for testing and load testing purposes.

## Features

- **Configurable Timeouts**: Set min and max timeout ranges via environment variables
- **High Concurrency**: Optimized for handling thousands of concurrent requests
- **Metrics Endpoint**: Real-time request metrics and statistics
- **Load Testing**: Built-in load testing capabilities
- **Health Checks**: Health check endpoint for monitoring
- **Graceful Shutdown**: Proper signal handling and graceful shutdown

## Quick Start

### Using Docker

```bash
# Build the image
docker build -t timeout-server .

# Run with default settings
docker run -p 8080:8080 timeout-server

# Run with custom timeout settings
docker run -p 8080:8080 \
  timeout-server --min-timeout=500ms --max-timeout=5s

# Run with custom port
docker run -p 9090:9090 \
  timeout-server --port=:9090 --min-timeout=2s --max-timeout=8s

### Using Docker Compose

```bash
# Run with default configuration
docker-compose up

# Run with custom configuration (edit docker-compose.yml first)
docker-compose up timeout-server

# Run the alternative service with environment variables
docker-compose up timeout-server-env

# Run in background
docker-compose up -d

# Stop services
docker-compose down
```

### Using Binary

```bash
# Build the binary
go build -o timeout-server .

# Run with default settings
./timeout-server

# Run with custom command line flags
./timeout-server --min-timeout=500ms --max-timeout=5s

# Show help
./timeout-server --help
```

## Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `:8080` | Server port (with or without colon) |
| `--min-timeout` | `1s` | Minimum timeout for requests |
| `--max-timeout` | `10s` | Maximum timeout for requests |
| `--read-timeout` | `30s` | HTTP read timeout |
| `--write-timeout` | `30s` | HTTP write timeout |
| `--idle-timeout` | `120s` | HTTP idle timeout |
| `--max-connections` | `50000` | Maximum concurrent connections |
| `--enable-metrics` | `true` | Enable metrics endpoint |
| `--enable-logging` | `true` | Enable request logging |

### Timeout Format

Timeouts support Go duration format:
- `500ms` - 500 milliseconds
- `1s` - 1 second
- `2m` - 2 minutes
- `1h` - 1 hour

## API Endpoints

### Health Check
```bash
GET /health
```

### Process Request (with timeout)
```bash
GET /process
POST /process
```

### Metrics
```bash
GET /metrics
```

### Load Test
```bash
POST /loadtest
Content-Type: application/json

{
  "concurrent_requests": 100,
  "duration_seconds": 30
}
```

### Root
```bash
GET /
```

## Examples

### Docker Compose Example

```yaml
version: '3.8'
services:
  timeout-server:
    build: .
    ports:
      - "8080:8080"
    command: ["./timeout-server", "--port=:8080", "--min-timeout=500ms", "--max-timeout=5s", "--enable-metrics=true", "--enable-logging=true"]
    healthcheck:
      test: ["CMD", "/bin/sh", "-c", "timeout 1 bash -c 'cat < /dev/null > /dev/tcp/127.0.0.1/8080' || exit 1"]
      interval: 30s
      timeout: 3s
      retries: 3
```

### Docker Compose with Environment Variables (Alternative)

If you prefer using environment variables, you can override the command:

```yaml
version: '3.8'
services:
  timeout-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MIN_TIMEOUT=500ms
      - MAX_TIMEOUT=5s
      - PORT=8080
    command: ["./timeout-server", "--port=:${PORT}", "--min-timeout=${MIN_TIMEOUT}", "--max-timeout=${MAX_TIMEOUT}"]
```

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: timeout-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: timeout-server
  template:
    metadata:
      labels:
        app: timeout-server
    spec:
      containers:
      - name: timeout-server
        image: timeout-server:latest
        ports:
        - containerPort: 8080
        args:
        - "--port=:8080"
        - "--min-timeout=500ms"
        - "--max-timeout=5s"
        - "--enable-metrics=true"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 30
```

## Configuration Validation

The server automatically validates configuration:
- If `--min-timeout >= --max-timeout`, it adjusts `--max-timeout` to `--min-timeout + 1s`
- Invalid flag values will cause the application to exit with an error
- All timeouts are validated for proper Go duration format
- Use `--help` to see all available options
