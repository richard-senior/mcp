package processor

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
)

// PromptVariable represents a variable in a prompt template
type PromptVariable struct {
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// StoredPrompt represents a prompt stored in the registry
type StoredPrompt struct {
	ID          string                    `json:"id"`
	Description string                    `json:"description,omitempty"`
	Content     string                    `json:"content"`
	Tags        []string                  `json:"tags"`
	Variables   map[string]PromptVariable `json:"variables"`
	Metadata    map[string]interface{}    `json:"metadata"`
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
func (pr *PromptRegistry) GetPrompt(id string) (*StoredPrompt, error) {
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

	var prompt StoredPrompt
	err = json.Unmarshal(data, &prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt file: %w", err)
	}

	return &prompt, nil
}

// SavePrompt saves a prompt to the registry
func (pr *PromptRegistry) SavePrompt(prompt *StoredPrompt) error {
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
func (pr *PromptRegistry) ListPrompts() ([]StoredPrompt, error) {
	var prompts []StoredPrompt

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
	samplePrompt := &StoredPrompt{
		ID:          "sample",
		Description: "A sample prompt for testing",
		Content:     "This is a sample prompt with {{variable1}} and {{variable2}}.",
		Tags:        []string{"sample", "test"},
		Variables: map[string]PromptVariable{
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
func ProcessPromptRegistryRequest(query string, requestID string) ([]byte, error) {
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
			return CreateErrorResponse("prompt_registry_error", err.Error(), requestID)
		}

		response := MCPResponse{
			RequestID: requestID,
			Context: map[string]interface{}{
				"prompts": prompts,
				"count":   len(prompts),
			},
			Metadata: map[string]interface{}{
				"version": "1.0.0",
			},
		}

		result, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal response", err)
			return CreateErrorResponse("internal_error", "Failed to create response", requestID)
		}

		return result, nil
	} else if strings.HasPrefix(query, "get_prompt ") {
		// Get a specific prompt
		id := strings.TrimPrefix(query, "get_prompt ")
		prompt, err := registry.GetPrompt(id)
		if err != nil {
			logger.Error("Failed to get prompt", err)
			return CreateErrorResponse("prompt_registry_error", err.Error(), requestID)
		}

		response := MCPResponse{
			RequestID: requestID,
			Context: map[string]interface{}{
				"prompt": prompt,
			},
			Metadata: map[string]interface{}{
				"version": "1.0.0",
			},
		}

		result, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal response", err)
			return CreateErrorResponse("internal_error", "Failed to create response", requestID)
		}

		return result, nil
	}

	// If we get here, it's not a prompt registry command
	return nil, fmt.Errorf("not a prompt registry command")
}
