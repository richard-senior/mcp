package server

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/richard-senior/mcp/_digital-io/internal/iobank"
	"github.com/richard-senior/mcp/_digital-io/internal/logger"
	"github.com/richard-senior/mcp/_digital-io/pkg/protocol"
	"github.com/richard-senior/mcp/_digital-io/pkg/transport"
)

// Server represents an MCP server
type Server struct {
	transport  transport.Transport
	handlers   map[string]HandlerFunc
	tools      []protocol.Tool
	ioBank     *iobank.IOBank
	httpClient *HTTPClient // For HTTP client mode
}

// HandlerFunc is a function that handles an MCP request
type HandlerFunc func(params interface{}) (interface{}, error)

// NewServer creates a new MCP server instance
func NewServer(t transport.Transport, bank *iobank.IOBank) *Server {
	server := &Server{
		transport:  t,
		handlers:   make(map[string]HandlerFunc),
		tools:      []protocol.Tool{},
		ioBank:     bank,
		httpClient: nil,
	}

	// Register default tools
	server.RegisterDefaultTools()

	return server
}

// NewServerWithHTTPClient creates a new MCP server instance with HTTP client access
func NewServerWithHTTPClient(t transport.Transport, client *HTTPClient) *Server {
	server := &Server{
		transport:  t,
		handlers:   make(map[string]HandlerFunc),
		tools:      []protocol.Tool{},
		ioBank:     nil,
		httpClient: client,
	}

	// Register default tools
	server.RegisterDefaultTools()

	return server
}

// RegisterTool registers a tool with the server
func (s *Server) RegisterTool(tool protocol.Tool, handler HandlerFunc) {
	s.tools = append(s.tools, tool)
	s.handlers[tool.Name] = handler
	logger.Info("Registered tool:", tool.Name)
}

// RegisterDefaultTools registers all the default tools with the server
func (s *Server) RegisterDefaultTools() {
	logger.Info("Registering default tools...")

	// Register digital input tool
	s.RegisterTool(s.createGetDigitalInputTool(), s.handleGetDigitalInput)

	// Register digital output tools
	s.RegisterTool(s.createSetDigitalOutputTool(), s.handleSetDigitalOutput)
	s.RegisterTool(s.createUnsetDigitalOutputTool(), s.handleUnsetDigitalOutput)
	s.RegisterTool(s.createGetDigitalOutputTool(), s.handleGetDigitalOutput)

	// Register analog input tools
	s.RegisterTool(s.createGetAnalogInputTool(), s.handleGetAnalogInput)

	// Register analog output tools
	s.RegisterTool(s.createSetAnalogOutputTool(), s.handleSetAnalogOutput)
	s.RegisterTool(s.createGetAnalogOutputTool(), s.handleGetAnalogOutput)

	// Register system status tool
	s.RegisterTool(s.createGetSystemStatusTool(), s.handleGetSystemStatus)
}

// Start starts the server and begins processing requests
func (s *Server) Start() error {
	logger.Info("Starting MCP server")

	// Register built-in handlers
	s.handlers[string(protocol.MethodInitialize)] = s.handleInitialize
	s.handlers[string(protocol.MethodInitialized)] = s.handleInitialized
	s.handlers[string(protocol.MethodToolsList)] = s.handleToolsList
	s.handlers[string(protocol.MethodToolsCall)] = s.handleToolsCall

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start processing in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.processRequests()
	}()

	// Wait for either an error or a signal
	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		logger.Info("Received signal:", sig)
		return nil
	}
}

// processRequests continuously processes incoming requests
func (s *Server) processRequests() error {
	for {
		// Read a request
		req, err := s.transport.ReadRequest()
		if err != nil {
			return err
		}

		// Process the request
		resp := s.handleRequest(req)
		if resp == nil {
			continue
		}

		// Send the response
		if err := s.transport.WriteResponse(resp); err != nil {
			return err
		}
	}
}

