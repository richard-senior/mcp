package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
)

// RuleConfig represents configuration for a specific tool's rule format
type RuleConfig struct {
	RuleGlob     string `json:"ruleGlob"`
	RuleDir      string `json:"ruleDir"`
	TargetSubdir string `json:"targetSubdir"`
}

// ToolConfigs maps tool names to their configurations
var ToolConfigs = map[string]RuleConfig{
	"amazonq": {
		RuleGlob:     "q-rulestore-rule.md",
		RuleDir:      "amazonq",
		TargetSubdir: "rules/amazonq",
	},
	"cline": {
		RuleGlob:     "cline-rulestore-rule.md",
		RuleDir:      "cline",
		TargetSubdir: "rules/cline",
	},
	"roo": {
		RuleGlob:     "roo-rulestore-rule.md",
		RuleDir:      "roo",
		TargetSubdir: "rules/roo",
	},
	"cursor": {
		RuleGlob:     "cursor-rulestore-rule.md",
		RuleDir:      "cursor",
		TargetSubdir: "rules/cursor",
	},
}

// RuleMetadata represents the metadata for a rule
type RuleMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Globs       []string `json:"globs"`
	AlwaysApply bool     `json:"alwaysApply"`
}

// Rule represents a complete rule definition
type Rule struct {
	Metadata RuleMetadata             `json:"metadata"`
	Content  string                   `json:"content"`
	Filters  []map[string]string      `json:"filters"`
	Actions  []map[string]interface{} `json:"actions"`
	Examples []map[string]string      `json:"examples"`
	Priority string                   `json:"priority"`
	Version  string                   `json:"version"`
}

// GetMCPBaseDir returns the base directory for MCP assets
func GetMCPBaseDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".mcp")

	// Create the directory if it doesn't exist
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create MCP base directory: %w", err)
	}

	return baseDir, nil
}

// CreateRule creates a new rule file
func CreateRule(toolName, ruleName, description string, globs []string, alwaysApply bool, content string) (string, error) {
	// Validate tool name
	config, ok := ToolConfigs[toolName]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}

	// Get MCP base directory
	baseDir, err := GetMCPBaseDir()
	if err != nil {
		return "", err
	}

	// Create rule directory if it doesn't exist
	targetDir := filepath.Join(baseDir, config.TargetSubdir)
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create rule directory: %w", err)
	}

	// Format the rule file content
	ruleContent := formatRuleContent(ruleName, description, globs, alwaysApply, content)

	// Write the rule file
	rulePath := filepath.Join(targetDir, ruleName+".md")
	err = os.WriteFile(rulePath, []byte(ruleContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write rule file: %w", err)
	}

	// Create or update the registry file
	registryPath := filepath.Join(baseDir, "registry.json")
	err = updateRegistry(registryPath, ruleName, description, rulePath, globs, alwaysApply)
	if err != nil {
		logger.Warn("Failed to update registry", err)
	}

	return rulePath, nil
}

// CreateRuleFromMarkdown creates a rule from a markdown section
func CreateRuleFromMarkdown(toolName, ruleName, markdownContent string) (string, error) {
	// Extract description from the markdown content
	descriptionRegex := regexp.MustCompile(`(?m)^#+\s+(.+)$`)
	descMatch := descriptionRegex.FindStringSubmatch(markdownContent)

	var description string
	if len(descMatch) > 1 {
		description = descMatch[1]
	} else {
		description = ruleName
	}

	// Default globs for Go files
	globs := []string{"**/*.go"}

	// Always apply by default
	alwaysApply := true

	// Create the rule
	return CreateRule(toolName, ruleName, description, globs, alwaysApply, markdownContent)
}

