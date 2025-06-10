package main

import (
	"fmt"
	"os"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/resources"
	"github.com/richard-senior/mcp/pkg/server"
	"github.com/richard-senior/mcp/pkg/tools"
	"github.com/richard-senior/mcp/pkg/transport"
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
			logger.Debug(fmt.Sprintf("Argument %d:", i+1), arg)
		}
	} else {
		logger.Info("No command line arguments provided")
	}

	// Create a transport for communication
	t := transport.NewStdioTransport()

	// Create the MCP server
	s := server.NewServer(t)

	// Register tools with proper MCP naming convention (mcp___)
	dateTimeTool := tools.DateTimeTool()
	dateTimeTool.Name = "mcp___" + dateTimeTool.Name
	s.RegisterTool(dateTimeTool, tools.HandleDateTimeTool)

	// Register tools from the backup
	calculatorTool := tools.CalculatorTool()
	calculatorTool.Name = "mcp___" + calculatorTool.Name
	s.RegisterTool(calculatorTool, tools.HandleCalculatorTool)

	googleSearchTool := tools.GoogleSearchTool()
	googleSearchTool.Name = "mcp___" + googleSearchTool.Name
	s.RegisterTool(googleSearchTool, tools.HandleGoogleSearchTool)

	wikipediaImageTool := tools.WikipediaImageTool()
	wikipediaImageTool.Name = "mcp___" + wikipediaImageTool.Name
	s.RegisterTool(wikipediaImageTool, tools.HandleWikipediaImageTool)

	// Register resources
	s.RegisterResource(resources.ExampleResource())
	s.RegisterResource(resources.WeatherResource())

	// Start the server
	logger.Info("Starting MCP server...")
	if err := s.Start(); err != nil {
		logger.Error("Server error:", err)
		os.Exit(1)
	}

	logger.Info("MCP server shutting down")
}
