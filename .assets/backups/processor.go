package processor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
)

// MCPRequest represents an MCP request
type MCPRequest struct {
	Query     string `json:"query"`
	RequestID string `json:"requestId"`
	JSONRPC   string `json:"jsonrpc,omitempty"`
	Method    string `json:"method,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	ID        interface{} `json:"id,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	RequestID   string                 `json:"requestId,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Tools       []Tool                 `json:"tools,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	// JSON-RPC fields
	JSONRPC string                 `json:"jsonrpc,omitempty"`
	Result  map[string]interface{} `json:"result,omitempty"`
	ID      interface{}            `json:"id,omitempty"`
}

// Tool represents a tool that can be used by the MCP client
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string      `json:"requestId,omitempty"`
	JSONRPC   string      `json:"jsonrpc,omitempty"`
	ID        interface{} `json:"id,omitempty"`
}

// createErrorResponse creates an error response
func CreateErrorResponse(code, message, requestID string) ([]byte, error) {
	var response ErrorResponse
	response.Error.Code = code
	response.Error.Message = message
	response.RequestID = requestID

	return json.MarshalIndent(response, "", "  ")
}

// ProcessRequest processes an MCP request and returns a response
func ProcessRequest(input []byte) ([]byte, error) {
	logger.Debug("ProcessRequest called with input length:", len(input))
	if len(input) > 0 {
		logger.Debug("First 100 bytes of input:", string(input[:min(100, len(input))]))
	}
	
	// Try to parse as JSON-RPC first
	var jsonRPCRequest MCPRequest
	jsonRPCErr := json.Unmarshal(input, &jsonRPCRequest)
	
	if jsonRPCErr != nil {
		logger.Debug("Failed to parse as JSON-RPC:", jsonRPCErr)
	} else {
		logger.Debug("Parsed as JSON-RPC request:", 
			"JSONRPC:", jsonRPCRequest.JSONRPC,
			"Method:", jsonRPCRequest.Method,
			"ID:", jsonRPCRequest.ID)
	}
	
	// Check if this is a JSON-RPC request
	if jsonRPCErr == nil && jsonRPCRequest.JSONRPC == "2.0" && jsonRPCRequest.Method != "" {
		logger.Debug("Valid JSON-RPC request detected")
		logger.Info("Processing JSON-RPC request:", jsonRPCRequest.Method)
		
		// Handle JSON-RPC ping
		if jsonRPCRequest.Method == "mcp.ping" {
			logger.Debug("Handling mcp.ping request")
			response := MCPResponse{
				JSONRPC: "2.0",
				Result: map[string]interface{}{
					"status": "ok",
				},
				ID: jsonRPCRequest.ID,
			}
			
			jsonResult, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				logger.Error("Failed to marshal JSON-RPC response", err)
				return nil, err
			}
			
			logger.Debug("Returning ping response")
			return jsonResult, nil
		}
		
		// Handle initialize method specially
		if jsonRPCRequest.Method == "initialize" {
			logger.Debug("Handling initialize request with special response format")
			
			// Create a response with the initialize-specific format
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id": jsonRPCRequest.ID,
				"result": map[string]interface{}{
					"capabilities": map[string]interface{}{
						"toolsSupport": true,
					},
					"serverInfo": map[string]interface{}{
						"name": "MCP Server",
						"version": "1.0.0",
					},
				},
			}
			
			jsonResult, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				logger.Error("Failed to marshal initialize response", err)
				return nil, err
			}
			
			logger.Debug("Returning initialize response")
			return jsonResult, nil
		}
		
		// For other JSON-RPC methods, return tool definitions
		logger.Debug("Returning tool definitions for JSON-RPC method:", jsonRPCRequest.Method)
		return createToolDefinitionsResponse(jsonRPCRequest.ID, "2.0", "")
	}
	
	// Parse as standard MCP request
	var request MCPRequest
	if err := json.Unmarshal(input, &request); err != nil {
		logger.Debug("Failed to parse as standard MCP request:", err)
		logger.Error("Failed to parse input JSON", err)
		return CreateErrorResponse("invalid_request", fmt.Sprintf("Invalid JSON: %v", err), "")
	}

	logger.Debug("Parsed as standard MCP request:", 
		"Query:", request.Query,
		"RequestID:", request.RequestID)
	logger.Info("Processing request:", request.Query)

	// Check if this is a calculator request
	if strings.HasPrefix(request.Query, "calculate ") {
		logger.Debug("Calculator request detected")
		expression := strings.TrimPrefix(request.Query, "calculate ")
		result, err := CalculateResult(expression)
		if err != nil {
			logger.Error("Calculation error", err)
			return CreateErrorResponse("calculation_error", err.Error(), request.RequestID)
		}

		// Create a response with the calculation result
		response := MCPResponse{
			RequestID: request.RequestID,
			Context: map[string]interface{}{
				"result":     result,
				"expression": expression,
			},
			Metadata: map[string]interface{}{
				"version": "1.0.0",
			},
		}

		// Marshal the response to JSON
		jsonResult, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal response to JSON", err)
			return CreateErrorResponse("internal_error", "Failed to create response", request.RequestID)
		}

		logger.Debug("Returning calculator response")
		return jsonResult, nil
	}

	// Check if this is a prompt registry request
	if strings.HasPrefix(request.Query, "list_prompts") || strings.HasPrefix(request.Query, "get_prompt ") {
		logger.Debug("Prompt registry request detected")
		return ProcessPromptRegistryRequest(request.Query, request.RequestID)
	}

	// Check if this is a rule creator request
	if strings.HasPrefix(request.Query, "create_rule ") || strings.HasPrefix(request.Query, "list_rules ") {
		logger.Debug("Rule creator request detected")
		return ProcessRuleCreatorRequest(request.Query, request.RequestID)
	}

	// Check if this is a rules processor request
	if strings.HasPrefix(request.Query, "process_rules ") || strings.HasPrefix(request.Query, "get_rule_content ") {
		logger.Debug("Rules processor request detected")
		return ProcessRulesProcessorRequest(request.Query, request.RequestID)
	}

	// Check if this is a Google search request
	if strings.HasPrefix(request.Query, "googlesearch ") {
		logger.Debug("Google search request detected")
		return ProcessGoogleSearchRequest(request.Query, request.RequestID)
	}

	// Check if this is a Wikipedia image save request
	if strings.HasPrefix(request.Query, "wikipediaimagesave ") {
		logger.Debug("Wikipedia image save request detected")
		return ProcessWikipediaImageSaveRequest(request.Query, request.RequestID)
	}

	// Check if this is a Wikipedia image search request
	if strings.HasPrefix(request.Query, "wikipediaimage ") {
		logger.Debug("Wikipedia image search request detected")
		return ProcessWikipediaImageRequest(request.Query, request.RequestID)
	}

	// Check if this is a webpage to markdown conversion request
	if strings.HasPrefix(request.Query, "webpage2markdown ") {
		logger.Debug("Webpage to markdown request detected")
		return ProcessWebPage2MarkdownRequest(request.Query, request.RequestID)
	}
	
	// Check if the query is empty (when run with no parameters)
	if request.Query == "" {
		logger.Debug("Empty query received")
		logger.Info("Empty query received, returning tool definitions")
	}
	
	// If no specific command was recognized, return tool definitions
	logger.Debug("No specific command recognized, returning tool definitions")
	return createToolDefinitionsResponse(nil, "", request.RequestID)
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// createToolDefinitionsResponse creates a response with tool definitions
func createToolDefinitionsResponse(id interface{}, jsonrpc string, requestID string) ([]byte, error) {
	logger.Debug("Creating tool definitions response",
		"id:", id,
		"jsonrpc:", jsonrpc,
		"requestID:", requestID)
		
	// Create a response with example tools
	response := MCPResponse{
		RequestID: requestID,
		JSONRPC:   jsonrpc,
		ID:        id,
		Tools: []Tool{
			{
				Name:        "calculator",
				Description: "A calculator tool that can perform basic arithmetic operations",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"expression": map[string]interface{}{
							"type":        "string",
							"description": "The arithmetic expression to calculate (e.g., '2 + 2')",
						},
					},
					"required": []string{"expression"},
				},
			},
			{
				Name:        "prompt_registry",
				Description: "A tool to manage and retrieve prompts from the prompt registry",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "The action to perform (list_prompts, get_prompt)",
							"enum":        []string{"list_prompts", "get_prompt"},
						},
						"prompt_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the prompt to retrieve (required for get_prompt)",
						},
					},
					"required": []string{"action"},
				},
			},
			{
				Name:        "rule_creator",
				Description: "A tool to create and manage development standard rules",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "The action to perform (create_rule, list_rules)",
							"enum":        []string{"create_rule", "list_rules"},
						},
						"tool": map[string]interface{}{
							"type":        "string",
							"description": "The tool to create rules for (amazonq, cline, roo, cursor)",
							"enum":        []string{"amazonq", "cline", "roo", "cursor"},
						},
						"rule_name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the rule to create",
						},
					},
					"required": []string{"action", "tool"},
				},
			},
			{
				Name:        "rules_processor",
				Description: "A tool to process files against development standard rules",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "The action to perform (process_rules, get_rule_content)",
							"enum":        []string{"process_rules", "get_rule_content"},
						},
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "The path to the file to process",
						},
						"registry_path": map[string]interface{}{
							"type":        "string",
							"description": "The path to the rules registry file",
						},
					},
					"required": []string{"action", "registry_path"},
				},
			},
			{
				Name:        "google_search",
				Description: "A tool to perform Google searches and return the top results",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query to perform",
						},
						"num_results": map[string]interface{}{
							"type":        "integer",
							"description": "The number of results to return (default: 5, max: 10)",
							"default":     5,
							"maximum":     10,
						},
					},
					"required": []string{"query"},
				},
			},
			{
				Name:        "wikipedia_image",
				Description: "A tool to search for images on Wikipedia",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query (e.g., 'Albert Einstein')",
						},
						"size": map[string]interface{}{
							"type":        "integer",
							"description": "The desired image size in pixels (default: 500)",
							"default":     500,
						},
					},
					"required": []string{"query"},
				},
			},
			{
				Name:        "webpage2markdown",
				Description: "A tool to convert a webpage to Markdown format and create a precis/summary of website content",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "The URL of the webpage to convert to Markdown and summarize",
						},
						"summarize": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to create a precis/summary of the content (default: false)",
							"default":     false,
						},
					},
					"required": []string{"url"},
				},
			},
		},
		Suggestions: []string{
			"Try using the calculator tool with 'calculate 2 + 2'",
			"List available prompts with 'list_prompts'",
			"Get a specific prompt with 'get_prompt [id]'",
			"Create a rule with 'create_rule [tool] [name] [description] [globs] [alwaysApply] [content]'",
			"List rules with 'list_rules [tool]'",
			"Process rules with 'process_rules [registry_path] [file_path]'",
			"Get rule content with 'get_rule_content [rule_id] [registry_path]'",
			"Search Google with 'googlesearch [query] [num_results]'",
			"Search Wikipedia for images with 'wikipediaimage [query] [size]'",
			"Save Wikipedia images to disk with 'wikipediaimagesave [query] [size] [output_path]'",
			"Convert a webpage to Markdown with 'webpage2markdown [url]'",
			"Create a summary of a webpage with 'webpage2markdown [url] summarize=true'",
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	// Marshal the response to JSON
	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal response to JSON", err)
		return CreateErrorResponse("internal_error", "Failed to create response", requestID)
	}

	logger.Debug("Tool definitions response created, length:", len(jsonResult))
	return jsonResult, nil
}
