package tools

import (
	"time"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

// DateTimeTool returns the current date and time
func DateTimeTool() protocol.Tool {
	return protocol.Tool{
		Name:        "get_datetime",
		Description: "Returns the current date and time",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"format": {
					Type:        "string",
					Description: "The format of the datetime to be returned such as 2006-01-02T15:04:05Z07:00",
				},
			},
			Required: []string{},
		},
	}
}

// HandleDateTimeTool handles the date time tool invocation
func HandleDateTimeTool(params any) (any, error) {
	logger.Info("Handling datetime tool invocation")

	var format string = time.RFC3339

	// Parse parameters if provided
	if params != nil {
		paramsMap, ok := params.(map[string]any)
		if ok {
			if fmt, ok := paramsMap["format"].(string); ok && fmt != "" {
				format = fmt
			}
		}
	}

	// Return the current date and time
	return map[string]any{
		"datetime": time.Now().Format(format),
	}, nil
}