// handleRequest processes a request and returns a response
func (s *Server) handleRequest(req *protocol.JsonRpcRequest) *protocol.JsonRpcResponse {
	// Create a base response
	resp := &protocol.JsonRpcResponse{
		JsonRPC: protocol.JsonRpcVersion,
		ID:      req.ID,
	}

	logger.Info(">> ", req.Method)

	// Find the appropriate handler
	var handler HandlerFunc
	var params any

	if req.Method == string(protocol.MethodInvokeTool) {
		// For invoke_tool, extract the tool name and parameters
		var invokeParams map[string]any
		if err := json.Unmarshal(req.Params, &invokeParams); err != nil {
			resp.Error = &protocol.JsonRpcError{
				Code:    protocol.ErrInvalidParams,
				Message: "Invalid parameters for invoke_tool: " + err.Error(),
			}
			return resp
		}

		toolName, ok := invokeParams["name"].(string)
		if !ok {
			resp.Error = &protocol.JsonRpcError{
				Code:    protocol.ErrInvalidParams,
				Message: "Missing tool name in invoke_tool parameters",
			}
			return resp
		}

		logger.Info("Tool invocation requested for:", toolName)

		// Record MCP message
		if s.ioBank != nil {
			s.ioBank.AddMCPMessage(toolName, fmt.Sprintf("MCP tool '%s' invoked", toolName))
		} else if s.httpClient != nil {
			// In HTTP client mode, send the message to the HTTP server
			if err := s.httpClient.RecordMCPMessage(toolName, fmt.Sprintf("MCP tool '%s' invoked", toolName)); err != nil {
				logger.Warn("Failed to record MCP message:", err)
			}
		}

		handler = s.handlers[toolName]
		params = invokeParams["parameters"]
	} else {
		// For other methods, use the method name directly
		handler = s.handlers[req.Method]
		params = req.Params
	}

	// If no handler is found, return an error
	if handler == nil {
		resp.Error = &protocol.JsonRpcError{
			Code:    protocol.ErrMethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
		return resp
	}

	// Execute the handler
	result, err := handler(params)

	if err == nil && result == nil {
		return nil
	}

	if err != nil {
		resp.Error = &protocol.JsonRpcError{
			Code:    protocol.ErrToolExecutionFailed,
			Message: err.Error(),
		}
		return resp
	}

	// Set the result
	resultBytes, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		resp.Error = &protocol.JsonRpcError{
			Code:    protocol.ErrInternal,
			Message: "Failed to marshal result: " + err.Error(),
		}
		return resp
	}

	logger.Inform("output \n", string(resultBytes))
	resp.Result = resultBytes
	return resp
}

