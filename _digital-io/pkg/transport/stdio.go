package transport

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/richard-senior/mcp/_digital-io/internal/logger"
	"github.com/richard-senior/mcp/_digital-io/pkg/protocol"
)

// prettyPrint controls whether JSON responses include line breaks
const prettyPrint = true

// StdioTransport implements communication over standard input/output
type StdioTransport struct {
	reader *bufio.Reader
	writer *bufio.Writer
}

// NewStdioTransport creates a new transport that uses stdin/stdout
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
}

// ReadRequest reads a JSON-RPC request from stdin
func (t *StdioTransport) ReadRequest() (*protocol.JsonRpcRequest, error) {
	logger.Debug("Waiting for request on stdin...")

	// Read the entire JSON object
	var requestData []byte
	var depth int
	var inString bool
	var escapeNext bool

	for {
		b, err := t.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				logger.Info("Received EOF on stdin, client disconnected")
				return nil, err
			}
			logger.Error("Error reading from stdin:", err)
			return nil, err
		}

		requestData = append(requestData, b)

		// Track string literals to avoid counting braces inside strings
		if !escapeNext && b == '"' {
			inString = !inString
		}

		// Track escape sequences inside strings
		if inString && b == '\\' {
			escapeNext = !escapeNext
		} else {
			escapeNext = false
		}

		// Only count braces when not inside a string
		if !inString {
			if b == '{' {
				depth++
			} else if b == '}' {
				depth--
				// If we've closed the outermost brace, we're done
				if depth == 0 {
					break
				}
			}
		}

		// Handle array-based requests
		if !inString && depth == 0 && b == ']' {
			break
		}

		// Initialize depth when we find the first opening brace or bracket
		if depth == 0 && !inString && b == '[' {
			depth = 1
		}
	}

	// Trim any whitespace
	requestStr := strings.TrimSpace(string(requestData))
	logger.Debug("Received raw request:", requestStr)

	// Parse the JSON-RPC request
	request, err := protocol.ParseJsonRpcRequest([]byte(requestStr))
	if err != nil {
		logger.Error("Failed to parse JSON-RPC request:", err)
		return nil, err
	}

	return request, nil
}

// WriteResponse writes a JSON-RPC response to stdout
func (t *StdioTransport) WriteResponse(response *protocol.JsonRpcResponse) error {
	var responseBytes []byte
	var err error

	// Marshal the response to JSON based on prettyPrint setting
	if prettyPrint {
		responseBytes, err = json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response:", err)
			return err
		}
	} else {
		// For non-pretty printing, use json.Marshal and then compact
		prettyBytes, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response:", err)
			return err
		}

		// Create a buffer with no whitespace
		var buf bytes.Buffer
		if err := json.Compact(&buf, prettyBytes); err != nil {
			logger.Error("Failed to compact JSON:", err)
			return err
		}
		responseBytes = buf.Bytes()
	}

	// Add a newline to the response
	responseBytes = append(responseBytes, '\n')

	logger.Debug("Sending response:", string(responseBytes))

	// Write the response to stdout
	if _, err := t.writer.Write(responseBytes); err != nil {
		logger.Error("Failed to write response:", err)
		return err
	}

	// Flush to ensure the response is sent
	if err := t.writer.Flush(); err != nil {
		logger.Error("Failed to flush response:", err)
		return err
	}

	logger.Info("Response sent successfully")
	return nil
}
