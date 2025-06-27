package server

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
	"github.com/richard-senior/mcp/pkg/prompts"
	"github.com/richard-senior/mcp/pkg/resources"
	"github.com/richard-senior/mcp/pkg/tools"
	"github.com/richard-senior/mcp/pkg/transport"
)

// Server represents an MCP server
type Server struct {
	transport transport.Transport
	handlers  map[string]HandlerFunc
	tools     []protocol.Tool
	resources []protocol.Resource
	prompts   []protocol.Prompt
}

// HandlerFunc is a function that handles an MCP request
type HandlerFunc func(params interface{}) (interface{}, error)

// Singleton instance
var (
	instance *Server
	once     sync.Once
	mu       sync.Mutex
)

// GetInstance returns the singleton instance of the Server
func GetInstance() *Server {
	if instance == nil {
		// Create a transport for communication
		t := transport.NewStdioTransport()
		// TODO more transports!
		instance = InitInstance(t)
		logger.Warn("Server instance requested but not initialized. Use InitInstance first.")
	}
	return instance
}

// InitInstance initializes the singleton instance of the Server with the specified transport
func InitInstance(t transport.Transport) *Server {
	once.Do(func() {
		instance = &Server{
			transport: t,
			handlers:  make(map[string]HandlerFunc),
			tools:     []protocol.Tool{},
			resources: []protocol.Resource{},
			prompts:   []protocol.Prompt{},
		}
		// Register default tools and resources
		instance.RegisterDefaultTools()
		instance.RegisterDefaultResources()
		instance.RegisterDefaultPrompts()
	})
	return instance
}

// RegisterTool registers a tool with the server
func (s *Server) RegisterTool(tool protocol.Tool, handler HandlerFunc) {
	mu.Lock()
	defer mu.Unlock()

	s.tools = append(s.tools, tool)
	s.handlers[tool.Name] = handler
	logger.Info("Registered tool:", tool.Name)
}

// RegisterResource registers a resource with the server
func (s *Server) RegisterResource(resource protocol.Resource) {
	mu.Lock()
	defer mu.Unlock()

	s.resources = append(s.resources, resource)
	logger.Info("Registered resource:", resource.Name)
}

// RegisterDefaultTools registers all the default tools with the server
func (s *Server) RegisterDefaultTools() {
	logger.Info("Registering default tools...")

	// Register Google search tool
	googleSearchTool := tools.GoogleSearchTool()
	googleSearchTool.Name = "mcp___" + googleSearchTool.Name
	s.RegisterTool(googleSearchTool, tools.HandleGoogleSearchTool)

	// Register Html to Markdown tool
	html2MarkdownTool := tools.HTMLToMarkdownTool()
	html2MarkdownTool.Name = "mcp___" + html2MarkdownTool.Name
	s.RegisterTool(html2MarkdownTool, tools.HandleURLToMarkdown)

	// Register Wikipedia image tool
	wikipediaImageTool := tools.WikipediaImageTool()
	wikipediaImageTool.Name = "mcp___" + wikipediaImageTool.Name
	s.RegisterTool(wikipediaImageTool, tools.HandleWikipediaImageTool)

	// Register Meme tool
	memeTool := tools.NewMemeTool()
	memeTool.Name = "mcp___" + memeTool.Name
	s.RegisterTool(memeTool, tools.HandleMemeTool)

	// Register Thoughts tool
	//thoughtsTool := tools.NewThoughtsTool()
	//thoughtsTool.Name = "mcp___" + thoughtsTool.Name
	//s.RegisterTool(thoughtsTool, tools.HandleThoughts)

	// Register SVG Tools
	//svgTool := tools.NewSvgTool()
	//svgTool.Name = "mcp___" + svgTool.Name
	//s.RegisterTool(svgTool, tools.HandleSvgTool)
}