// handleInitialize handles the initialize method
func (s *Server) handleInitialize(params interface{}) (interface{}, error) {
	logger.Info("Handling initialize request")

	initializeResponse := struct {
		ProtocolVersion string         `json:"protocolVersion"`
		Capabilities    map[string]any `json:"capabilities"`
		ServerInfo      struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description,omitempty"`
		} `json:"serverInfo"`
	}{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]any{
			"tools": struct{}{},
		},
		ServerInfo: struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description,omitempty"`
		}{
			Name:    "digital-io-server",
			Version: "1.0.0",
			Description: `
				Digital and Analog I/O Bank tools for controlling and monitoring external hardware device with digital and analog I/O pins.
				Use these tools to read digital inputs, set digital outputs, read analog inputs (voltages), set analog outputs (voltages), and monitor the system status.
				Digital I/O is currently connected to a tea making machine which works as follows:

				- There is a 2L capacity kettle above a cup. Also above the cup and adjacent to the kettle are milk, sugar and teabag dispensers which dispense into the cup.
				- The water can enter the kettle at the top and exit the kettle at the bottom where it can flow directly into a cup if that cup is in place.
				- IMPORTANT: Don't have both inlet and outlet valves open simultaneously
				- IMPORTANT: Ensure a cup is dispensed before opening the kettle outlet
				- IMPORTANT: The cup can only hold 300ml. Monitor the weight of the kettle or the cup when filling to ensure you do not overfill the cup
				- The cup weight is zero when there is no cup, and when there is an empty cup, and only raises when the cup is filled. Use DI0 to check if a cup is in place.
				- The Kettle does not turn itself off, you must monitor the temperature
				- IMPORTANT: Do not power the kettle when there is less than 100ml of water in it, or the kettle will be destroyed
				- You can tell if anything has been dispensed into the cup by reading its weight.
				- The teaspoon actuators perform various teaspoon related tasks such as raising and lowering the spoon, and 'squashing' it to the side of the cup etc.
				- Squashing will trap the teabag between the side of the up and the spoon, squashing and raising will remove the teabag from the cup.
				- IMPORTANT do not stir and squash at the same time, though you may (and will) squash and raise at the same time
				- IMPORTANT you will need to write code to monitor fast moving processes like kettle filling etc.

				The Digital IO is connected to the tea machine as follows:
				digital_input pin 0: DI 00 (unused)
				digital_input pin 1: Cup is dispensed (high = true)
				digital_input pin 2: Teaspoon in cup (high = true)
				digital_input pin 3: Stirring (high = true)
				digital_input pin 4: Squashing (high = true)
				digital_input pin 5: Teabag In (high = true)
				digital_output pin 0: DO 00 (unused)
				digital_output pin 1: Water In valve (Kettle) (high = open)
    			digital_output pin 2: Water Out valve (Kettle) (high = open)
    			digital_output pin 3: Power Relay (Kettle) (high = on)
				digital_output pin 4: Cup Dispenser Solenoid (~100ms Pulse [off-on-off] (falling edge) to dispense a cup)
				digital_output pin 5: Teabag Dispenser Solenoid (~100ms Pulse [off-on-off] (falling edge) to dispense a teabag)
    			digital_output pin 6: Sugar Dispenser Solenoid (~100ms Pulse [off-on-off] (falling edge) to dispense ONE sugar)
    			digital_output pin 7: Milk Dispenser Solenoid (high = open) Read cup weight to determine how long to leave this valve open
    			digital_output pin 8: Up/Down Teaspoon actuator (output high[on] to LOWER the spoon, output low[off] to RAISE the spoon)
				digital_output pin 9: Stir Teaspoon actuator (high = stir, low = not stir)
				digital_output pin 10: Squash Teaspoon Actuator (high = squash, low = return to centre of cup)
				analog_input pin 0: AI 00 (unused)
				analog_input pin 1: Kettle Water Temperature (0-5v = 0-100 degrees c)
				analog_input pin 2: Cup Weight (0-5v = 0-1000g)
				analog_input pin 3: Kettle Weight (0-5v = 0-2000g)
		`,
		},
	}

	return initializeResponse, nil
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(params interface{}) (interface{}, error) {
	logger.Info("Handling initialized notification")
	return nil, nil
}

// handleToolsList handles the tools/list method
func (s *Server) handleToolsList(params interface{}) (interface{}, error) {
	logger.Info("Handling tools/list request")

	toolsResponse := struct {
		Description string          `json:"description,omitempty"`
		Tools       []protocol.Tool `json:"tools"`
	}{
		Description: `Tools for manipulating the IO`,
		Tools:       s.tools,
	}

	return toolsResponse, nil
}

