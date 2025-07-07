package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPClient provides methods to interact with the HTTP server
type HTTPClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPClient creates a new HTTP client for the I/O server
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second, // Increased timeout
		},
	}
}

// isServerDown checks if the error indicates the server is down
func (c *HTTPClient) isServerDown(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		   strings.Contains(errStr, "no such host") ||
		   strings.Contains(errStr, "network is unreachable") ||
		   strings.Contains(errStr, "connection reset") ||
		   strings.Contains(errStr, "broken pipe")
}

// wrapError creates a user-friendly error message
func (c *HTTPClient) wrapError(operation string, err error) error {
	if c.isServerDown(err) {
		return fmt.Errorf("Digital I/O server is not running or not accessible. Please start the HTTP server first (run: ./digital-io-server). Original error: %v", err)
	}
	
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return fmt.Errorf("Digital I/O server is not responding (timeout). The server may be overloaded or stuck. Original error: %v", err)
	}
	
	return fmt.Errorf("%s failed: %v", operation, err)
}

// HealthCheck performs a basic health check on the server
func (c *HTTPClient) HealthCheck() error {
	resp, err := c.client.Get(fmt.Sprintf("%s/status", c.baseURL))
	if err != nil {
		return c.wrapError("Health check", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Digital I/O server returned HTTP %d - server may be in an error state", resp.StatusCode)
	}
	
	return nil
}

// RecordMCPMessage sends an MCP message to the HTTP server for recording
func (c *HTTPClient) RecordMCPMessage(toolName, message string) error {
	data := map[string]string{
		"tool_name": toolName,
		"message":   message,
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal MCP message: %v", err)
	}
	
	resp, err := c.client.Post(fmt.Sprintf("%s/mcp/message", c.baseURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return c.wrapError("Record MCP message", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error recording MCP message: %d", resp.StatusCode)
	}
	
	return nil
}

// GetDigitalInput reads a digital input via HTTP
func (c *HTTPClient) GetDigitalInput(pin int) (bool, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/digital/input/%d", c.baseURL, pin))
	if err != nil {
		return false, c.wrapError(fmt.Sprintf("Get digital input pin %d", pin), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Digital I/O server returned HTTP %d for digital input pin %d", resp.StatusCode, pin)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode digital input response: %v", err)
	}

	value, ok := result["value"].(bool)
	if !ok {
		return false, fmt.Errorf("invalid response format for digital input pin %d", pin)
	}

	return value, nil
}

// SetDigitalOutput sets a digital output via HTTP
func (c *HTTPClient) SetDigitalOutput(pin int, value bool) error {
	payload := map[string]interface{}{
		"value": value,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/digital/output/%d", c.baseURL, pin),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return c.wrapError(fmt.Sprintf("Set digital output pin %d", pin), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Digital I/O server returned HTTP %d for setting digital output pin %d", resp.StatusCode, pin)
	}

	return nil
}

// GetDigitalOutput reads a digital output via HTTP
func (c *HTTPClient) GetDigitalOutput(pin int) (bool, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/digital/output/%d", c.baseURL, pin))
	if err != nil {
		return false, c.wrapError(fmt.Sprintf("Get digital output pin %d", pin), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Digital I/O server returned HTTP %d for digital output pin %d", resp.StatusCode, pin)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode digital output response: %v", err)
	}

	value, ok := result["value"].(bool)
	if !ok {
		return false, fmt.Errorf("invalid response format for digital output pin %d", pin)
	}

	return value, nil
}

// GetAnalogInput reads an analog input via HTTP
func (c *HTTPClient) GetAnalogInput(pin int) (float64, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/analog/input/%d", c.baseURL, pin))
	if err != nil {
		return 0, c.wrapError(fmt.Sprintf("Get analog input pin %d", pin), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Digital I/O server returned HTTP %d for analog input pin %d", resp.StatusCode, pin)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode analog input response: %v", err)
	}

	valueStr, ok := result["value"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid response format for analog input pin %d", pin)
	}

	var value float64
	if _, err := fmt.Sscanf(valueStr, "%f", &value); err != nil {
		return 0, fmt.Errorf("failed to parse analog input value: %v", err)
	}

	return value, nil
}

// SetAnalogOutput sets an analog output via HTTP
func (c *HTTPClient) SetAnalogOutput(pin int, value float64) error {
	payload := map[string]interface{}{
		"value": value,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/analog/output/%d", c.baseURL, pin),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return c.wrapError(fmt.Sprintf("Set analog output pin %d", pin), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Digital I/O server returned HTTP %d for setting analog output pin %d", resp.StatusCode, pin)
	}

	return nil
}

// GetAnalogOutput reads an analog output via HTTP
func (c *HTTPClient) GetAnalogOutput(pin int) (float64, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/analog/output/%d", c.baseURL, pin))
	if err != nil {
		return 0, c.wrapError(fmt.Sprintf("Get analog output pin %d", pin), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Digital I/O server returned HTTP %d for analog output pin %d", resp.StatusCode, pin)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode analog output response: %v", err)
	}

	valueStr, ok := result["value"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid response format for analog output pin %d", pin)
	}

	var value float64
	if _, err := fmt.Sscanf(valueStr, "%f", &value); err != nil {
		return 0, fmt.Errorf("failed to parse analog output value: %v", err)
	}

	return value, nil
}

// GetSystemStatus gets the complete system status via HTTP
func (c *HTTPClient) GetSystemStatus() (map[string]interface{}, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/status", c.baseURL))
	if err != nil {
		return nil, c.wrapError("Get system status", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Digital I/O server returned HTTP %d for system status", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode system status response: %v", err)
	}

	return result, nil
}
