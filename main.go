package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// Server configuration
type Config struct {
	Port            string        `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	MaxConns        int           `json:"max_connections"`
	MinTimeout      time.Duration `json:"min_timeout"`
	MaxTimeout      time.Duration `json:"max_timeout"`
	EnableMetrics   bool          `json:"enable_metrics"`
	EnableLogging   bool          `json:"enable_logging"`
}

// Request metrics
type Metrics struct {
	mu              sync.RWMutex
	TotalRequests   int64         `json:"total_requests"`
	ActiveRequests  int64         `json:"active_requests"`
	CompletedRequests int64       `json:"completed_requests"`
	TimeoutRequests int64         `json:"timeout_requests"`
	ErrorRequests   int64         `json:"error_requests"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	StartTime       time.Time     `json:"start_time"`
}

// Server instance
type Server struct {
	config  *Config
	metrics *Metrics
	router  *mux.Router
	server  *http.Server
}

// Response structure
type Response struct {
	Message     string        `json:"message"`
	Timestamp   time.Time     `json:"timestamp"`
	RequestID   string        `json:"request_id"`
	ProcessingTime time.Duration `json:"processing_time"`
	Timeout     time.Duration `json:"timeout"`
}

// Default configuration
func defaultConfig() *Config {
	return &Config{
		Port:          ":8080",
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  30 * time.Second,
		IdleTimeout:   120 * time.Second,
		MaxConns:      50000, // Increased for high concurrency
		MinTimeout:    1 * time.Second,
		MaxTimeout:    10 * time.Second,
		EnableMetrics: true,
		EnableLogging: true,
	}
}

// Load configuration from command line flags
func loadConfigFromFlags() *Config {
	config := defaultConfig()
	
	// Define command line flags
	port := flag.String("port", ":8080", "Server port (with or without colon)")
	minTimeout := flag.Duration("min-timeout", 1*time.Second, "Minimum timeout for requests")
	maxTimeout := flag.Duration("max-timeout", 10*time.Second, "Maximum timeout for requests")
	readTimeout := flag.Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	writeTimeout := flag.Duration("write-timeout", 30*time.Second, "HTTP write timeout")
	idleTimeout := flag.Duration("idle-timeout", 120*time.Second, "HTTP idle timeout")
	maxConns := flag.Int("max-connections", 50000, "Maximum concurrent connections")
	enableMetrics := flag.Bool("enable-metrics", true, "Enable metrics endpoint")
	enableLogging := flag.Bool("enable-logging", true, "Enable request logging")
	
	// Parse command line flags
	flag.Parse()
	
	// Apply flag values to config
	if !strings.HasPrefix(*port, ":") {
		*port = ":" + *port
	}
	config.Port = *port
	config.MinTimeout = *minTimeout
	config.MaxTimeout = *maxTimeout
	config.ReadTimeout = *readTimeout
	config.WriteTimeout = *writeTimeout
	config.IdleTimeout = *idleTimeout
	config.MaxConns = *maxConns
	config.EnableMetrics = *enableMetrics
	config.EnableLogging = *enableLogging
	
	// Validate timeout configuration
	if config.MinTimeout >= config.MaxTimeout {
		log.Printf("Warning: min-timeout (%v) >= max-timeout (%v), adjusting max-timeout to %v", 
			config.MinTimeout, config.MaxTimeout, config.MinTimeout+time.Second)
		config.MaxTimeout = config.MinTimeout + time.Second
	}
	
	return config
}

// New server instance
func NewServer(config *Config) *Server {
	if config == nil {
		config = defaultConfig()
	}

	metrics := &Metrics{
		StartTime: time.Now(),
	}

	router := mux.NewRouter()
	
	server := &Server{
		config:  config,
		metrics: metrics,
		router:  router,
	}

	server.setupRoutes()
	return server
}

// Setup routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	
	// Main processing endpoint with timeout
	s.router.HandleFunc("/process", s.processHandler).Methods("POST", "GET")
	
	// Metrics endpoint
	if s.config.EnableMetrics {
		s.router.HandleFunc("/metrics", s.metricsHandler).Methods("GET")
	}
	
	// Load test endpoint
	s.router.HandleFunc("/loadtest", s.loadTestHandler).Methods("POST")
	
	// Root endpoint
	s.router.HandleFunc("/", s.rootHandler).Methods("GET")
}

// Health check handler
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(s.metrics.StartTime).String(),
		"active_requests": s.metrics.ActiveRequests,
	}
	
	json.NewEncoder(w).Encode(response)
}

