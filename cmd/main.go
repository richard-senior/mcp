package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/server"
	"github.com/richard-senior/mcp/pkg/tools"
)

func main() {
	// Set log output to file before any logging occurs
	logger.SetLogOutput('f')
	// Configure logging
	logger.SetShowDateTime(true)

	// Set correct GOARCH before any tool operations
	setCorrectArchitecture()

	// Disable logging for MCP server mode to avoid interfering with JSON-RPC
	logger.SetLevel(logger.FATAL)

	// Initialize the MCP server singleton
	s := server.GetInstance()

	//logger.Info("Starting github.com/richard-senior/mcp application")

	// Check for command line arguments - if present, handle as CLI tool
	if len(os.Args) > 1 {
		logger.Info("Command line arguments received:", len(os.Args)-1)
		for i, arg := range os.Args[1:] {
			logger.Info("Arg", i+1, ":", arg)
		}
	}
	s.ProcessRequests()
	/*
		// Start the server
		if err := s.Start(); err != nil {
			os.Exit(1)
		}
	*/
}

func handleGoogleSearch(query string) {
	params := map[string]any{
		"query": query,
		"num":   5,
	}

	result, err := tools.HandleGoogleSearchTool(params)
	if err != nil {
		logger.Error("Search failed:", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}

func setCorrectArchitecture() {
	// Force correct architecture for Apple Silicon
	if runtime.GOOS == "darwin" {
		// Check if we're on Apple Silicon
		cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), "Apple") {
			logger.Info("Detected Apple Silicon, forcing GOARCH=arm64")
			os.Setenv("GOARCH", "arm64")
			os.Setenv("CGO_ENABLED", "1")
			return
		}
	}

	// Get system architecture
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("Failed to detect system architecture: %v", err)
		return // Continue with current settings
	}

	systemArch := strings.TrimSpace(string(output))
	logger.Info("Detected system architecture: %s, runtime GOARCH: %s", systemArch, runtime.GOARCH)

	// Map to Go architecture
	var targetArch string
	switch systemArch {
	case "arm64", "aarch64":
		targetArch = "arm64"
	case "x86_64", "amd64":
		targetArch = "amd64"
	default:
		logger.Warn("Unknown system architecture: %s", systemArch)
		return // Unknown architecture
	}

	// Set GOARCH if it doesn't match
	if runtime.GOARCH != targetArch {
		logger.Info("Setting GOARCH from %s to %s", runtime.GOARCH, targetArch)
		os.Setenv("GOARCH", targetArch)
	} else {
		logger.Info("GOARCH already correct: %s", targetArch)
	}
}
