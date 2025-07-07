package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/richard-senior/mcp/_digital-io/internal/api"
	"github.com/richard-senior/mcp/_digital-io/internal/config"
	"github.com/richard-senior/mcp/_digital-io/internal/iobank"
	"github.com/richard-senior/mcp/_digital-io/internal/logger"
	"github.com/richard-senior/mcp/_digital-io/pkg/server"
	"github.com/richard-senior/mcp/_digital-io/pkg/transport"
)

func main() {
	// Parse command line flags
	mcpMode := flag.Bool("mcp", false, "Run as MCP server over STDIO")
	flag.Parse()

	if *mcpMode {
		runMCPServer()
	} else {
		runHTTPServer()
	}
}

func runMCPServer() {
	// In MCP mode, redirect all logs to stderr to avoid interfering with JSON-RPC on stdout
	logger.SetMCPMode(true)
	
	logger.Info("Starting Digital I/O Bank MCP Server (HTTP Client Mode)")

	// Create HTTP client to connect to running HTTP server
	httpClient := server.NewHTTPClient("http://localhost:8327")
	
	// Test connection to HTTP server with retries
	maxRetries := 3
	var err error
	for i := 0; i < maxRetries; i++ {
		_, err = httpClient.GetSystemStatus()
		if err == nil {
			break
		}
		logger.Warn("Failed to connect to HTTP server (attempt %d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	
	if err != nil {
		logger.Fatal("Failed to connect to HTTP server at localhost:8327 after %d attempts. Please ensure the HTTP server is running first. Last error: %v", maxRetries, err)
	}
	
	logger.Info("Successfully connected to HTTP server at localhost:8327")

	// Create MCP server with STDIO transport and HTTP client
	transport := transport.NewStdioTransport()
	mcpServer := server.NewServerWithHTTPClient(transport, httpClient)

	// Start the MCP server with panic recovery
	defer func() {
		if r := recover(); r != nil {
			logger.Fatal("MCP server panicked: %v", r)
		}
	}()

	// Start the MCP server
	if err := mcpServer.Start(); err != nil {
		logger.Fatal("MCP server failed: %v", err)
	}

	logger.Info("MCP server stopped")
}

func runHTTPServer() {
	logger.Info("Starting Digital I/O Bank HTTP Server")

	// Add panic recovery for the HTTP server
	defer func() {
		if r := recover(); r != nil {
			logger.Fatal("HTTP server panicked: %v", r)
		}
	}()

	// Load I/O labels with error handling
	if err := loadConfigSafely(); err != nil {
		logger.Warn("Failed to load configuration: %v", err)
	}

	// Create the I/O bank simulation
	bank := iobank.NewIOBank()
	
	// Start the simulation (inputs will change over time)
	bank.StartSimulation()
	defer bank.StopSimulation()

	// Create API handler
	apiHandler := api.NewAPIHandler(bank)
	router := apiHandler.SetupRoutes()

	// Configure HTTP server with more robust settings
	server := &http.Server{
		Addr:         ":8327",
		Handler:      router,
		ReadTimeout:  30 * time.Second,  // Increased from 15s
		WriteTimeout: 30 * time.Second,  // Increased from 15s
		IdleTimeout:  120 * time.Second, // Increased from 60s
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in a goroutine with error handling
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Server starting on http://localhost:8327")
		logger.Info("Status endpoint: http://localhost:8327/status")
		logger.Info("Press Ctrl+C to stop")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case err := <-serverErrors:
		logger.Fatal("Server failed to start: %v", err)
	case sig := <-quit:
		logger.Info("Received signal: %v", sig)
	}

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	} else {
		logger.Info("Server stopped gracefully")
	}
}

// loadConfigSafely loads configuration with error handling
func loadConfigSafely() error {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic while loading configuration: %v", r)
		}
	}()
	
	config.GetIOLabels()
	return nil
}
