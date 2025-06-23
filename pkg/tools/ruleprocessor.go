package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

// RulesRegistry represents the registry of rules
type RulesRegistry struct {
	Rules []RuleInfo `json:"rules"`
}

// RuleInfo represents information about a rule in the registry
type RuleInfo struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Path        string   `json:"path"`
	Globs       []string `json:"globs"`
	AlwaysApply bool     `json:"alwaysApply"`
}

// RuleContent represents the content of a rule
type RuleContent struct {
	ID          string                   `json:"id"`
	Description string                   `json:"description"`
	Content     string                   `json:"content"`
	Filters     []map[string]interface{} `json:"filters"`
	Actions     []map[string]interface{} `json:"actions"`
	Examples    []map[string]interface{} `json:"examples"`
	Metadata    map[string]interface{}   `json:"metadata"`
}

// RuleResult represents the result of applying a rule to a file
type RuleResult struct {
	RuleID      string   `json:"ruleId"`
	Passed      bool     `json:"passed"`
	Violations  []string `json:"violations,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// LoadRulesRegistry loads the rules registry from a file
func LoadRulesRegistry(path string) (*RulesRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules registry: %w", err)
	}

	var registry RulesRegistry
	err = json.Unmarshal(data, &registry)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rules registry: %w", err)
	}

	return &registry, nil
}

// GetRuleContent loads the content of a rule
func GetRuleContent(ruleID string, registryPath string) (*RuleContent, error) {
	// Load the registry to get the rule path
	registry, err := LoadRulesRegistry(registryPath)
	if err != nil {
		return nil, err
	}

	// Find the rule in the registry
	var rulePath string
	for _, rule := range registry.Rules {
		if rule.ID == ruleID {
			rulePath = rule.Path
			break
		}
	}

	if rulePath == "" {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	// Load the rule content
	data, err := os.ReadFile(rulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %w", err)
	}

	// Parse the rule content
	content := string(data)

	// Extract rule components using regex
	nameRegex := regexp.MustCompile(`<rule>\s*name:\s*([^\n]+)`)
	descRegex := regexp.MustCompile(`description:\s*([^\n]+)`)
	filtersRegex := regexp.MustCompile(`filters:([\s\S]*?)actions:`)
	actionsRegex := regexp.MustCompile(`actions:([\s\S]*?)examples:`)
	examplesRegex := regexp.MustCompile(`examples:([\s\S]*?)metadata:`)
	metadataRegex := regexp.MustCompile(`metadata:([\s\S]*?)</rule>`)

	nameMatch := nameRegex.FindStringSubmatch(content)
	descMatch := descRegex.FindStringSubmatch(content)

	// These are not used yet but will be needed for a more complete implementation
	_ = filtersRegex.FindStringSubmatch(content)
	_ = actionsRegex.FindStringSubmatch(content)
	_ = examplesRegex.FindStringSubmatch(content)
	_ = metadataRegex.FindStringSubmatch(content)

	if len(nameMatch) < 2 || len(descMatch) < 2 {
		return nil, fmt.Errorf("failed to parse rule content")
	}

	// Create a simplified rule content object
	ruleContent := &RuleContent{
		ID:          strings.TrimSpace(nameMatch[1]),
		Description: strings.TrimSpace(descMatch[1]),
		Content:     content,
		Filters:     []map[string]interface{}{},
		Actions:     []map[string]interface{}{},
		Examples:    []map[string]interface{}{},
		Metadata:    map[string]interface{}{},
	}

	return ruleContent, nil
}

// IsFileMatchingRule checks if a file matches a rule's globs
func IsFileMatchingRule(filePath string, rule RuleInfo) bool {
	// If the rule always applies, return true
	if rule.AlwaysApply {
		return true
	}

	// Check if the file matches any of the rule's globs
	for _, glob := range rule.Globs {
		matched, err := filepath.Match(glob, filePath)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// ApplyRuleToFile applies a rule to a file
func ApplyRuleToFile(filePath string, rule *RuleContent) (*RuleResult, error) {
	// Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// This is a simplified implementation that just checks for basic patterns
	// In a real implementation, you would parse the rule's filters and actions
	// and apply them to the file content

	// For now, we'll just check if the file contains any patterns that might violate the rule
	result := &RuleResult{
		RuleID: rule.ID,
		Passed: true,
	}

	// Example: Check for error handling patterns if this is an error handling rule
	if strings.Contains(rule.ID, "error_handling") {
		// Look for error handling patterns that might be problematic
		if strings.Contains(content, "if err != nil {") &&
			strings.Contains(content, "return errors.New(") {
			result.Passed = false
			result.Violations = append(result.Violations, "Found potential error handling issue: creating new error instead of wrapping")
			result.Suggestions = append(result.Suggestions, "Use fmt.Errorf(\"context: %w\", err) to wrap errors")
		}
	}

	// Example: Check for receiver naming if this is a receiver naming rule
	if strings.Contains(rule.ID, "receiver_names") {
		// Look for receiver names that might be problematic
		if strings.Contains(content, "func (this ") ||
			strings.Contains(content, "func (self ") {
			result.Passed = false
			result.Violations = append(result.Violations, "Found non-idiomatic receiver names: 'this' or 'self'")
			result.Suggestions = append(result.Suggestions, "Use short, consistent receiver names derived from the type name")
		}
	}

	return result, nil
}

// ProcessRulesProcessorRequest handles rules processor related requests
func ProcessRulesProcessorRequest(query string, requestID string) (*protocol.JsonRpcResponse, error) {
	if strings.HasPrefix(query, "process_rules ") {
		// Parse the process rules command
		// Format: process_rules <registry_path> <file_path>
		parts := strings.SplitN(query, " ", 3)
		if len(parts) < 3 {
			ret := protocol.NewJsonRpcErrorResponse(-32602, "Invalid process_rules command format", "", "")
			return ret, nil
		}

		registryPath := parts[1]
		filePath := parts[2]

		// If registry path is "default", use the default registry path
		if registryPath == "default" {
			var err error
			registryPath, err = GetRegistryPath()
			if err != nil {
				logger.Error("Failed to get default registry path", err)
				ret := protocol.NewJsonRpcErrorResponse(-32603, "Failed to get registry path", "", "")
				return ret, nil
			}
		}

		// Load the rules registry
		registry, err := LoadRulesRegistry(registryPath)
		if err != nil {
			logger.Error("Failed to load rules registry", err)
			ret := protocol.NewJsonRpcErrorResponse(-32603, "Failed to load rules registry", "", "")
			return ret, nil
		}

		// Find applicable rules for the file
		var applicableRules []RuleInfo
		for _, rule := range registry.Rules {
			if IsFileMatchingRule(filePath, rule) {
				applicableRules = append(applicableRules, rule)
			}
		}

		// Apply each rule to the file
		var results []RuleResult
		for _, rule := range applicableRules {
			// Get the rule content
			ruleContent, err := GetRuleContent(rule.ID, registryPath)
			if err != nil {
				logger.Warn("Failed to get rule content", rule.ID, err)
				continue
			}

			// Apply the rule to the file
			result, err := ApplyRuleToFile(filePath, ruleContent)
			if err != nil {
				logger.Warn("Failed to apply rule", rule.ID, err)
				continue
			}

			results = append(results, *result)
		}

		// Create success response
		response, err := protocol.NewJsonRpcResponse(results, "")
		return response, nil

	} else if strings.HasPrefix(query, "get_rule_content ") {
		// Parse the get rule content command
		parts := strings.SplitN(query, " ", 3)
		if len(parts) < 3 {
			ret := protocol.NewJsonRpcErrorResponse(-32603, "Invalid rule content", "", "")
			return ret, nil
		}

		ruleID := parts[1]
		registryPath := parts[2]

		// If registry path is "default", use the default registry path
		if registryPath == "default" {
			var err error
			registryPath, err = GetRegistryPath()
			if err != nil {
				logger.Error("Failed to get default registry path", err)
				ret := protocol.NewJsonRpcErrorResponse(-32603, "Failed to get default registry", "", "")
				return ret, nil
			}
		}

		// Get the rule content
		ruleContent, err := GetRuleContent(ruleID, registryPath)
		if err != nil {
			logger.Error("Failed to get rule content", err)
			ret := protocol.NewJsonRpcErrorResponse(-32603, "Failed to load rule content", "", "")
			return ret, nil
		}

		ctx := map[string]any{
			"ruleContent": ruleContent,
			"ruleID":      ruleID,
		}

		response, err := protocol.NewJsonRpcResponse(ctx, "")
		return response, nil
	}

	// If we get here, it's not a rules processor command
	return nil, fmt.Errorf("not a rules processor command")
}

// GetRegistryPath returns the path to the rules registry
func GetRegistryPath() (string, error) {
	// For now, simply return the hardcoded path as requested
	return "/Users/richard/.mcp/registry.json", nil
}

// Helper functions
func countPassedRules(results []RuleResult) int {
	count := 0
	for _, result := range results {
		if result.Passed {
			count++
		}
	}
	return count
}

func countFailedRules(results []RuleResult) int {
	count := 0
	for _, result := range results {
		if !result.Passed {
			count++
		}
	}
	return count
}