// RegisterDefaultResources registers all the default resources with the server
func (s *Server) RegisterDefaultPrompts() {
	logger.Info("Registering default prompts...")
	
	// Initialize the prompt registry which will create sample prompts
	registry := prompts.GetGlobalRegistry()
	
	// Get all prompts from the registry
	promptList, err := registry.ListPrompts()
	if err != nil {
		logger.Error("Failed to load prompts from registry", err)
		return
	}
	
	// Add prompts to server
	mu.Lock()
	s.prompts = promptList
	mu.Unlock()
	
	logger.Info("Loaded prompts from registry", len(promptList))
}

// RegisterDefaultResources registers all the default resources with the server
func (s *Server) RegisterDefaultResources() {
	logger.Info("Registering default resources...")

	// Register example resource
	s.RegisterResource(resources.ExampleResource())

	// Register weather resource
	s.RegisterResource(resources.WeatherResource())
}

// Start starts the server and begins processing requests
func (s *Server) Start() error {
	logger.Info("Starting MCP server")
	// Register built-in handlers
	s.handlers[string(protocol.MethodInitialize)] = s.handleInitialize
	s.handlers[string(protocol.MethodInitialized)] = s.handleInitialized
	s.handlers[string(protocol.MethodToolsList)] = s.handleToolsList
	//s.handlers[string(protocol.MethodResourcesList)] = s.handleResourcesList
	s.handlers[string(protocol.MethodToolsCall)] = s.handleToolsCall
	s.handlers[string(protocol.MethodPromptsList)] = s.handlePromptsList
	s.handlers[string(protocol.MethodPromptsGet)] = s.handlePromptsGet

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
		// if it is nil then this is not an error, it is just that no response is required
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
// TODO deal with multiple protocols
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

		// Log the requested tool name
		logger.Info("Tool invocation requested for:", toolName)

		// Try to find the handler directly
		handler = s.handlers[toolName]

		// If not found, try to strip the prefix if it exists (for mcp___ prefix)
		if handler == nil && strings.HasPrefix(toolName, "mcp___") {
			strippedName := strings.TrimPrefix(toolName, "mcp___")
			logger.Info("Trying with stripped name:", strippedName)
			handler = s.handlers[strippedName]
		}

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

// handlePromptsList returns a list of stored prompts
func (s *Server) handlePromptsList(params interface{}) (interface{}, error) {
	logger.Info("Handling prompts/list request")

	// Create simplified prompt entries for the list response
	type PromptListEntry struct {
		Name        string                    `json:"name"`
		Description string                    `json:"description,omitempty"`
		Arguments   map[string]protocol.PromptArgument `json:"arguments,omitempty"`
	}

	var promptList []PromptListEntry
	for _, prompt := range s.prompts {
		promptList = append(promptList, PromptListEntry{
			Name:        prompt.ID, // Use ID as name for MCP compatibility
			Description: prompt.Description,
			Arguments:   prompt.Variables,
		})
	}

	// Create a response structure that lists all registered prompts
	promptsResponse := struct {
		Prompts []PromptListEntry `json:"prompts"`
	}{
		Prompts: promptList,
	}

	return promptsResponse, nil
}

// handlePromptsGet handles the prompts/get method
func (s *Server) handlePromptsGet(params interface{}) (interface{}, error) {
	logger.Info("Handling prompts/get request")

	// Parse the parameters to get the prompt name/ID
	type PromptsGetParams struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments,omitempty"`
	}

	var getParams PromptsGetParams

	// Convert params to JSON and then unmarshal it
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %v", err)
	}

	if err := json.Unmarshal(paramsBytes, &getParams); err != nil {
		return nil, fmt.Errorf("invalid prompts/get parameters: %v", err)
	}

	logger.Info("Prompt get requested for:", getParams.Name)

	// Get the prompt from the registry
	registry := prompts.GetGlobalRegistry()
	prompt, err := registry.GetPrompt(getParams.Name)
	if err != nil {
		return nil, fmt.Errorf("prompt not found: %s", getParams.Name)
	}

	// Process the prompt content with any provided arguments
	content := prompt.Content
	if getParams.Arguments != nil {
		for key, value := range getParams.Arguments {
			placeholder := fmt.Sprintf("{{%s}}", key)
			content = strings.ReplaceAll(content, placeholder, value)
		}
	}

	// Return the processed prompt
	response := struct {
		Description string                 `json:"description"`
		Messages    []protocol.PromptMessage `json:"messages"`
	}{
		Description: prompt.Description,
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.PromptContent{
					Type: "text",
					Text: content,
				},
			},
		},
	}

	return response, nil
}

