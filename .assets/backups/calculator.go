package processor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
)

// CalculateResult performs a simple calculation based on the input expression
func CalculateResult(expression string) (float64, error) {
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