// updateRegistry adds or updates a rule in the registry file
func updateRegistry(registryPath, ruleName, description, rulePath string, globs []string, alwaysApply bool) error {
	var registry RulesRegistry

	// Try to read existing registry
	data, err := os.ReadFile(registryPath)
	if err == nil {
		// Registry exists, parse it
		err = json.Unmarshal(data, &registry)
		if err != nil {
			return fmt.Errorf("failed to parse registry: %w", err)
		}
	} else if os.IsNotExist(err) {
		// Registry doesn't exist, create a new one
		registry = RulesRegistry{
			Rules: []RuleInfo{},
		}
	} else {
		return fmt.Errorf("failed to read registry: %w", err)
	}

	// Check if rule already exists in registry
	found := false
	for i, rule := range registry.Rules {
		if rule.ID == ruleName {
			// Update existing rule
			registry.Rules[i] = RuleInfo{
				ID:          ruleName,
				Description: description,
				Path:        rulePath,
				Globs:       globs,
				AlwaysApply: alwaysApply,
			}
			found = true
			break
		}
	}

	if !found {
		// Add new rule
		registry.Rules = append(registry.Rules, RuleInfo{
			ID:          ruleName,
			Description: description,
			Path:        rulePath,
			Globs:       globs,
			AlwaysApply: alwaysApply,
		})
	}

	// Write updated registry
	data, err = json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	err = os.WriteFile(registryPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// formatRuleContent formats the rule content as a markdown file
func formatRuleContent(name, description string, globs []string, alwaysApply bool, content string) string {
	// Format globs as YAML list
	globsYAML := ""
	for _, glob := range globs {
		globsYAML += fmt.Sprintf("  - \"%s\"\n", glob)
	}

	// Create the frontmatter
	frontmatter := fmt.Sprintf("---\ndescription: %s\nglobs:\n%salwaysApply: %t\n---\n",
		description, globsYAML, alwaysApply)

	// Format the rule content
	ruleContent := fmt.Sprintf("%s# %s\n\n%s\n\n<rule>\nname: %s\ndescription: %s\n",
		frontmatter, name, description, name, description)

	// Add filters section
	ruleContent += "filters:\n  - type: file_extension\n    pattern: \"\\\\.go$\"\n"

	// Add actions section
	ruleContent += "actions:\n  - type: suggest\n    message: |\n      Add your suggestion message here.\n"

	// Add examples section
	ruleContent += "examples:\n  - input: |\n      // Example input code\n    output: \"Example output or message\"\n"

	// Add metadata section
	ruleContent += "metadata:\n  priority: medium\n  version: 1.0\n</rule>\n"

	return ruleContent
}

// ListRules lists all rules for a specific tool
func ListRules(toolName string) ([]string, error) {
	// Validate tool name
	config, ok := ToolConfigs[toolName]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	// Get MCP base directory
	baseDir, err := GetMCPBaseDir()
	if err != nil {
		return nil, err
	}

	targetDir := filepath.Join(baseDir, config.TargetSubdir)

	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	// List all rule files
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule directory: %w", err)
	}

	var rules []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			rules = append(rules, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}

	return rules, nil
}

// GetRegistryPath returns the path to the registry file
func GetRegistryPath() (string, error) {
	baseDir, err := GetMCPBaseDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "registry.json"), nil
}

// ProcessRuleCreatorRequest handles rule creator related requests
func ProcessRuleCreatorRequest(query string, requestID string) ([]byte, error) {
	if strings.HasPrefix(query, "create_rule ") {
		// Parse the create rule command
		// Format: create_rule <tool> <name> <description> <globs> <alwaysApply> <content>
		parts := strings.SplitN(query, " ", 7)
		if len(parts) < 7 {
			return CreateErrorResponse("rule_creator_error", "Invalid create_rule command format", requestID)
		}

		toolName := parts[1]
		ruleName := parts[2]
		description := parts[3]
		globsStr := parts[4]
		alwaysApplyStr := parts[5]
		content := parts[6]

		// Parse globs
		globs := strings.Split(globsStr, ",")

		// Parse alwaysApply
		alwaysApply := alwaysApplyStr == "true"

		// Create the rule
		rulePath, err := CreateRule(toolName, ruleName, description, globs, alwaysApply, content)
		if err != nil {
			logger.Error("Failed to create rule", err)
			return CreateErrorResponse("rule_creator_error", err.Error(), requestID)
		}

		// Return success response
		response := MCPResponse{
			RequestID: requestID,
			Context: map[string]interface{}{
				"rulePath": rulePath,
				"ruleName": ruleName,
				"tool":     toolName,
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
	} else if strings.HasPrefix(query, "create_rule_from_markdown ") {
		// Parse the create rule from markdown command
		// Format: create_rule_from_markdown <tool> <name> <markdown_content>
		parts := strings.SplitN(query, " ", 4)
		if len(parts) < 4 {
			return CreateErrorResponse("rule_creator_error", "Invalid create_rule_from_markdown command format", requestID)
		}

		toolName := parts[1]
		ruleName := parts[2]
		markdownContent := parts[3]

		// Create the rule
		rulePath, err := CreateRuleFromMarkdown(toolName, ruleName, markdownContent)
		if err != nil {
			logger.Error("Failed to create rule from markdown", err)
			return CreateErrorResponse("rule_creator_error", err.Error(), requestID)
		}

		// Return success response
		response := MCPResponse{
			RequestID: requestID,
			Context: map[string]interface{}{
				"rulePath": rulePath,
				"ruleName": ruleName,
				"tool":     toolName,
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
	} else if strings.HasPrefix(query, "list_rules ") {
		// Parse the list rules command
		parts := strings.SplitN(query, " ", 2)
		if len(parts) < 2 {
			return CreateErrorResponse("rule_creator_error", "Invalid list_rules command format", requestID)
		}

		toolName := parts[1]

		// List the rules
		rules, err := ListRules(toolName)
		if err != nil {
			logger.Error("Failed to list rules", err)
			return CreateErrorResponse("rule_creator_error", err.Error(), requestID)
		}

		// Return success response
		response := MCPResponse{
			RequestID: requestID,
			Context: map[string]interface{}{
				"rules": rules,
				"count": len(rules),
				"tool":  toolName,
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
	} else if strings.HasPrefix(query, "get_registry_path") {
		// Get the registry path
		registryPath, err := GetRegistryPath()
		if err != nil {
			logger.Error("Failed to get registry path", err)
			return CreateErrorResponse("rule_creator_error", err.Error(), requestID)
		}

		// Return success response
		response := MCPResponse{
			RequestID: requestID,
			Context: map[string]interface{}{
				"registryPath": registryPath,
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

	// If we get here, it's not a rule creator command
	return nil, fmt.Errorf("not a rule creator command")
}
