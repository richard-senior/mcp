package protocol

import (
	"encoding/json"
	"fmt"
)

// MethodType defines the possible JSON-RPC method types
type MethodType string

// Method types for JSON-RPC requests
const (
	// Core protocol methods
	MethodInitialize    MethodType = "initialize"
	MethodInitialized   MethodType = "initialized"
	MethodToolsList     MethodType = "tools/list"
	MethodToolsCall     MethodType = "tools/call"
	MethodResourcesList MethodType = "resources/list"
	MethodPromptsList   MethodType = "prompts/list"
	MethodPromptsGet    MethodType = "prompts/get"
	MethodShutdown      MethodType = "shutdown"
	MethodExit          MethodType = "exit"
	MethodCancelRequest MethodType = "$/cancelRequest"

	// Tool execution methods
	MethodToolResult    MethodType = "tool/result"
	MethodDiscoverTools MethodType = "discover_tools"
	MethodInvokeTool    MethodType = "invoke_tool"
)

// Version is the JSON-RPC protocol version
const JsonRpcVersion = "2.0"

// Request represents a JSON-RPC 2.0 request object
type JsonRpcRequest struct {
	JsonRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

// Response represents a JSON-RPC 2.0 response object
type JsonRpcResponse struct {
	JsonRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JsonRpcError   `json:"error,omitempty"`
	ID      any             `json:"id,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object
type JsonRpcError struct {
	Code    int `json:"code"`
	Message string `json:"message"`
	Data    any `json:"data,omitempty"`
}

// Tool property definition
type ToolProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Minimum     *int   `json:"minimum,omitempty"`
	Maximum     *int   `json:"maximum,omitempty"`
}

// Input schema for tools
type InputSchema struct {
	Type                 string                  `json:"type"`
	Properties           map[string]ToolProperty `json:"properties,omitempty"`
	Required             []string                `json:"required"`
	AdditionalProperties bool                    `json:"additionalProperties"`
}

// Tool represents a tool that can be invoked
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// Standard error codes defined by the JSON-RPC 2.0 specification
const (
	ErrParse               = -32700
	ErrInvalidRequest      = -32600
	ErrMethodNotFound      = -32601
	ErrInvalidParams       = -32602
	ErrInternal            = -32603
	ErrServer              = -32000
	ErrToolExecutionFailed = -32000
)

// NewRequest creates a new JSON-RPC 2.0 request
func NewJsonRpcRequest(method string, params interface{}, id interface{}) (*JsonRpcRequest, error) {
	var paramsJSON json.RawMessage
	var err error

	if params != nil {
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return nil, err
		}
	}

	return &JsonRpcRequest{
		JsonRPC: JsonRpcVersion,
		Method:  method,
		Params:  paramsJSON,
		ID:      id,
	}, nil
}

// NewResponse creates a new JSON-RPC 2.0 success response
func NewJsonRpcResponse(result any, id any) (*JsonRpcResponse, error) {
	var resultJSON json.RawMessage
	var err error

	if result != nil {
		resultJSON, err = json.Marshal(result)
		if err != nil {
			return nil, err
		}
	}

	return &JsonRpcResponse{
		JsonRPC: JsonRpcVersion,
		Result:  resultJSON,
		ID:      id,
	}, nil
}

// Error returns a string representation of the error
func (e *JsonRpcError) Error() string {
	return fmt.Sprintf("jsonrpc error: code=%d message=%s", e.Code, e.Message)
}

// CreateError creates a JsonRpcError structure
func CreateError(code int, message string, data any) *JsonRpcError {
	return &JsonRpcError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse creates a new JSON-RPC 2.0 error response
func NewJsonRpcErrorResponse(code int, message string, data any, id any) *JsonRpcResponse {
	err := CreateError(code, message, data)
	return &JsonRpcResponse{
		JsonRPC: JsonRpcVersion,
		Error:   err,
		ID:      id,
	}
}

// ParseRequest parses a JSON-RPC 2.0 request from raw JSON
func ParseJsonRpcRequest(data []byte) (*JsonRpcRequest, error) {
	var req JsonRpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}

	if req.JsonRPC != JsonRpcVersion {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", req.JsonRPC)
	}

	return &req, nil
}

// String returns a JSON string representation of the request
func (r *JsonRpcRequest) String() string {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling request: %v", err)
	}
	return string(bytes)
}

// String returns a JSON string representation of the response
func (r *JsonRpcResponse) String() string {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling response: %v", err)
	}
	return string(bytes)
}
