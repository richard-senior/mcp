package debugger

import (
	"fmt"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/richard-senior/mcp/internal/logger"
)

// SetBreakpoint sets a breakpoint at the specified file and line
func (c *Client) SetBreakpoint(file string, line int) BreakpointResponse {
	if c.client == nil {
		return BreakpointResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: "no active debug session",
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	logger.Debug("Setting breakpoint at %s:%d", file, line)
	bp, err := c.client.CreateBreakpoint(&api.Breakpoint{
		File: file,
		Line: line,
	})

	if err != nil {
		return BreakpointResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: fmt.Sprintf("failed to set breakpoint: %v", err),
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state after setting breakpoint: %v", err)
	}

	breakpoint := &Breakpoint{
		DelveBreakpoint: bp,
		ID:              bp.ID,
		Status:          getBreakpointStatus(bp),
		Location:        getBreakpointLocation(bp),
		HitCount:        uint64(bp.TotalHitCount),
	}

	context := c.createDebugContext(state)
	context.Operation = "set_breakpoint"

	return BreakpointResponse{
		Status:     "success",
		Context:    context,
		Breakpoint: *breakpoint,
	}
}

// ListBreakpoints returns all currently set breakpoints
func (c *Client) ListBreakpoints() BreakpointListResponse {
	if c.client == nil {
		return BreakpointListResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: "no active debug session",
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	bps, err := c.client.ListBreakpoints(false)
	if err != nil {
		return BreakpointListResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: fmt.Sprintf("failed to list breakpoints: %v", err),
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	var breakpoints []Breakpoint
	for _, bp := range bps {
		breakpoints = append(breakpoints, Breakpoint{
			DelveBreakpoint: bp,
			ID:              bp.ID,
			Status:          getBreakpointStatus(bp),
			Location:        getBreakpointLocation(bp),
			HitCount:        uint64(bp.TotalHitCount),
		})
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state while listing breakpoints: %v", err)
	}

	context := c.createDebugContext(state)
	context.Operation = "list_breakpoints"

	return BreakpointListResponse{
		Status:      "success",
		Context:     context,
		Breakpoints: breakpoints,
	}
}

// RemoveBreakpoint removes a breakpoint by its ID
func (c *Client) RemoveBreakpoint(id int) BreakpointResponse {
	if c.client == nil {
		return BreakpointResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: "no active debug session",
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	// Get breakpoint info before removing
	bps, err := c.client.ListBreakpoints(false)
	if err != nil {
		return BreakpointResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: fmt.Sprintf("failed to get breakpoint info: %v", err),
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	var targetBp *api.Breakpoint
	for _, bp := range bps {
		if bp.ID == id {
			targetBp = bp
			break
		}
	}

	if targetBp == nil {
		return BreakpointResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: fmt.Sprintf("breakpoint %d not found", id),
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	logger.Debug("Removing breakpoint %d at %s:%d", id, targetBp.File, targetBp.Line)
	_, err = c.client.ClearBreakpoint(id)
	if err != nil {
		return BreakpointResponse{
			Status: "error",
			Context: DebugContext{
				ErrorMessage: fmt.Sprintf("failed to remove breakpoint: %v", err),
				Timestamp:    getCurrentTimestamp(),
			},
		}
	}

	// Get current state for context
	state, err := c.client.GetState()
	if err != nil {
		logger.Debug("Warning: Failed to get state after removing breakpoint: %v", err)
	}

	breakpoint := Breakpoint{
		DelveBreakpoint: targetBp,
		ID:              targetBp.ID,
		Status:          "removed",
		Location:        getBreakpointLocation(targetBp),
		HitCount:        uint64(targetBp.TotalHitCount),
	}

	context := c.createDebugContext(state)
	context.Operation = "remove_breakpoint"

	return BreakpointResponse{
		Status:     "success",
		Context:    context,
		Breakpoint: breakpoint,
	}
}

func getCurrentTimestamp() time.Time {
	return time.Now()
}
