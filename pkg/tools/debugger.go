package tools

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/richard-senior/mcp/pkg/debugger"
	"github.com/richard-senior/mcp/pkg/protocol"
)

var (
	debugClient *debugger.Client
	debugMutex  sync.Mutex
)

func getDebugClient() *debugger.Client {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	if debugClient == nil {
		debugClient = debugger.NewClient()
	}
	return debugClient
}

// GoDebugLaunchTool creates a tool for launching Go programs for debugging
func GoDebugLaunchTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_launch",
		Description: `Launch a Go program for debugging with Delve debugger.
		This tool starts a new debugging session for a Go executable.`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"program": {
					Type:        "string",
					Description: "Path to the Go executable to debug",
				},
				"args": {
					Type:        "array",
					Description: "Command line arguments for the program (optional)",
				},
			},
			Required: []string{"program"},
		},
	}
}

func HandleGoDebugLaunch(params any) (any, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	program, ok := paramsMap["program"].(string)
	if !ok || program == "" {
		return nil, fmt.Errorf("program path is required")
	}

	var args []string
	if argsInterface, exists := paramsMap["args"]; exists {
		if argsList, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsList {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}
	}

	client := getDebugClient()
	
	// Create a timeout context for the entire operation
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	
	// Run the launch in a goroutine with timeout
	responseChan := make(chan debugger.LaunchResponse, 1)
	
	go func() {
		response := client.LaunchProgram(program, args)
		responseChan <- response
	}()
	
	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("debug launch operation timed out after 50 seconds")
	}
}

// GoDebugContinueTool creates a tool for continuing program execution
func GoDebugContinueTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_continue",
		Description: `Continue execution of the debugged program until next breakpoint or program termination.`,
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

func HandleGoDebugContinue(params any) (any, error) {
	client := getDebugClient()
	
	// Create a timeout context for the operation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	
	// Run the continue in a goroutine with timeout
	responseChan := make(chan debugger.ContinueResponse, 1)
	
	go func() {
		response := client.Continue()
		responseChan <- response
	}()
	
	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("debug continue operation timed out after 100 seconds")
	}
}

// GoDebugStepTool creates a tool for stepping into functions
func GoDebugStepTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_step",
		Description: `Execute a single instruction, stepping into function calls.`,
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

func HandleGoDebugStep(params any) (any, error) {
	client := getDebugClient()
	response := client.Step()
	return response, nil
}

// GoDebugStepOverTool creates a tool for stepping over functions
func GoDebugStepOverTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_step_over",
		Description: `Execute the next instruction, stepping over function calls.`,
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

func HandleGoDebugStepOver(params any) (any, error) {
	client := getDebugClient()
	response := client.StepOver()
	return response, nil
}

// GoDebugStepOutTool creates a tool for stepping out of functions
func GoDebugStepOutTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_step_out",
		Description: `Execute until the current function returns.`,
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

func HandleGoDebugStepOut(params any) (any, error) {
	client := getDebugClient()
	response := client.StepOut()
	return response, nil
}

// GoDebugSetBreakpointTool creates a tool for setting breakpoints
func GoDebugSetBreakpointTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_set_breakpoint",
		Description: `Set a breakpoint at the specified file and line number.`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"file": {
					Type:        "string",
					Description: "Path to the source file",
				},
				"line": {
					Type:        "integer",
					Description: "Line number to set the breakpoint",
				},
			},
			Required: []string{"file", "line"},
		},
	}
}

func HandleGoDebugSetBreakpoint(params any) (any, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	file, ok := paramsMap["file"].(string)
	if !ok || file == "" {
		return nil, fmt.Errorf("file path is required")
	}

	var line int
	if lineFloat, ok := paramsMap["line"].(float64); ok {
		line = int(lineFloat)
	} else if lineStr, ok := paramsMap["line"].(string); ok {
		var err error
		line, err = strconv.Atoi(lineStr)
		if err != nil {
			return nil, fmt.Errorf("invalid line number: %v", err)
		}
	} else {
		return nil, fmt.Errorf("line number is required")
	}

	client := getDebugClient()
	response := client.SetBreakpoint(file, line)
	return response, nil
}

// GoDebugListBreakpointsTool creates a tool for listing breakpoints
func GoDebugListBreakpointsTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_list_breakpoints",
		Description: `List all currently set breakpoints.`,
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

func HandleGoDebugListBreakpoints(params any) (any, error) {
	client := getDebugClient()
	response := client.ListBreakpoints()
	return response, nil
}

// GoDebugRemoveBreakpointTool creates a tool for removing breakpoints
func GoDebugRemoveBreakpointTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_remove_breakpoint",
		Description: `Remove a breakpoint by its ID.`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"id": {
					Type:        "integer",
					Description: "Breakpoint ID to remove",
				},
			},
			Required: []string{"id"},
		},
	}
}

func HandleGoDebugRemoveBreakpoint(params any) (any, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	var id int
	if idFloat, ok := paramsMap["id"].(float64); ok {
		id = int(idFloat)
	} else if idStr, ok := paramsMap["id"].(string); ok {
		var err error
		id, err = strconv.Atoi(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid breakpoint ID: %v", err)
		}
	} else {
		return nil, fmt.Errorf("breakpoint ID is required")
	}

	client := getDebugClient()
	response := client.RemoveBreakpoint(id)
	return response, nil
}

// GoDebugEvalVariableTool creates a tool for evaluating variables
func GoDebugEvalVariableTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_eval_variable",
		Description: `Evaluate a variable expression in the current debugging context.`,
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.ToolProperty{
				"name": {
					Type:        "string",
					Description: "Variable name or expression to evaluate",
				},
				"depth": {
					Type:        "integer",
					Description: "Depth of variable expansion (optional, default 1)",
				},
			},
			Required: []string{"name"},
		},
	}
}

func HandleGoDebugEvalVariable(params any) (any, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters format")
	}

	name, ok := paramsMap["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("variable name is required")
	}

	depth := 1
	if depthFloat, ok := paramsMap["depth"].(float64); ok {
		depth = int(depthFloat)
	} else if depthStr, ok := paramsMap["depth"].(string); ok {
		var err error
		depth, err = strconv.Atoi(depthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid depth: %v", err)
		}
	}

	client := getDebugClient()
	response := client.EvalVariable(name, depth)
	return response, nil
}

// GoDebugCloseTool creates a tool for closing debug sessions
func GoDebugCloseTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_close",
		Description: `Close the current debugging session and terminate the debugged program.`,
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

func HandleGoDebugClose(params any) (any, error) {
	client := getDebugClient()
	response, err := client.Close()
	if err != nil {
		return nil, err
	}
	
	// Reset the client for next session
	debugMutex.Lock()
	debugClient = nil
	debugMutex.Unlock()
	
	return response, nil
}

// GoDebugGetOutputTool creates a tool for getting program output
func GoDebugGetOutputTool() protocol.Tool {
	return protocol.Tool{
		Name: "go_debug_get_output",
		Description: `Get the captured stdout and stderr output from the debugged program.`,
		InputSchema: protocol.InputSchema{
			Type:       "object",
			Properties: map[string]protocol.ToolProperty{},
		},
	}
}

func HandleGoDebugGetOutput(params any) (any, error) {
	client := getDebugClient()
	response := client.GetDebuggerOutput()
	return response, nil
}
