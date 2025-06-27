package prompts

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

// PromptRegistry manages the storage and retrieval of prompts for MCP
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

	registry := &PromptRegistry{
		baseDir: baseDir,
	}

	// Create sample prompts if directory is empty
	registry.ensureSamplePrompts()

	return registry
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

// ListPrompts returns a list of all available prompts
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

// ensureSamplePrompts creates sample prompts if they don't exist
func (pr *PromptRegistry) ensureSamplePrompts() {
	// Check if specific sample prompts exist, create them if they don't
	samplePrompts := []*protocol.Prompt{
		{
			ID:          "code-review",
			Name:        "Code Review",
			Description: "Review code for best practices, bugs, and improvements",
			Content:     "Please review the following {{language}} code for:\n- Best practices\n- Potential bugs\n- Performance improvements\n- Security issues\n\nCode:\n```{{language}}\n{{code}}\n```",
			Tags:        []string{"development", "review", "code-quality"},
			Variables: map[string]protocol.PromptArgument{
				"language": {
					Description: "Programming language of the code",
					Required:    true,
				},
				"code": {
					Description: "The code to review",
					Required:    true,
				},
			},
			Metadata: map[string]interface{}{
				"author":   "MCP Server",
				"version":  "1.0.0",
				"category": "development",
			},
		},
		{
			ID:          "explain-concept",
			Name:        "Explain Technical Concept",
			Description: "Explain a technical concept in simple terms",
			Content:     "Please explain {{concept}} in simple terms that a {{audience}} would understand. Include:\n- What it is\n- Why it's important\n- How it works\n- Real-world examples\n\nAdjust the explanation level for: {{audience}}",
			Tags:        []string{"education", "explanation", "technical"},
			Variables: map[string]protocol.PromptArgument{
				"concept": {
					Description: "The technical concept to explain",
					Required:    true,
				},
				"audience": {
					Description: "Target audience (e.g., beginner, intermediate, expert)",
					Required:    false,
				},
			},
			Metadata: map[string]interface{}{
				"author":   "MCP Server",
				"version":  "1.0.0",
				"category": "education",
			},
		},
		{
			ID:          "aws-architecture",
			Name:        "AWS Architecture Review",
			Description: "Review and suggest improvements for AWS architecture",
			Content:     "Please review this AWS architecture for {{use_case}}:\n\n{{architecture_description}}\n\nProvide feedback on:\n- Cost optimization\n- Security best practices\n- Scalability\n- Reliability\n- Performance\n\nSuggest specific AWS services and configurations that would improve this architecture.",
			Tags:        []string{"aws", "architecture", "cloud", "review"},
			Variables: map[string]protocol.PromptArgument{
				"use_case": {
					Description: "The use case or application type",
					Required:    true,
				},
				"architecture_description": {
					Description: "Description of the current architecture",
					Required:    true,
				},
			},
			Metadata: map[string]interface{}{
				"author":   "MCP Server",
				"version":  "1.0.0",
				"category": "aws",
			},
		},
		{
			ID:          "sample",
			Name:        "Sample Prompt",
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
		},
	}

	for _, prompt := range samplePrompts {
		// Check if prompt already exists
		_, err := pr.GetPrompt(prompt.ID)
		if err != nil {
			// Prompt doesn't exist, create it
			err := pr.SavePrompt(prompt)
			if err != nil {
				logger.Warn("Failed to create sample prompt", prompt.ID, err)
			} else {
				logger.Info("Created sample prompt", prompt.ID)
			}
		}
	}
}

// Global registry instance
var globalRegistry *PromptRegistry

// GetGlobalRegistry returns the global prompt registry instance
func GetGlobalRegistry() *PromptRegistry {
	if globalRegistry == nil {
		globalRegistry = NewPromptRegistry()
	}
	return globalRegistry
}
