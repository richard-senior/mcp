package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

// Configuration constants
const (
	// Location for storing thoughts data
	THOUGHTS_DATA_DIR  = "~/.mcp/thoughts"
	THOUGHTS_DATA_FILE = "thoughts.json"
	// Auto-save interval in seconds
	AUTO_SAVE_INTERVAL = 30
)

// ThoughtData represents a single thought in the sequential thinking process
type ThoughtData struct {
	Thought           string    `json:"thought"`
	ThoughtNumber     int       `json:"thoughtNumber"`
	TotalThoughts     int       `json:"totalThoughts"`
	NextThoughtNeeded bool      `json:"nextThoughtNeeded"`
	IsRevision        bool      `json:"isRevision,omitempty"`
	RevisesThought    int       `json:"revisesThought,omitempty"`
	BranchFromThought int       `json:"branchFromThought,omitempty"`
	BranchID          string    `json:"branchId,omitempty"`
	NeedsMoreThoughts bool      `json:"needsMoreThoughts,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
}

// ThoughtResponse is the structure returned to the client
type ThoughtResponse struct {
	ThoughtNumber        int      `json:"thoughtNumber"`
	TotalThoughts        int      `json:"totalThoughts"`
	NextThoughtNeeded    bool     `json:"nextThoughtNeeded"`
	Branches             []string `json:"branches"`
	ThoughtHistoryLength int      `json:"thoughtHistoryLength"`
}

// ErrorResponse is returned when an error occurs
type ErrorResponse struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

// PersistentData is the structure that will be saved to the JSON file
type PersistentData struct {
	ThoughtHistory []ThoughtData              `json:"thoughtHistory"`
	Branches       map[string][]ThoughtData   `json:"branches"`
	Sessions       map[string][]ThoughtData   `json:"sessions"`
	Topics         map[string][]string        `json:"topics"`
	LastUpdated    time.Time                  `json:"lastUpdated"`
	Metadata       map[string]json.RawMessage `json:"metadata"`
}

// SequentialThinking manages the sequential thinking process
type SequentialThinking struct {
	ThoughtHistory []ThoughtData
	Branches       map[string][]ThoughtData
	Sessions       map[string][]ThoughtData
	Topics         map[string][]string
	Metadata       map[string]json.RawMessage
	LastUpdated    time.Time
	mutex          sync.RWMutex
	dataFile       string
	autoSaveTimer  *time.Timer
}

// NewThoughtsTool creates a new thoughts tool
func NewThoughtsTool() protocol.Tool {
	return protocol.Tool{
		Name: "thoughts",
		Description: `
A detailed tool for dynamic and reflective problem-solving through thoughts.
This tool helps analyze problems through a flexible thinking process that can adapt and evolve.
Each thought can build on, question, or revise previous insights as understanding deepens.

When to use this tool:
- Breaking down complex problems into steps
- Planning and design with room for revision
- Analysis that might need course correction
- Problems where the full scope might not be clear initially
- Problems that require a multi-step solution
- Tasks that need to maintain context over multiple steps
- Situations where irrelevant information needs to be filtered out

Key features:
- You can adjust total_thoughts up or down as you progress
- You can question or revise previous thoughts
- You can add more thoughts even after reaching what seemed like the end
- You can express uncertainty and explore alternative approaches
- Not every thought needs to build linearly - you can branch or backtrack
- Generates a solution hypothesis
- Verifies the hypothesis based on the Chain of Thought steps
- Repeats the process until satisfied
- Provides a correct answer

Parameters explained:
- thought: Your current thinking step, which can include:
  * Regular analytical steps
  * Revisions of previous thoughts
  * Questions about previous decisions
  * Realizations about needing more analysis
  * Changes in approach
  * Hypothesis generation
  * Hypothesis verification
- nextThoughtNeeded: True if you need more thinking, even if at what seemed like the end
- thoughtNumber: Current number in sequence (can go beyond initial total if needed)
- totalThoughts: Current estimate of thoughts needed (can be adjusted up/down)
- isRevision: A boolean indicating if this thought revises previous thinking
- revisesThought: If isRevision is true, which thought number is being reconsidered
- branchFromThought: If branching, which thought number is the branching point
- branchId: Identifier for the current branch (if any)
- needsMoreThoughts: If reaching end but realizing more thoughts needed

This tool should be used when:
- starting a new chat session, to see if you have any previous thoughts on the subject
- you feel that the has asked the same thing multiple times and is not getting the required answer
- the user has asked you to try harder
- whenever you encounter a new subject or project etc.
- when you are not sure about the answer and want to try again
`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"thought": {
					Type:        "string",
					Description: "Your current thinking step",
				},
				"nextThoughtNeeded": {
					Type:        "boolean",
					Description: "Whether another thought step is needed",
				},
				"thoughtNumber": {
					Type:        "integer",
					Description: "Current thought number",
				},
				"totalThoughts": {
					Type:        "integer",
					Description: "Estimated total thoughts needed",
				},
				"isRevision": {
					Type:        "boolean",
					Description: "Whether this revises previous thinking",
				},
				"revisesThought": {
					Type:        "integer",
					Description: "Which thought is being reconsidered",
				},
				"branchFromThought": {
					Type:        "integer",
					Description: "Branching point thought number",
				},
				"branchId": {
					Type:        "string",
					Description: "Branch identifier",
				},
				"needsMoreThoughts": {
					Type:        "boolean",
					Description: "If more thoughts are needed",
				},
			},
			Required: []string{"thought", "nextThoughtNeeded", "thoughtNumber", "totalThoughts"},
		},
	}
}

// expandPath expands the tilde in the path to the user's home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Error("Failed to get user home directory: %v", err)
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// NewSequentialThinking creates a new instance of SequentialThinking
func NewSequentialThinking() *SequentialThinking {
	dataDir := expandPath(THOUGHTS_DATA_DIR)
	dataFile := filepath.Join(dataDir, THOUGHTS_DATA_FILE)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Error("Failed to create thoughts data directory: %v", err)
	}

	st := &SequentialThinking{
		ThoughtHistory: []ThoughtData{},
		Branches:       make(map[string][]ThoughtData),
		Sessions:       make(map[string][]ThoughtData),
		Topics:         make(map[string][]string),
		Metadata:       make(map[string]json.RawMessage),
		dataFile:       dataFile,
	}

	// Load existing data if available
	st.loadFromFile()

	// Start auto-save timer
	st.startAutoSave()

	return st
}

// startAutoSave starts the auto-save timer
func (st *SequentialThinking) startAutoSave() {
	st.autoSaveTimer = time.AfterFunc(time.Duration(AUTO_SAVE_INTERVAL)*time.Second, func() {
		st.saveToFile()
		st.startAutoSave() // Restart the timer
	})
}

// loadFromFile loads the data from the JSON file
func (st *SequentialThinking) loadFromFile() {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	// Check if file exists
	if _, err := os.Stat(st.dataFile); os.IsNotExist(err) {
		logger.Info("Thoughts data file does not exist yet, will create on first save")
		return
	}

	// Read file
	data, err := os.ReadFile(st.dataFile)
	if err != nil {
		logger.Error("Failed to read thoughts data file: %v", err)
		return
	}

	// Parse JSON
	var persistentData PersistentData
	if err := json.Unmarshal(data, &persistentData); err != nil {
		logger.Error("Failed to parse thoughts data file: %v", err)
		return
	}

	// Update instance data
	st.ThoughtHistory = persistentData.ThoughtHistory
	st.Branches = persistentData.Branches
	st.Sessions = persistentData.Sessions
	st.Topics = persistentData.Topics
	st.Metadata = persistentData.Metadata
	st.LastUpdated = persistentData.LastUpdated

	logger.Info("Loaded thoughts data from %s (last updated: %v)", st.dataFile, st.LastUpdated)
}

// saveToFile saves the data to the JSON file
func (st *SequentialThinking) saveToFile() {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	// Update timestamp
	st.LastUpdated = time.Now()

	// Prepare data structure
	persistentData := PersistentData{
		ThoughtHistory: st.ThoughtHistory,
		Branches:       st.Branches,
		Sessions:       st.Sessions,
		Topics:         st.Topics,
		Metadata:       st.Metadata,
		LastUpdated:    st.LastUpdated,
	}

	// Convert to JSON
	data, err := json.MarshalIndent(persistentData, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal thoughts data: %v", err)
		return
	}

	// Write to file
	if err := os.WriteFile(st.dataFile, data, 0644); err != nil {
		logger.Error("Failed to write thoughts data file: %v", err)
		return
	}

	logger.Info("Saved thoughts data to %s", st.dataFile)
}

// singleton instance of SequentialThinking
var thinkingInstance *SequentialThinking
var once sync.Once

// GetThinkingInstance returns the singleton instance of SequentialThinking
func GetThinkingInstance() *SequentialThinking {
	once.Do(func() {
		thinkingInstance = NewSequentialThinking()
	})
	return thinkingInstance
}

// ValidateThoughtData validates the input data
func (st *SequentialThinking) ValidateThoughtData(data map[string]interface{}) (ThoughtData, error) {
	thought, ok := data["thought"].(string)
	if !ok || thought == "" {
		return ThoughtData{}, fmt.Errorf("invalid thought: must be a string")
	}

	thoughtNumber, ok := data["thoughtNumber"].(float64)
	if !ok {
		return ThoughtData{}, fmt.Errorf("invalid thoughtNumber: must be a number")
	}

	totalThoughts, ok := data["totalThoughts"].(float64)
	if !ok {
		return ThoughtData{}, fmt.Errorf("invalid totalThoughts: must be a number")
	}

	nextThoughtNeeded, ok := data["nextThoughtNeeded"].(bool)
	if !ok {
		return ThoughtData{}, fmt.Errorf("invalid nextThoughtNeeded: must be a boolean")
	}

	result := ThoughtData{
		Thought:           thought,
		ThoughtNumber:     int(thoughtNumber),
		TotalThoughts:     int(totalThoughts),
		NextThoughtNeeded: nextThoughtNeeded,
		Timestamp:         time.Now(),
	}

	// Optional fields
	if isRevision, ok := data["isRevision"].(bool); ok {
		result.IsRevision = isRevision
	}

	if revisesThought, ok := data["revisesThought"].(float64); ok {
		result.RevisesThought = int(revisesThought)
	}

	if branchFromThought, ok := data["branchFromThought"].(float64); ok {
		result.BranchFromThought = int(branchFromThought)
	}

	if branchID, ok := data["branchId"].(string); ok {
		result.BranchID = branchID
	}

	if needsMoreThoughts, ok := data["needsMoreThoughts"].(bool); ok {
		result.NeedsMoreThoughts = needsMoreThoughts
	}

	return result, nil
}

// FormatThought formats a thought for display
func (st *SequentialThinking) FormatThought(td ThoughtData) string {
	var prefix, context string

	if td.IsRevision {
		prefix = "🔄 Revision"
		context = fmt.Sprintf(" (revising thought %d)", td.RevisesThought)
	} else if td.BranchFromThought > 0 {
		prefix = "🌿 Branch"
		context = fmt.Sprintf(" (from thought %d, ID: %s)", td.BranchFromThought, td.BranchID)
	} else {
		prefix = "💭 Thought"
		context = ""
	}

	header := fmt.Sprintf("%s %d/%d%s", prefix, td.ThoughtNumber, td.TotalThoughts, context)

	// Calculate border length
	headerLen := len(header)
	thoughtLen := len(td.Thought)
	borderLen := max(headerLen, thoughtLen) + 4
	border := strings.Repeat("─", borderLen)

	// Format the thought box
	result := fmt.Sprintf("\n┌%s┐\n│ %s%s │\n├%s┤\n│ %s%s │\n└%s┘",
		border,
		header, strings.Repeat(" ", borderLen-len(header)-2),
		border,
		td.Thought, strings.Repeat(" ", borderLen-len(td.Thought)-2),
		border)

	return result
}

// ProcessThought processes a thought and returns a response
func (st *SequentialThinking) ProcessThought(input map[string]interface{}) (interface{}, error) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	validatedInput, err := st.ValidateThoughtData(input)
	if err != nil {
		return ErrorResponse{
			Error:  err.Error(),
			Status: "failed",
		}, err
	}

	// Adjust total thoughts if needed
	if validatedInput.ThoughtNumber > validatedInput.TotalThoughts {
		validatedInput.TotalThoughts = validatedInput.ThoughtNumber
	}

	// Add to thought history
	st.ThoughtHistory = append(st.ThoughtHistory, validatedInput)

	// Handle branch if applicable
	if validatedInput.BranchFromThought > 0 && validatedInput.BranchID != "" {
		if _, exists := st.Branches[validatedInput.BranchID]; !exists {
			st.Branches[validatedInput.BranchID] = []ThoughtData{}
		}
		st.Branches[validatedInput.BranchID] = append(st.Branches[validatedInput.BranchID], validatedInput)
	}

	// Format and log the thought
	formattedThought := st.FormatThought(validatedInput)
	logger.Info(formattedThought)

	// Save to file after processing (immediate save)
	go st.saveToFile()

	// Prepare response
	branchKeys := make([]string, 0, len(st.Branches))
	for k := range st.Branches {
		branchKeys = append(branchKeys, k)
	}

	return ThoughtResponse{
		ThoughtNumber:        validatedInput.ThoughtNumber,
		TotalThoughts:        validatedInput.TotalThoughts,
		NextThoughtNeeded:    validatedInput.NextThoughtNeeded,
		Branches:             branchKeys,
		ThoughtHistoryLength: len(st.ThoughtHistory),
	}, nil
}

// HandleThoughts is the handler function for the thoughts tool
func HandleThoughts(params any) (any, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	thinkingInstance := GetThinkingInstance()
	response, err := thinkingInstance.ProcessThought(paramsMap)
	if err != nil {
		return response, err
	}
	return response, nil
}

// Helper function for Go versions before 1.21 which don't have built-in max for ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
