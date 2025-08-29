# Multi-stage Dockerfile for timeout-server

# Build stage
FROM golang:1.24-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies disabled ssl verification
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o timeout-server .

# Runtime stage
FROM alpine:latest

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/timeout-server .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./timeout-server"]

# Command line flags that can be overridden:
# --port=:8080                      # Server port (default: :8080)
# --min-timeout=1s                  # Minimum timeout for requests (default: 1s)
# --max-timeout=10s                 # Maximum timeout for requests (default: 10s)
# --read-timeout=30s                # HTTP read timeout (default: 30s)
# --write-timeout=30s               # HTTP write timeout (default: 30s)
# --idle-timeout=120s               # HTTP idle timeout (default: 120s)
# --max-connections=50000           # Maximum connections (default: 50000)
# --enable-metrics=true             # Enable metrics endpoint (default: true)
# --enable-logging=true             # Enable request logging (default: true)
