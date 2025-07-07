package server

import (
	"fmt"
	"strconv"

	"github.com/richard-senior/mcp/_digital-io/internal/config"
	"github.com/richard-senior/mcp/_digital-io/pkg/protocol"
)

// Tool creation methods

func (s *Server) createGetDigitalInputTool() protocol.Tool {
	return protocol.Tool{
		Name:        "get_digital_input",
		Description: "Read the state of a digital input pin (0-7). REQUIRED: You must specify the 'pin' parameter (integer 0-7). NOTE: Pin 0 exists but should be avoided due to potential truthy issues in MCP systems - prefer pins 1-7.",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"pin": {
					Type:        "integer",
					Description: "Digital input pin number (0-7) - Pin 0 should be avoided due to MCP truthy issues, use pins 1-7",
					Minimum:     intPtr(0),
					Maximum:     intPtr(7),
				},
			},
			Required: []string{"pin"},
		},
	}
}

func (s *Server) createSetDigitalOutputTool() protocol.Tool {
	return protocol.Tool{
		Name:        "set_digital_output",
		Description: "Set a digital output pin to HIGH/TRUE (0-15). REQUIRED: You must specify the 'pin' parameter (integer 0-15). NOTE: Pin 0 exists but should be avoided due to potential truthy issues in MCP systems - prefer pins 1-15.",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"pin": {
					Type:        "integer",
					Description: "Digital output pin number (0-15) - Pin 0 should be avoided due to MCP truthy issues, use pins 1-15",
					Minimum:     intPtr(0),
					Maximum:     intPtr(15),
				},
			},
			Required: []string{"pin"},
		},
	}
}

func (s *Server) createUnsetDigitalOutputTool() protocol.Tool {
	return protocol.Tool{
		Name:        "unset_digital_output",
		Description: "Set a digital output pin to LOW/FALSE (0-15). REQUIRED: You must specify the 'pin' parameter (integer 0-15). NOTE: Pin 0 exists but should be avoided due to potential truthy issues in MCP systems - prefer pins 1-15.",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"pin": {
					Type:        "integer",
					Description: "Digital output pin number (0-15) - Pin 0 should be avoided due to MCP truthy issues, use pins 1-15",
					Minimum:     intPtr(0),
					Maximum:     intPtr(15),
				},
			},
			Required: []string{"pin"},
		},
	}
}

func (s *Server) createGetDigitalOutputTool() protocol.Tool {
	return protocol.Tool{
		Name:        "get_digital_output",
		Description: "Read the current state of a digital output pin (0-15). REQUIRED: You must specify the 'pin' parameter (integer 0-15). NOTE: Pin 0 exists but should be avoided due to potential truthy issues in MCP systems - prefer pins 1-15.",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"pin": {
					Type:        "integer",
					Description: "Digital output pin number (0-15) - Pin 0 should be avoided due to MCP truthy issues, use pins 1-15",
					Minimum:     intPtr(0),
					Maximum:     intPtr(15),
				},
			},
			Required: []string{"pin"},
		},
	}
}

func (s *Server) createGetAnalogInputTool() protocol.Tool {
	return protocol.Tool{
		Name:        "get_analog_input",
		Description: "Read the voltage of an analog input pin (0-3). NOTE: Pin 0 exists but should be avoided due to potential truthy issues in MCP systems - prefer pins 1-3.",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"pin": {
					Type:        "integer",
					Description: "Analog input pin number (0-3) - Pin 0 should be avoided due to MCP truthy issues, use pins 1-3",
					Minimum:     intPtr(0),
					Maximum:     intPtr(3),
				},
			},
			Required: []string{"pin"},
		},
	}
}



func (s *Server) createSetAnalogOutputTool() protocol.Tool {
	return protocol.Tool{
		Name:        "set_analog_output",
		Description: "Set the voltage of an analog output pin (0-3). NOTE: Pin 0 exists but should be avoided due to potential truthy issues in MCP systems - prefer pins 1-3.",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"pin": {
					Type:        "integer",
					Description: "Analog output pin number (0-3) - Pin 0 should be avoided due to MCP truthy issues, use pins 1-3",
					Minimum:     intPtr(0),
					Maximum:     intPtr(3),
				},
				"value": {
					Type:        "number",
					Description: "Voltage to set (0.0-5.0V)",
				},
			},
			Required: []string{"pin", "value"},
		},
	}
}

