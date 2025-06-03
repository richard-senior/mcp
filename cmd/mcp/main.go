package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/internal/processor"
)

func main() {
	// Parse command line flags
	debug := flag.Bool("debug", false, "Enable debug logging")
	inputFile := flag.String("input", "", "Input file path (if not provided, stdin will be used)")
	outputFile := flag.String("output", "", "Output file path (if not provided, stdout will be used)")
	flag.Parse()

	// Configure logging
	logger.SetShowDateTime(true)
	if *debug {
		logger.Debug("Debug logging enabled")
	}

	logger.Info("Starting MCP CLI application")

	// Determine input source
	var input []byte
	var err error
	if *inputFile != "" {
		input, err = os.ReadFile(*inputFile)
		if err != nil {
			logger.Fatal("Failed to read input file", err)
		}
	} else {
		// Check if there are command line arguments
		args := flag.Args()
		if len(args) > 0 {
			// Create a JSON request from command line arguments
			query := strings.Join(args, " ")
			requestID := fmt.Sprintf("cli-%d", os.Getpid())
			request := map[string]string{
				"query":     query,
				"requestId": requestID,
			}
			input, err = json.Marshal(request)
			if err != nil {
				logger.Fatal("Failed to create request from command line arguments", err)
			}
		} else {
			// Read from stdin
			input, err = io.ReadAll(os.Stdin)
			if err != nil {
				logger.Fatal("Failed to read from stdin", err)
			}
		}
	}

	// Process the input
	result, err := processor.ProcessRequest(input)
	if err != nil {
		logger.Error("Failed to process request", err)
		os.Exit(1)
	}

	// Determine output destination
	if *outputFile != "" {
		err = os.WriteFile(*outputFile, result, 0644)
		if err != nil {
			logger.Fatal("Failed to write to output file", err)
		}
	} else {
		fmt.Print(string(result))
	}

	logger.Info("MCP CLI application completed successfully")
}
