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
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	RequestID   string                 `json:"requestId,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Tools       []Tool                 `json:"tools,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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
}

// createErrorResponse creates an error response
func createErrorResponse(code, message, requestID string) ([]byte, error) {
	var response ErrorResponse
	response.Error.Code = code
	response.Error.Message = message

	return json.MarshalIndent(response, "", "  ")
}

// ProcessRequest processes an MCP request and returns a response
func ProcessRequest(input []byte) ([]byte, error) {
	// Parse the input JSON
	var request MCPRequest
	if err := json.Unmarshal(input, &request); err != nil {
		logger.Error("Failed to parse input JSON", err)
		return createErrorResponse("invalid_request", fmt.Sprintf("Invalid JSON: %v", err), request.RequestID)
	}

	logger.Info("Processing request", request.Query)

	// Check if this is a calculator request
	if strings.HasPrefix(request.Query, "calculate ") {
		expression := strings.TrimPrefix(request.Query, "calculate ")
		result, err := CalculateResult(expression)
		if err != nil {
			logger.Error("Calculation error", err)
			return createErrorResponse("calculation_error", err.Error(), request.RequestID)
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
			return createErrorResponse("internal_error", "Failed to create response", request.RequestID)
		}

		return jsonResult, nil
	}

	// Check if this is a prompt registry request
	if strings.HasPrefix(request.Query, "list_prompts") || strings.HasPrefix(request.Query, "get_prompt ") {
		return ProcessPromptRegistryRequest(request.Query, request.RequestID)
	}

	// Check if this is a rule creator request
	if strings.HasPrefix(request.Query, "create_rule ") || strings.HasPrefix(request.Query, "list_rules ") {
		return ProcessRuleCreatorRequest(request.Query, request.RequestID)
	}

	// Check if this is a rules processor request
	if strings.HasPrefix(request.Query, "process_rules ") || strings.HasPrefix(request.Query, "get_rule_content ") {
		return ProcessRulesProcessorRequest(request.Query, request.RequestID)
	}
	
	// Check if this is a Google search request
	if strings.HasPrefix(request.Query, "googlesearch ") {
		return ProcessGoogleSearchRequest(request.Query, request.RequestID)
	}
	
	// Check if this is a Wikipedia image save request
	if strings.HasPrefix(request.Query, "wikipediaimagesave ") {
		return ProcessWikipediaImageSaveRequest(request.Query, request.RequestID)
	}
	
	// Check if this is a Wikipedia image search request
	if strings.HasPrefix(request.Query, "wikipediaimage ") {
		return ProcessWikipediaImageRequest(request.Query, request.RequestID)
	}

	// Create a response with example tools
	response := MCPResponse{
		RequestID: request.RequestID,
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
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	// Marshal the response to JSON
	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal response to JSON", err)
		return createErrorResponse("internal_error", "Failed to create response", request.RequestID)
	}

	return jsonResult, nil
}