// handleToolsList handles the tools/list method
func (s *Server) handleToolsList(params interface{}) (interface{}, error) {
	logger.Info("Handling tools/list request")

	// Example response format from comment:
	// {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"add","inputSchema":{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"],"additionalProperties":false,"$schema":"http://json-schema.org/draft-07/schema#"}}]}}

	// Create a response structure that lists all registered tools
	toolsResponse := struct {
		Tools []protocol.Tool `json:"tools"`
	}{
		Tools: s.tools,
	}

	// Return the tools response directly - the handleRequest function will wrap it in a JSON-RPC response
	return toolsResponse, nil
}

// handleResourcesList handles the resources/list method
func (s *Server) handleResourcesList(params interface{}) (interface{}, error) {
	logger.Info("Handling resources/list request")

	// Create a response structure that lists all registered resources
	resourcesResponse := struct {
		Resources []protocol.Resource `json:"resources"`
	}{
		Resources: s.resources,
	}

	// Return the resources response directly - the handleRequest function will wrap it in a JSON-RPC response
	return resourcesResponse, nil
}

// handleInitialize handles the initialize method
func (s *Server) handleInitialize(params interface{}) (interface{}, error) {
	logger.Info("Handling initialize request:")

	// Create the initialize response structure based on the example
	// {"jsonrpc":"2.0","id":0,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"Demo","version":"1.0.0"}}}
	initializeResponse := struct {
		ProtocolVersion string         `json:"protocolVersion"`
		Capabilities    map[string]any `json:"capabilities"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]any{
			// Resources do not seem to be supported
			// "resources": struct{}{},
			// Tools are very much supported
			"tools": struct{}{},
			// Prompts are supported
			"prompts": struct{}{},
			// Rules are not supported
			//"rules": struct{}{},
		},
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name:    "mpc",
			Version: "1.0.0",
		},
	}

	return initializeResponse, nil
}

// handleInitialized handles the initialized notification
// 'initialized' Does not require a response
func (s *Server) handleInitialized(params interface{}) (interface{}, error) {
	logger.Info("Handling initialized notification")
	// This is typically just an acknowledgment that doesn't require a response
	return nil, nil
}

func (s *Server) handleToolsCall(params any) (any, error) {
	logger.Info("Handling tools/call request")

	// Parse the parameters
	type ToolCallParams struct {
		Arguments map[string]any `json:"arguments"`
		Name      string         `json:"name"`
	}

	var toolCallParams ToolCallParams

	// Convert params to JSON and then unmarshal it
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %v", err)
	}

	if err := json.Unmarshal(paramsBytes, &toolCallParams); err != nil {
		return nil, fmt.Errorf("invalid tools/call parameters: %v", err)
	}

	logger.Info("Tool call requested for:", toolCallParams.Name)

	// Look up the tool handler
	toolName := toolCallParams.Name
	handler := s.handlers[toolName]

	// If not found, try to strip the prefix if it exists (for mcp___ prefix)
	if handler == nil && strings.HasPrefix(toolName, "mcp___") {
		strippedName := strings.TrimPrefix(toolName, "mcp___")
		logger.Info("Trying with stripped name:", strippedName)
		handler = s.handlers[strippedName]
	}

	// If still no handler is found, return an error
	if handler == nil {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// Execute the tool with the provided arguments
	result, err := handler(toolCallParams.Arguments)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %v", err)
	}

	return result, nil
}
