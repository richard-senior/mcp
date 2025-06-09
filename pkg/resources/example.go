package resources

import (
	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

// ExampleResource returns an example resource
func ExampleResource() protocol.Resource {
	logger.Info("Creating example resource")
	return protocol.Resource{
		Name:        "example_documentation",
		Description: "Example documentation resource for MCP",
		Type:        "documentation",
		Metadata: map[string]interface{}{
			"version": "1.0.0",
			"format":  "markdown",
			"topics":  []string{"mcp", "protocol", "example"},
		},
	}
}

// WeatherResource returns a weather data resource
func WeatherResource() protocol.Resource {
	logger.Info("Creating weather resource")
	return protocol.Resource{
		Name:        "weather_data",
		Description: "Historical weather data resource",
		Type:        "dataset",
		Metadata: map[string]interface{}{
			"regions":    []string{"US", "Europe", "Asia"},
			"timeRange":  "2020-2025",
			"dataPoints": []string{"temperature", "humidity", "precipitation"},
		},
	}
}

// GetResources returns all available resources
func GetResources() []protocol.Resource {
	return []protocol.Resource{
		ExampleResource(),
		WeatherResource(),
	}
}

// HandleResourceQuery handles a query to a resource
func HandleResourceQuery(resourceName string, query interface{}) (interface{}, error) {
	logger.Info("Handling resource query for:", resourceName)

	// In a real implementation, this would query the actual resource
	// For now, we'll just return some example data
	switch resourceName {
	case "example_documentation":
		return map[string]interface{}{
			"content": "# MCP Documentation\n\nThis is example documentation for the Model Context Protocol.",
			"format":  "markdown",
		}, nil
	case "weather_data":
		return map[string]interface{}{
			"location": "San Francisco",
			"current": map[string]interface{}{
				"temperature": 72,
				"humidity":    65,
				"conditions":  "Partly Cloudy",
			},
			"forecast": []map[string]interface{}{
				{
					"day":         "Tomorrow",
					"temperature": 75,
					"conditions":  "Sunny",
				},
				{
					"day":         "Day after",
					"temperature": 68,
					"conditions":  "Cloudy",
				},
			},
		}, nil
	default:
		return nil, protocol.NewJsonRpcErrorResponse(
			protocol.ErrInvalidParams,
			"Resource not found: "+resourceName,
			nil,
			nil,
		).Error
	}
}
