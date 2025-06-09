package protocol

import (
	"encoding/json"
	"fmt"
)

/**
https://modelcontextprotocol.info/specification/draft/basic/lifecycle/
Flow:
	LLM starts up and notices our server in config in mcp.json
	Makes json rpc 'initialize' request : eg {"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"claude-ai","version":"0.1.0"}},"jsonrpc":"2.0","id":0}
	we responsd with something telling the LLM what we are: eg {"jsonrpc":"2.0","id":0,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"Demo","version":"1.0.0"}}}
	The above tells the LLM that we are a tools server with name 'Demo'
	The LLM returns two responses usually (actually one 'notification' and one 'request':
	1) {"method":"notifications/initialized","jsonrpc":"2.0"}
	   This tells us that the LLM has acknowledge our MCP server
	2) {"method":"tools/list","params":{},"jsonrpc":"2.0","id":1}
	   This tells us that the LLM knows we are a tools server and wants to know what tools we have
	We respond with a tools listing: eg {"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"add","inputSchema":{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"],"additionalProperties":false,"$schema":"http://json-schema.org/draft-07/schema#"}}]}}
	Obviously this example is not entirely correct, but here we tell the LLM exactly what tools we expose and how to use them.
	The LLM May or May not then (regardless of whether we've published any resources or not) ask us to resources/list: {"method":"resources/list","params":{},"jsonrpc":"2.0","id":3}
	To which we can responde with a list of any resources (documents etc.) we publish, or simply

*/

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
	MethodShutdown      MethodType = "shutdown"
	MethodExit          MethodType = "exit"
	MethodCancelRequest MethodType = "$/cancelRequest"

	// Server capability methods
	MethodRegisterCapability   MethodType = "client/registerCapability"
	MethodUnregisterCapability MethodType = "client/unregisterCapability"

	// Window methods
	MethodShowMessage        MethodType = "window/showMessage"
	MethodShowMessageRequest MethodType = "window/showMessageRequest"
	MethodLogMessage         MethodType = "window/logMessage"

	// Telemetry methods
	MethodTelemetryEvent MethodType = "telemetry/event"

	// Tool execution methods
	MethodToolResult    MethodType = "tool/result"
	MethodDiscoverTools MethodType = "discover_tools"
	MethodInvokeTool    MethodType = "invoke_tool"

	// Custom MCP methods
	MethodCalculate MethodType = "mcp/calculate"
	MethodWeather   MethodType = "mcp/weather"
	MethodTimer     MethodType = "mcp/timer"
)

// Version is the JSON-RPC protocol version
const JsonRpcVersion = "2.0"

// Request represents a JSON-RPC 2.0 request object
type JsonRpcRequest struct {
	// A String specifying the version of the JSON-RPC protocol. MUST be exactly "2.0".
	JsonRPC string `json:"jsonrpc"`

	// A String containing the name of the method to be invoked.
	Method string `json:"method"`

	// A Structured value that holds the parameter values to be used during the invocation of the method.
	// This member MAY be omitted.
	Params json.RawMessage `json:"params,omitempty"`

	// An identifier established by the Client that MUST contain a String, Number, or NULL value if included.
	// If it is not included it is assumed to be a notification.
	ID interface{} `json:"id,omitempty"`
}

// Response represents a JSON-RPC 2.0 response object
type JsonRpcResponse struct {
	// A String specifying the version of the JSON-RPC protocol. MUST be exactly "2.0".
	JsonRPC string `json:"jsonrpc"`

	// This member is REQUIRED on success.
	// This member MUST NOT exist if there was an error invoking the method.
	Result json.RawMessage `json:"result,omitempty"`

	// This member is REQUIRED on error.
	// This member MUST NOT exist if there was no error triggered during invocation.
	Error *JsonRpcError `json:"error,omitempty"`

	// This member is REQUIRED.
	// It MUST be the same as the value of the id member in the Request Object.
	// If there was an error in detecting the id in the Request object (e.g. Parse error/Invalid Request), it MUST be Null.
	ID any `json:"id"`
}

// Error represents a JSON-RPC 2.0 error object
type JsonRpcError struct {
	// A Number that indicates the error type that occurred.
	Code int `json:"code"`

	// A String providing a short description of the error.
	Message string `json:"message"`

	// A Primitive or Structured value that contains additional information about the error.
	// This may be omitted.
	Data any `json:"data,omitempty"`
}

type ToolProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type InputSchema struct {
	Type                 string                  `json:"type"`
	Properties           map[string]ToolProperty `json:"properties,omitempty"`
	Required             []string                `json:"required"`
	AdditionalProperties bool                    `json:"additionalProperties"`
}

// Tool represents a tool that can be invoked by Amazon Q
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// ToolsResponse represents the response to a tools discovery request
type ToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// Resource represents a resource that can be accessed by Amazon Q, a document or other non-interactive resource
type Resource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Metadata    any    `json:"metadata,omitempty"`
}

// ToolsResponse represents the response to a tools discovery request
type ResourceResponse struct {
	Tools []Resource `json:"resources"`
}

// Standard error codes defined by the JSON-RPC 2.0 specification
const (
	// Parse error: Invalid JSON was received by the server.
	ErrParse = -32700

	// Invalid Request: The JSON sent is not a valid Request object.
	ErrInvalidRequest = -32600

	// Method not found: The method does not exist / is not available.
	ErrMethodNotFound = -32601

	// Invalid params: Invalid method parameter(s).
	ErrInvalidParams = -32602

	// Internal error: Internal JSON-RPC error.
	ErrInternal = -32603

	// Server error: Reserved for implementation-defined server-errors.
	// -32000 to -32099
	ErrServer = -32000

	// Tool execution failed
	ErrToolExecutionFailed = -32000
)

// Error returns a string representation of the error
func (e *JsonRpcError) Error() string {
	return fmt.Sprintf("jsonrpc error: code=%d message=%s", e.Code, e.Message)
}

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

// NewNotification creates a new JSON-RPC 2.0 notification (a request without an ID)
func NewJsonRpcNotification(method string, params interface{}) (*JsonRpcRequest, error) {
	return NewJsonRpcRequest(method, params, nil)
}

// NewResponse creates a new JSON-RPC 2.0 success response
func NewJsonRpcResponse(result interface{}, id interface{}) (*JsonRpcResponse, error) {
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

// NewErrorResponse creates a new JSON-RPC 2.0 error response
func NewJsonRpcErrorResponse(code int, message string, data interface{}, id interface{}) *JsonRpcResponse {
	return &JsonRpcResponse{
		JsonRPC: JsonRpcVersion,
		Error: &JsonRpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
}

// ParseRequest parses a JSON-RPC 2.0 request from raw JSON
func ParseJsonRpcRequest(data []byte) (*JsonRpcRequest, error) {
	var req JsonRpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}

	// Validate the request
	if req.JsonRPC != JsonRpcVersion {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", req.JsonRPC)
	}

	return &req, nil
}

// ParseResponse parses a JSON-RPC 2.0 response from raw JSON
func ParseJsonRpcResponse(data []byte) (*JsonRpcResponse, error) {
	var resp JsonRpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	// Validate the response
	if resp.JsonRPC != JsonRpcVersion {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", resp.JsonRPC)
	}

	return &resp, nil
}

// BatchRequest represents a batch of JSON-RPC 2.0 requests
type BatchRequest []*JsonRpcRequest

// BatchResponse represents a batch of JSON-RPC 2.0 responses
type BatchResponse []*JsonRpcResponse

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

// String returns a JSON string representation of the error
func (e *JsonRpcError) String() string {
	bytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling error: %v", err)
	}
	return string(bytes)
}
