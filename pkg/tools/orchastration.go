package tools

import (
	"fmt"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

func NewOrchastrationTool() protocol.Tool {
	return protocol.Tool{
		Name: "meme_tool",
		Description: `
		Creates an Agentic and Generative AI agent which can be used as a delegate to achieve sub-tasks in a larger operation
		`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"prompt": {
					Type: "string",
					Description: `
					The prompt to pass to the agent
					`,
				},
				"basepath": {
					Type:        "string",
					Description: "The directory in which this agent should be initialised. That directory may contain rules/prompts or other metadata for use by this delegate agent",
				},
			},
			Required: []string{"basepath", "prompt"},
		},
	}
}

// TODO this!
// given a raster image, creates a cheezy meme for demonstration purposes
func HandleOrchastrationTool(params any) (any, error) {

	if params == nil {
		return nil, fmt.Errorf("no params given")
	}
	// Convert params to map[string]any
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Couldn't format the parmeters as a map of strings")
	}
	prompt, ok := paramsMap["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("No prompt parameter was sent")
	}
	basepath, ok := paramsMap["basepath"].(string)
	if !ok {
		return nil, fmt.Errorf("No basepath parameter was sent")
	}

	logger.Debug("foo", prompt, basepath)

	return map[string]any{
		"location": "foo",
	}, nil
}