func (s *Server) createGetAnalogOutputTool() protocol.Tool {
	return protocol.Tool{
		Name:        "get_analog_output",
		Description: "Read the current voltage of an analog output pin (0-3). NOTE: Pin 0 exists but should be avoided due to potential truthy issues in MCP systems - prefer pins 1-3.",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"pin": {
					Type:        "integer",
					Description: "Analog output pin number (0-3) - Pin 0 should be avoided due to MCP truthy issues, use pins 1-3",
					Minimum:     intPtr(0),
					Maximum:     intPtr(3),
				},
			},
			Required: []string{"pin"},
		},
	}
}

func (s *Server) createGetSystemStatusTool() protocol.Tool {
	return protocol.Tool{
		Name:        "get_system_status",
		Description: "Get complete system status including all I/O states and labels",
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

// Tool handler methods

func (s *Server) handleGetDigitalInput(params interface{}) (interface{}, error) {
	pin, err := s.extractIntParam(params, "pin")
	if err != nil {
		return nil, err
	}

	// Pins are 0-based, no conversion needed
	if pin < 0 || pin > 7 {
		return nil, fmt.Errorf("digital input pin %d out of range (0-7)", pin)
	}

	var value bool
	if s.httpClient != nil {
		// Use HTTP client mode
		value, err = s.httpClient.GetDigitalInput(pin)
	} else {
		// Use direct I/O bank mode
		value, err = s.ioBank.GetDigitalInput(pin)
	}
	
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pin":   pin,
		"value": value,
	}, nil
}

func (s *Server) handleSetDigitalOutput(params interface{}) (interface{}, error) {
	pin, err := s.extractIntParam(params, "pin")
	if err != nil {
		return nil, err
	}

	// Pins are 0-based, no conversion needed
	if pin < 0 || pin > 15 {
		return nil, fmt.Errorf("digital output pin %d out of range (0-15)", pin)
	}

	// Always set to true (HIGH)
	value := true

	if s.httpClient != nil {
		// Use HTTP client mode
		err = s.httpClient.SetDigitalOutput(pin, value)
	} else {
		// Use direct I/O bank mode
		err = s.ioBank.SetDigitalOutput(pin, value)
	}
	
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pin":    pin,
		"value":  value,
		"status": "success",
	}, nil
}

func (s *Server) handleUnsetDigitalOutput(params interface{}) (interface{}, error) {
	pin, err := s.extractIntParam(params, "pin")
	if err != nil {
		return nil, err
	}

	// Pins are 0-based, no conversion needed
	if pin < 0 || pin > 15 {
		return nil, fmt.Errorf("digital output pin %d out of range (0-15)", pin)
	}

	// Always set to false (LOW)
	value := false

	if s.httpClient != nil {
		// Use HTTP client mode
		err = s.httpClient.SetDigitalOutput(pin, value)
	} else {
		// Use direct I/O bank mode
		err = s.ioBank.SetDigitalOutput(pin, value)
	}
	
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pin":    pin,
		"value":  value,
		"status": "success",
	}, nil
}

func (s *Server) handleGetDigitalOutput(params interface{}) (interface{}, error) {
	pin, err := s.extractIntParam(params, "pin")
	if err != nil {
		return nil, err
	}

	// Pins are 0-based, no conversion needed
	if pin < 0 || pin > 15 {
		return nil, fmt.Errorf("digital output pin %d out of range (0-15)", pin)
	}

	var value bool
	if s.httpClient != nil {
		// Use HTTP client mode
		value, err = s.httpClient.GetDigitalOutput(pin)
	} else if s.ioBank != nil {
		// Use direct I/O bank mode
		value, err = s.ioBank.GetDigitalOutput(pin)
	} else {
		return nil, fmt.Errorf("no IOBank or HTTP client available")
	}
	
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pin":   pin,
		"value": value,
	}, nil
}

func (s *Server) handleGetAnalogInput(params interface{}) (interface{}, error) {
	pin, err := s.extractIntParam(params, "pin")
	if err != nil {
		return nil, err
	}

	// Pins are 0-based, no conversion needed
	if pin < 0 || pin > 3 {
		return nil, fmt.Errorf("analog input pin %d out of range (0-3)", pin)
	}

	var value float64
	if s.httpClient != nil {
		// Use HTTP client mode
		value, err = s.httpClient.GetAnalogInput(pin)
	} else if s.ioBank != nil {
		// Use direct I/O bank mode
		value, err = s.ioBank.GetAnalogInput(pin)
	} else {
		return nil, fmt.Errorf("no IOBank or HTTP client available")
	}
	
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pin":   pin,
		"value": fmt.Sprintf("%.3f", value),
		"unit":  "V",
	}, nil
}



