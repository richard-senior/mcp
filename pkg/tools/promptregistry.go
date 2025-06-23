package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

func PromptRegistr_y() protocol.Tool {
	return protocol.Tool{
		Name:        "prompt_registry",
		Description: "Implements a registry of prompt data which can be read from or written to",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"id": {
					Type:        "string",
					Description: "The ID of the prompt we want to work with, usually a number",
				},
			},
			Required: []string{"url"},
		},
	}
}

// PromptRegistry manages the storage and retrieval of prompts
type PromptRegistry struct {
	baseDir string
}

// NewPromptRegistry creates a new prompt registry
func NewPromptRegistry() *PromptRegistry {
	// Use ~/.mcp/prompts directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get user home directory", err)
		homeDir = "."
	}

	baseDir := filepath.Join(homeDir, ".mcp", "prompts")

	// Create the directory if it doesn't exist
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		logger.Error("Failed to create prompt registry directory", err)
	}

	return &PromptRegistry{
		baseDir: baseDir,
	}
}

// GetPromptPath returns the file path for a prompt ID
func (pr *PromptRegistry) GetPromptPath(id string) (string, error) {
	// Validate the ID to prevent directory traversal
	if strings.Contains(id, "..") || strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return "", fmt.Errorf("invalid prompt ID format: %s", id)
	}

	return filepath.Join(pr.baseDir, fmt.Sprintf("%s.json", id)), nil
}

// GetPrompt retrieves a prompt by ID
func (pr *PromptRegistry) GetPrompt(id string) (*protocol.Prompt, error) {
	path, err := pr.GetPromptPath(id)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("prompt not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	var prompt protocol.Prompt
	err = json.Unmarshal(data, &prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt file: %w", err)
	}

	return &prompt, nil
}

// SavePrompt saves a prompt to the registry
func (pr *PromptRegistry) SavePrompt(prompt *protocol.Prompt) error {
	if prompt.ID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	path, err := pr.GetPromptPath(prompt.ID)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prompt: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	return nil
}

// ListPrompts returns a list of all prompts in the registry
func (pr *PromptRegistry) ListPrompts() ([]protocol.Prompt, error) {
	var prompts []protocol.Prompt

	err := filepath.WalkDir(pr.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			id := strings.TrimSuffix(d.Name(), ".json")
			prompt, err := pr.GetPrompt(id)
			if err != nil {
				logger.Warn("Failed to read prompt", id, err)
				return nil
			}
			prompts = append(prompts, *prompt)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return prompts, nil
}

// DeletePrompt removes a prompt from the registry
func (pr *PromptRegistry) DeletePrompt(id string) error {
	path, err := pr.GetPromptPath(id)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("prompt not found: %s", id)
		}
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	return nil
}

// CreateSamplePrompt creates a sample prompt in the registry for testing
func (pr *PromptRegistry) CreateSamplePrompt() error {
	samplePrompt := &protocol.Prompt{
		ID:          "sample",
		Description: "A sample prompt for testing",
		Content:     "This is a sample prompt with {{variable1}} and {{variable2}}.",
		Tags:        []string{"sample", "test"},
		Variables: map[string]protocol.PromptArgument{
			"variable1": {
				Description: "First variable",
				Required:    true,
			},
			"variable2": {
				Description: "Second variable",
				Required:    false,
			},
		},
		Metadata: map[string]interface{}{
			"author":  "MCP Application",
			"version": "1.0.0",
		},
	}

	return pr.SavePrompt(samplePrompt)
}

// ProcessPromptRegistryRequest handles prompt registry related requests
func ProcessPromptRegistryRequest(query string, requestID string) (*protocol.JsonRpcResponse, error) {
	registry := NewPromptRegistry()

	if strings.HasPrefix(query, "list_prompts") {
		// Create a sample prompt if none exist
		prompts, _ := registry.ListPrompts()
		if len(prompts) == 0 {
			err := registry.CreateSamplePrompt()
			if err != nil {
				logger.Warn("Failed to create sample prompt", err)
			}
		}

		// List all prompts
		prompts, err := registry.ListPrompts()
		if err != nil {
			logger.Error("Failed to list prompts", err)
			ret := protocol.NewJsonRpcErrorResponse(-32603, "Failed to list prompts", "", "")
			return ret, nil
		}

		ctx := map[string]interface{}{
			"prompts": prompts,
			"count":   len(prompts),
		}

		response, err := protocol.NewJsonRpcResponse(ctx, "")
		return response, nil

	} else if strings.HasPrefix(query, "get_prompt ") {
		// Get a specific prompt
		id := strings.TrimPrefix(query, "get_prompt ")
		prompt, err := registry.GetPrompt(id)
		if err != nil {
			logger.Error("Failed to get prompt", err)
			ret := protocol.NewJsonRpcErrorResponse(-32603, "Failed to get prompt", "", "")
			return ret, nil
		}

		ctx := map[string]interface{}{
			"prompt": prompt,
		}
		response, err := protocol.NewJsonRpcResponse(ctx, "")
		return response, nil
	}

	// If we get here, it's not a prompt registry command
	return nil, fmt.Errorf("not a prompt registry command")
}
