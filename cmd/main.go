package main

import (
	"os"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/server"
	"github.com/richard-senior/mcp/pkg/util/podds"
)

func main() {
	// Configure logging
	logger.SetShowDateTime(true)

	// Set log output to file before any logging occurs
	logger.SetLogOutput('f')

	logger.Info("Starting github.com/richard-senior/mcp application")

	// Log command line arguments
	if len(os.Args) > 1 {
		logger.Info("Command line arguments received:", len(os.Args)-1)
		for i, arg := range os.Args[1:] {
			logger.Info("Arg", i+1, ":", arg)
		}
		// Check for bulk load command
		if len(os.Args) > 1 && os.Args[1] == "update-podds" {
			logger.Info("Starting PODDS bulk data load...")
			ds := podds.GetDatasourceInstance()
			if ds == nil {
				logger.Error("Failed to initialize datasource")
				os.Exit(1)
			}
			logger.Info("PODDS bulk data load completed successfully")
			return
		} else {
			logger.Info("No command line arguments provided, returning")
			return
		}
	} else {
		logger.Info("No command line arguments provided")
	}

	// Initialize the MCP server singleton
	s := server.GetInstance()

	// Start the server
	logger.Info("Starting MCP server...")
	if err := s.Start(); err != nil {
		logger.Error("Server error:", err)
		os.Exit(1)
	}

	logger.Info("MCP server shutting down")
}