// handleToolsCall handles the tools/call method
func (s *Server) handleToolsCall(params any) (any, error) {
	logger.Info("Handling tools/call request")

	type ToolCallParams struct {
		Arguments map[string]any `json:"arguments"`
		Name      string         `json:"name"`
	}

	var toolCallParams ToolCallParams

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %v", err)
	}

	if err := json.Unmarshal(paramsBytes, &toolCallParams); err != nil {
		return nil, fmt.Errorf("invalid tools/call parameters: %v", err)
	}

	logger.Info("Tool call requested for:", toolCallParams.Name)

	// Record MCP message
	if s.ioBank != nil {
		s.ioBank.AddMCPMessage(toolCallParams.Name, fmt.Sprintf("MCP tool '%s' called", toolCallParams.Name))
	} else if s.httpClient != nil {
		// In HTTP client mode, send the message to the HTTP server
		if err := s.httpClient.RecordMCPMessage(toolCallParams.Name, fmt.Sprintf("MCP tool '%s' called", toolCallParams.Name)); err != nil {
			logger.Warn("Failed to record MCP message:", err)
		}
	}

	handler := s.handlers[toolCallParams.Name]
	if handler == nil {
		return nil, fmt.Errorf("tool not found: %s", toolCallParams.Name)
	}

	result, err := handler(toolCallParams.Arguments)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %v", err)
	}

	// Format result as MCP tool response
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": s.formatToolResult(toolCallParams.Name, result),
			},
		},
	}, nil
}

// formatToolResult formats tool results for MCP text response
func (s *Server) formatToolResult(toolName string, result interface{}) string {
	switch {
	case strings.Contains(toolName, "digital_input") || strings.Contains(toolName, "digital_output"):
		if data, ok := result.(map[string]interface{}); ok {
			pin := data["pin"]
			value := data["value"]
			return fmt.Sprintf("Pin %v: %v", pin, value)
		}
	case strings.Contains(toolName, "analog"):
		if data, ok := result.(map[string]interface{}); ok {
			pin := data["pin"]
			value := data["value"]
			unit := data["unit"]
			if unit != nil {
				return fmt.Sprintf("Pin %v: %v %v", pin, value, unit)
			}
			return fmt.Sprintf("Pin %v: %v", pin, value)
		}
	case strings.Contains(toolName, "system_status"):
		return s.formatSystemStatus(result)
	case strings.Contains(toolName, "simulation"):
		if data, ok := result.(map[string]interface{}); ok {
			if status, exists := data["status"]; exists {
				return fmt.Sprintf("Simulation: %v", status)
			}
		}
	}

	// Default JSON formatting
	if jsonBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
		return string(jsonBytes)
	}
	return fmt.Sprintf("%v", result)
}

// formatSystemStatus formats system status for readable text output
func (s *Server) formatSystemStatus(result interface{}) string {
	status, ok := result.(map[string]interface{})
	if !ok {
		return "Unable to format system status"
	}

	var output strings.Builder
	output.WriteString("=== Digital I/O Bank Status ===\n\n")

	// Digital Inputs
	if digitalInputs, ok := status["digital_inputs"].([]bool); ok {
		output.WriteString("Digital Inputs:\n")
		for i, value := range digitalInputs {
			output.WriteString(fmt.Sprintf("  %d: %t\n", i, value))
		}
		output.WriteString("\n")
	}

	// Digital Outputs
	if digitalOutputs, ok := status["digital_outputs"].([]bool); ok {
		output.WriteString("Digital Outputs:\n")
		for i, value := range digitalOutputs {
			output.WriteString(fmt.Sprintf("  %d: %t\n", i, value))
		}
		output.WriteString("\n")
	}

	// Analog Inputs
	if analogInputs, ok := status["analog_inputs"].([]float64); ok {
		output.WriteString("Analog Inputs:\n")
		for i, voltage := range analogInputs {
			output.WriteString(fmt.Sprintf("  %d: %.3fV\n", i, voltage))
		}
		output.WriteString("\n")
	}

	// Analog Outputs
	if analogOutputs, ok := status["analog_outputs"].([]float64); ok {
		output.WriteString("Analog Outputs:\n")
		for i, voltage := range analogOutputs {
			output.WriteString(fmt.Sprintf("  %d: %.3fV\n", i, voltage))
		}
		output.WriteString("\n")
	}

	return output.String()
}
