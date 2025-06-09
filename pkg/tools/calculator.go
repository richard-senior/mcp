package tools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

// CalculatorTool returns the calculator tool definition
func CalculatorTool() protocol.Tool {
	return protocol.Tool{
		Name:        "calculator",
		Description: "A simple calculator that can perform basic arithmetic operations",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"expression": {
					Type:        "string",
					Description: "A simple arithmetic expression such as 2+2 or 4*6",
				},
			},
			Required: []string{"expression"},
		},
	}
}

// HandleCalculatorTool handles the calculator tool invocation
func HandleCalculatorTool(params interface{}) (any, error) {
	logger.Info("Handling calculator tool invocation")

	// Parse parameters
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	expression, ok := paramsMap["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("expression parameter is required and must be a string")
	}

	// Calculate the result
	result, err := calculateResult(expression)
	if err != nil {
		return nil, err
	}

	// Return the result
	return map[string]any{
		"result":     result,
		"expression": expression,
	}, nil
}

// calculateResult performs a simple calculation based on the input expression
func calculateResult(expression string) (float64, error) {
	// Trim whitespace
	expression = strings.TrimSpace(expression)

	// Simple parser for basic operations
	parts := strings.Fields(expression)

	if len(parts) != 3 {
		return 0, fmt.Errorf("expression must be in format 'number operator number'")
	}

	// Parse first number
	num1, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid first number: %v", err)
	}

	// Get operator
	operator := parts[1]

	// Parse second number
	num2, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid second number: %v", err)
	}

	// Perform calculation
	var result float64
	switch operator {
	case "+":
		result = num1 + num2
	case "-":
		result = num1 - num2
	case "*":
		result = num1 * num2
	case "/":
		if num2 == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		result = num1 / num2
	default:
		return 0, fmt.Errorf("unsupported operator: %s", operator)
	}

	logger.Info("Calculated", expression, "=", result)
	return result, nil
}