func (s *Server) handleSetAnalogOutput(params interface{}) (interface{}, error) {
	pin, err := s.extractIntParam(params, "pin")
	if err != nil {
		return nil, err
	}

	value, err := s.extractFloatParam(params, "value")
	if err != nil {
		return nil, err
	}

	// Pins are 0-based, no conversion needed
	if pin < 0 || pin > 3 {
		return nil, fmt.Errorf("analog output pin %d out of range (0-3)", pin)
	}

	if s.httpClient != nil {
		// Use HTTP client mode
		err = s.httpClient.SetAnalogOutput(pin, value)
	} else if s.ioBank != nil {
		// Use direct I/O bank mode
		err = s.ioBank.SetAnalogOutput(pin, value)
	} else {
		return nil, fmt.Errorf("no IOBank or HTTP client available")
	}
	
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pin":    pin,
		"value":  fmt.Sprintf("%.3f", value),
		"unit":   "V",
		"status": "success",
	}, nil
}

func (s *Server) handleGetAnalogOutput(params interface{}) (interface{}, error) {
	pin, err := s.extractIntParam(params, "pin")
	if err != nil {
		return nil, err
	}

	// Pins are 0-based, no conversion needed
	if pin < 0 || pin > 3 {
		return nil, fmt.Errorf("analog output pin %d out of range (0-3)", pin)
	}

	var value float64
	if s.httpClient != nil {
		// Use HTTP client mode
		value, err = s.httpClient.GetAnalogOutput(pin)
	} else if s.ioBank != nil {
		// Use direct I/O bank mode
		value, err = s.ioBank.GetAnalogOutput(pin)
	} else {
		return nil, fmt.Errorf("no IOBank or HTTP client available")
	}
	
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pin":   pin,
		"value": fmt.Sprintf("%.3f", value),
		"unit":  "V",
	}, nil
}

func (s *Server) handleGetSystemStatus(params interface{}) (interface{}, error) {
	var status map[string]interface{}
	
	if s.ioBank != nil {
		// Direct mode - use IOBank directly
		status = s.ioBank.GetStatus()
	} else if s.httpClient != nil {
		// HTTP client mode - get status via HTTP
		var err error
		status, err = s.httpClient.GetSystemStatus()
		if err != nil {
			return nil, fmt.Errorf("failed to get system status via HTTP: %v", err)
		}
	} else {
		return nil, fmt.Errorf("no IOBank or HTTP client available")
	}
	
	// Add labels to the status (import the function from api package)
	labels := config.GetIOLabels()
	status["labels"] = map[string]interface{}{
		"digital_inputs":  labels.DigitalInputs,
		"digital_outputs": labels.DigitalOutputs,
		"analog_inputs":   labels.AnalogInputs,
		"analog_outputs":  labels.AnalogOutputs,
	}
	
	status["analog_ranges"] = map[string]interface{}{
		"inputs":  labels.AnalogInputRanges,
		"outputs": labels.AnalogOutputRanges,
	}
	
	return status, nil
}

// Helper methods for parameter extraction

func (s *Server) extractIntParam(params interface{}, key string) (int, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid parameters")
	}

	param, exists := paramsMap[key]
	if !exists {
		return 0, fmt.Errorf("missing required parameter: %s", key)
	}

	switch v := param.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("parameter %s must be a number", key)
	}
}

func (s *Server) extractBoolParam(params interface{}, key string) (bool, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("invalid parameters")
	}

	param, exists := paramsMap[key]
	if !exists {
		return false, fmt.Errorf("missing required parameter: %s", key)
	}

	switch v := param.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case float64:
		return v != 0, nil
	case int:
		return v != 0, nil
	default:
		return false, fmt.Errorf("parameter %s must be a boolean", key)
	}
}

func (s *Server) extractFloatParam(params interface{}, key string) (float64, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid parameters")
	}

	param, exists := paramsMap[key]
	if !exists {
		return 0, fmt.Errorf("missing required parameter: %s", key)
	}

	switch v := param.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("parameter %s must be a number", key)
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}