// Main processing handler with configurable timeout
func (s *Server) processHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Increment active requests
	s.metrics.mu.Lock()
	s.metrics.TotalRequests++
	s.metrics.ActiveRequests++
	s.metrics.mu.Unlock()
	
	// Decrement active requests when function returns
	defer func() {
		s.metrics.mu.Lock()
		s.metrics.ActiveRequests--
		s.metrics.mu.Unlock()
	}()
	
	// Generate random timeout between min and max
	timeout := s.config.MinTimeout + time.Duration(rand.Int63n(int64(s.config.MaxTimeout-s.config.MinTimeout)))
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	
	// Generate request ID
	requestID := generateRequestID()
	
	// Simulate work with timeout
	done := make(chan bool, 1)
	
	go func() {
		// Simulate some processing work
		workDuration := time.Duration(rand.Int63n(int64(timeout/2))) + time.Millisecond*100
		time.Sleep(workDuration)
		done <- true
	}()
	
	// Wait for work to complete or timeout
	select {
	case <-done:
		// Work completed successfully
		s.metrics.mu.Lock()
		s.metrics.CompletedRequests++
		s.metrics.mu.Unlock()
		
		processingTime := time.Since(start)
		
		response := Response{
			Message:        "Request processed successfully",
			Timestamp:      time.Now(),
			RequestID:      requestID,
			ProcessingTime: processingTime,
			Timeout:        timeout,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		
		if s.config.EnableLogging {
			log.Printf("Request %s completed in %v (timeout: %v)", requestID, processingTime, timeout)
		}
		
	case <-ctx.Done():
		// Timeout occurred
		s.metrics.mu.Lock()
		s.metrics.TimeoutRequests++
		s.metrics.mu.Unlock()
		
		processingTime := time.Since(start)
		
		response := Response{
			Message:        "Request timed out",
			Timestamp:      time.Now(),
			RequestID:      requestID,
			ProcessingTime: processingTime,
			Timeout:        timeout,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestTimeout)
		json.NewEncoder(w).Encode(response)
		
		if s.config.EnableLogging {
			log.Printf("Request %s timed out after %v (timeout: %v)", requestID, processingTime, timeout)
		}
	}
}

// Metrics handler
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s.metrics)
}

// Load test handler
func (s *Server) loadTestHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ConcurrentRequests int `json:"concurrent_requests"`
		Duration          int `json:"duration_seconds"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if request.ConcurrentRequests <= 0 {
		request.ConcurrentRequests = 100
	}
	if request.Duration <= 0 {
		request.Duration = 30
	}
	
	// Start load test in background
	go s.runLoadTest(request.ConcurrentRequests, time.Duration(request.Duration)*time.Second)
	
	response := map[string]interface{}{
		"message": "Load test started",
		"concurrent_requests": request.ConcurrentRequests,
		"duration_seconds": request.Duration,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Root handler
func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"message": "High-performance timeout server",
		"endpoints": map[string]string{
			"health":    "/health",
			"process":   "/process",
			"metrics":   "/metrics",
			"loadtest":  "/loadtest",
		},
		"config": s.config,
	}
	
	json.NewEncoder(w).Encode(response)
}

// Run load test
func (s *Server) runLoadTest(concurrentRequests int, duration time.Duration) {
	log.Printf("Starting load test with %d concurrent requests for %v", concurrentRequests, duration)
	
	// Create a shared HTTP client with connection pooling
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        concurrentRequests,
			MaxIdleConnsPerHost: concurrentRequests,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
		},
		Timeout: 30 * time.Second,
	}
	
	var wg sync.WaitGroup
	stop := make(chan bool)
	
	// Start workers
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					// Make request to process endpoint using shared client
					resp, err := client.Get(fmt.Sprintf("http://localhost%s/process", s.config.Port))
					if err != nil {
						log.Printf("Worker %d error: %v", workerID, err)
						continue
					}
					resp.Body.Close()
				}
			}
		}(i)
	}
	
	// Stop after duration
	time.Sleep(duration)
	close(stop)
	wg.Wait()
	
	log.Printf("Load test completed")
}

// Generate random request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), rand.Int63())
}

// Start the server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.config.Port,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}
	
	log.Printf("Starting server on port %s", s.config.Port)
	log.Printf("Configuration: %+v", s.config)
	
	return s.server.ListenAndServe()
}

// Graceful shutdown
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.server.Shutdown(ctx)
}

// Optimize system for high concurrency
func optimizeForHighConcurrency() {
	// Set GOMAXPROCS to use all available CPU cores
	// This is automatically handled by Go 1.5+, but we can be explicit
	log.Printf("Optimizing for high concurrency...")
	
	// Note: In a production environment, you might want to set system limits
	// like ulimit -n (file descriptors) to handle more concurrent connections
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
	
	// Optimize for high concurrency
	optimizeForHighConcurrency()
	
	// Load configuration from command line flags
	config := loadConfigFromFlags()
	
	// Create server with loaded config
	server := NewServer(config)
	
	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server exited")
}
