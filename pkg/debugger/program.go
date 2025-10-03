package debugger

import (
	"context"
	"fmt"
	"github.com/go-delve/delve/service/api"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/go-delve/delve/pkg/gobuild"
	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
	"github.com/richard-senior/mcp/internal/logger"
)

// ensureBinaryArchitecture checks if a binary matches the system architecture and rebuilds if necessary
func ensureBinaryArchitecture(binaryPath string) (string, error) {
	// Check the architecture of the binary
	cmd := exec.Command("file", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return binaryPath, fmt.Errorf("failed to check binary architecture: %v", err)
	}
	
	fileOutput := string(output)
	logger.Debug("Binary file info: %s", fileOutput)
	
	// Get system architecture
	systemCmd := exec.Command("uname", "-m")
	systemOutput, err := systemCmd.Output()
	if err != nil {
		return binaryPath, fmt.Errorf("failed to get system architecture: %v", err)
	}
	
	systemArch := strings.TrimSpace(string(systemOutput))
	
	// Check if binary architecture matches system
	var binaryMatchesSystem bool
	if systemArch == "arm64" && strings.Contains(fileOutput, "arm64") {
		binaryMatchesSystem = true
	} else if systemArch == "x86_64" && (strings.Contains(fileOutput, "x86_64") || strings.Contains(fileOutput, "amd64")) {
		binaryMatchesSystem = true
	}
	
	if binaryMatchesSystem {
		logger.Debug("Binary architecture matches system architecture")
		return binaryPath, nil
	}
	
	// Try to find the source file and rebuild
	logger.Info("Binary architecture doesn't match system, attempting to rebuild")
	
	// Look for a .go file with the same base name
	dir := filepath.Dir(binaryPath)
	baseName := filepath.Base(binaryPath)
	
	// Common source file patterns
	possibleSources := []string{
		filepath.Join(dir, baseName+".go"),
		filepath.Join(dir, "main.go"),
		filepath.Join(dir, "*.go"),
	}
	
	var sourceFile string
	for _, pattern := range possibleSources {
		if strings.Contains(pattern, "*") {
			matches, err := filepath.Glob(pattern)
			if err == nil && len(matches) > 0 {
				sourceFile = matches[0]
				break
			}
		} else {
			if _, err := os.Stat(pattern); err == nil {
				sourceFile = pattern
				break
			}
		}
	}
	
	if sourceFile == "" {
		return binaryPath, fmt.Errorf("could not find source file to rebuild binary with correct architecture")
	}
	
	// Rebuild with correct architecture
	newBinaryPath := binaryPath + "_fixed"
	
	// Set correct environment
	env := os.Environ()
	env = append(env, "GOARCH="+getTargetArch(systemArch))
	env = append(env, "GOOS="+runtime.GOOS)
	
	buildCmd := exec.Command("go", "build", "-o", newBinaryPath, sourceFile)
	buildCmd.Env = env
	
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		return binaryPath, fmt.Errorf("failed to rebuild binary: %v\nOutput: %s", err, string(buildOutput))
	}
	
	logger.Info("Successfully rebuilt binary with correct architecture: %s", newBinaryPath)
	return newBinaryPath, nil
}

// getTargetArch maps system architecture to Go architecture
func getTargetArch(systemArch string) string {
	switch systemArch {
	case "arm64", "aarch64":
		return "arm64"
	case "x86_64":
		return "amd64"
	default:
		return runtime.GOARCH
	}
}

// detectAndConfigureArchitecture detects the system architecture and configures Go environment accordingly
func detectAndConfigureArchitecture() error {
	logger.Info("Starting architecture detection and configuration")
	
	// Get the actual system architecture using multiple methods
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("Failed to detect system architecture: %v", err)
		return nil // Continue with current settings
	}
	
	actualArch := strings.TrimSpace(string(output))
	logger.Info("Detected system architecture: %s", actualArch)
	
	// Also check with arch command as a backup
	cmd = exec.Command("arch")
	archOutput, err := cmd.Output()
	if err == nil {
		archResult := strings.TrimSpace(string(archOutput))
		logger.Info("Arch command reports: %s", archResult)
		if archResult != actualArch {
			logger.Warn("uname reports %s but arch reports %s", actualArch, archResult)
		}
	}
	
	// Get the actual system OS
	cmd = exec.Command("uname", "-s")
	output, err = cmd.Output()
	if err != nil {
		logger.Warn("Failed to detect system OS: %v", err)
		return nil
	}
	
	actualOS := strings.TrimSpace(string(output))
	logger.Info("Detected system OS: %s", actualOS)
	
	// Map system architecture to Go architecture
	var targetArch string
	switch actualArch {
	case "arm64", "aarch64":
		targetArch = "arm64"
	case "x86_64", "amd64":
		targetArch = "amd64"
	default:
		logger.Warn("Unknown system architecture: %s, using current settings", actualArch)
		return nil
	}
	
	// Map system OS to Go OS
	var targetOS string
	switch strings.ToLower(actualOS) {
	case "darwin":
		targetOS = "darwin"
	case "linux":
		targetOS = "linux"
	case "windows":
		targetOS = "windows"
	default:
		logger.Warn("Unknown system OS: %s, using current settings", actualOS)
		return nil
	}
	
	logger.Info("Target architecture: %s, Target OS: %s", targetArch, targetOS)
	
	// Check current Go environment
	cmd = exec.Command("go", "env", "GOARCH", "GOOS")
	output, err = cmd.Output()
	if err != nil {
		logger.Warn("Failed to get Go environment: %v", err)
		return nil
	}
	
	envLines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(envLines) < 2 {
		logger.Warn("Unexpected go env output format")
		return nil
	}
	
	currentGoArch := strings.TrimSpace(envLines[0])
	currentGoOS := strings.TrimSpace(envLines[1])
	
	logger.Info("System: %s/%s, Go environment: %s/%s, Runtime: %s/%s", 
		actualOS, actualArch, currentGoOS, currentGoArch, runtime.GOOS, runtime.GOARCH)
	
	// Force set environment variables to match actual system architecture
	// This is especially important on Apple Silicon Macs where Go might be running under Rosetta
	if currentGoArch != targetArch || runtime.GOARCH != targetArch {
		logger.Info("Forcing GOARCH from %s to %s (system: %s, runtime: %s)", 
			currentGoArch, targetArch, actualArch, runtime.GOARCH)
		err = os.Setenv("GOARCH", targetArch)
		if err != nil {
			return fmt.Errorf("failed to set GOARCH environment variable: %v", err)
		}
		logger.Info("Successfully set GOARCH=%s", targetArch)
	}
	
	if currentGoOS != targetOS || runtime.GOOS != targetOS {
		logger.Info("Forcing GOOS from %s to %s (system: %s, runtime: %s)", 
			currentGoOS, targetOS, actualOS, runtime.GOOS)
		err = os.Setenv("GOOS", targetOS)
		if err != nil {
			return fmt.Errorf("failed to set GOOS environment variable: %v", err)
		}
		logger.Info("Successfully set GOOS=%s", targetOS)
	}
	
	// Also set CGO_ENABLED=1 to ensure proper native compilation
	err = os.Setenv("CGO_ENABLED", "1")
	if err != nil {
		logger.Warn("Failed to set CGO_ENABLED: %v", err)
	} else {
		logger.Info("Set CGO_ENABLED=1")
	}
	
	logger.Info("Architecture detection and configuration completed")
	return nil
}

// LaunchProgram starts a new program with debugging enabled
func (c *Client) LaunchProgram(program string, args []string) LaunchResponse {
	logger.Info("LaunchProgram called with program: %s", program)
	
	if c.client != nil {
		logger.Info("Debug session already active, returning error")
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("debug session already active"))
	}

	logger.Info("Starting LaunchProgram for %s", program)

	// Detect and configure correct architecture
	logger.Info("About to call detectAndConfigureArchitecture")
	if err := detectAndConfigureArchitecture(); err != nil {
		logger.Warn("Architecture configuration warning: %v", err)
	}
	logger.Info("Completed detectAndConfigureArchitecture call")

	// Ensure program file exists and is executable
	absPath, err := filepath.Abs(program)
	if err != nil {
		logger.Error("Failed to get absolute path: %v", err)
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("failed to get absolute path: %v", err))
	}

	logger.Info("Absolute path: %s", absPath)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		logger.Error("Program file not found: %s", absPath)
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("program file not found: %s", absPath))
	}

	// Ensure binary architecture matches system architecture
	logger.Info("About to check binary architecture")
	correctedPath, err := ensureBinaryArchitecture(absPath)
	if err != nil {
		logger.Warn("Binary architecture check failed: %v", err)
		// Continue with original path
		correctedPath = absPath
	} else if correctedPath != absPath {
		logger.Info("Using architecture-corrected binary: %s", correctedPath)
		absPath = correctedPath
	}
	logger.Info("Using binary path: %s", absPath)

	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("failed to find available port: %v", err))
	}

	// Configure Delve logging
	logflags.Setup(false, "", "")

	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("couldn't start listener: %s", err))
	}

	// Create pipes for stdout and stderr
	stdoutReader, stdoutRedirect, err := proc.Redirector()
	if err != nil {
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("failed to create stdout redirector: %v", err))
	}

	stderrReader, stderrRedirect, err := proc.Redirector()
	if err != nil {
		stdoutRedirect.File.Close()
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("failed to create stderr redirector: %v", err))
	}

	// Create Delve config
	config := &service.Config{
		Listener:    listener,
		APIVersion:  2,
		AcceptMulti: true,
		ProcessArgs: append([]string{absPath}, args...),
		Debugger: debugger.Config{
			WorkingDir:     "",
			Backend:        "default",
			CheckGoVersion: false, // Disable Go version check to avoid some issues
			DisableASLR:    true,
			Stdout:         stdoutRedirect,
			Stderr:         stderrRedirect,
		},
	}

	// Start goroutines to capture output
	go c.captureOutput(stdoutReader, "stdout")
	go c.captureOutput(stderrReader, "stderr")

	// Create and start the debugging server
	server := rpccommon.NewServer(config)
	if server == nil {
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("failed to create debug server"))
	}

	c.server = server

	// Start server in a goroutine
	serverError := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				serverError <- fmt.Errorf("server panicked: %v", r)
			}
		}()
		
		err := server.Run()
		if err != nil {
			serverError <- err
		}
	}()

	// Wait for server to be ready or fail
	addr := listener.Addr().String()
	logger.Debug("Waiting for server at %s", addr)
	
	// Check for server errors first
	select {
	case err := <-serverError:
		if err != nil {
			return c.createLaunchResponse(nil, program, args, fmt.Errorf("debug server failed to start: %v", err))
		}
	case <-time.After(1 * time.Second):
		// Continue with connection attempts
	}
	
	// Simple connection test with retries
	maxRetries := 50 // 5 seconds with 100ms intervals
	for i := 0; i < maxRetries; i++ {
		time.Sleep(100 * time.Millisecond)
		
		// Check for server errors during connection attempts
		select {
		case err := <-serverError:
			if err != nil {
				return c.createLaunchResponse(nil, program, args, fmt.Errorf("debug server failed during startup: %v", err))
			}
		default:
		}
		
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			logger.Debug("Server is accepting connections")
			break
		}
		
		if i == maxRetries-1 {
			return c.createLaunchResponse(nil, program, args, fmt.Errorf("server failed to accept connections after 5 seconds"))
		}
	}

	// Create RPC client
	client := rpc2.NewClient(addr)
	
	// Test the connection with a simple call
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	stateChan := make(chan *api.DebuggerState, 1)
	errChan := make(chan error, 1)
	
	go func() {
		state, err := client.GetState()
		if err != nil {
			errChan <- err
		} else {
			stateChan <- state
		}
	}()
	
	select {
	case state := <-stateChan:
		c.client = client
		c.target = absPath
		logger.Debug("Successfully connected to debugger")
		return c.createLaunchResponse(state, program, args, nil)
	case err := <-errChan:
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("failed to get initial state: %v", err))
	case <-ctx.Done():
		return c.createLaunchResponse(nil, program, args, fmt.Errorf("timeout connecting to debugger"))
	}
}

// AttachToProcess attaches to an existing process with the given PID
func (c *Client) AttachToProcess(pid int) AttachResponse {
	if c.client != nil {
		return c.createAttachResponse(nil, pid, "", nil, fmt.Errorf("debug session already active"))
	}

	logger.Debug("Starting AttachToProcess for PID %d", pid)

	// Get an available port for the debug server
	port, err := getFreePort()
	if err != nil {
		return c.createAttachResponse(nil, pid, "", nil, fmt.Errorf("failed to find available port: %v", err))
	}

	logger.Debug("Setting up Delve logging")
	// Configure Delve logging
	logflags.Setup(false, "", "")

	logger.Debug("Creating listener on port %d", port)
	// Create a listener for the debug server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return c.createAttachResponse(nil, pid, "", nil, fmt.Errorf("couldn't start listener: %s", err))
	}

	// Note: When attaching to an existing process, we can't easily redirect its stdout/stderr
	// as those file descriptors are already connected. Output capture is limited for attach mode.
	logger.Debug("Note: Output redirection is limited when attaching to an existing process")

	logger.Debug("Creating Delve config for attach")
	// Create Delve config for attaching to process
	config := &service.Config{
		Listener:    listener,
		APIVersion:  2,
		AcceptMulti: true,
		ProcessArgs: []string{},
		Debugger: debugger.Config{
			AttachPid:      pid,
			Backend:        "default",
			CheckGoVersion: true,
			DisableASLR:    true,
		},
	}

	logger.Debug("Creating debug server")
	// Create and start the debugging server with attach to PID
	server := rpccommon.NewServer(config)
	if server == nil {
		return c.createAttachResponse(nil, pid, "", nil, fmt.Errorf("failed to create debug server"))
	}

	c.server = server

	// Create a channel to signal when the server is ready or fails
	serverReady := make(chan error, 1)

	logger.Debug("Starting debug server in goroutine")
	// Start server in a goroutine
	go func() {
		logger.Debug("Running server")
		err := server.Run()
		if err != nil {
			logger.Debug("Debug server error: %v", err)
			serverReady <- err
		}
		logger.Debug("Server run completed")
	}()

	logger.Debug("Waiting for server to start")

	// Try to connect to the server with a longer timeout
	addr := listener.Addr().String()

	// Wait up to 10 seconds for server to be available
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Wait for server to be ready first
	time.Sleep(500 * time.Millisecond)

	// Try to connect repeatedly until timeout
	connected := false
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for !connected {
		select {
		case <-ctx.Done():
			// Timeout reached
			return c.createAttachResponse(nil, pid, "", nil, fmt.Errorf("timed out waiting for debug server to start"))
		case err := <-serverReady:
			// Server reported an error
			return c.createAttachResponse(nil, pid, "", nil, fmt.Errorf("debug server failed to start: %v", err))
		case <-ticker.C:
			// Try to connect
			client := rpc2.NewClient(addr)
			
			// Create a timeout context for the GetState call
			stateCtx, stateCancel := context.WithTimeout(context.Background(), 2*time.Second)
			
			// Use a goroutine to make the GetState call with timeout
			stateChan := make(chan *api.DebuggerState, 1)
			errChan := make(chan error, 1)
			
			go func() {
				defer stateCancel()
				state, err := client.GetState()
				if err != nil {
					errChan <- err
				} else {
					stateChan <- state
				}
			}()
			
			select {
			case state := <-stateChan:
				if state != nil {
					// Connection successful
					c.client = client
					c.pid = pid
					connected = true
					logger.Debug("Successfully attached to process with PID: %d", pid)
					return c.createAttachResponse(state, pid, "", nil, nil)
				}
			case <-errChan:
				// Connection failed, continue trying
				stateCancel()
			case <-stateCtx.Done():
				// GetState timed out, continue trying
				stateCancel()
			}
		}
	}

	return c.createAttachResponse(nil, pid, "", nil, fmt.Errorf("failed to attach to process"))
}

// Close terminates the debug session
func (c *Client) Close() (*CloseResponse, error) {
	if c.client == nil {
		return &CloseResponse{
			Status: "success",
			Context: DebugContext{
				Timestamp: time.Now(),
				Operation: "close",
			},
			Summary: "No active debug session to close",
		}, nil
	}

	// Signal to stop output capturing goroutines
	close(c.stopOutput)

	// Create a context with timeout to prevent indefinite hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create error channel
	errChan := make(chan error, 1)

	// Attempt to detach from the debugger in a separate goroutine
	go func() {
		err := c.client.Detach(true)
		if err != nil {
			logger.Debug("Warning: Failed to detach from debugged process: %v", err)
		}
		errChan <- err
	}()

	// Wait for completion or timeout
	var detachErr error
	select {
	case detachErr = <-errChan:
		// Operation completed successfully
	case <-ctx.Done():
		logger.Debug("Warning: Detach operation timed out after 5 seconds")
		detachErr = ctx.Err()
	}

	// Reset the client
	c.client = nil

	// Clean up the debug binary if it exists
	if c.target != "" {
		gobuild.Remove(c.target)
		c.target = ""
	}

	// Create a new channel for server stop operations
	stopChan := make(chan error, 1)

	// Stop the debug server if it's running
	if c.server != nil {
		go func() {
			err := c.server.Stop()
			if err != nil {
				logger.Debug("Warning: Failed to stop debug server: %v", err)
			}
			stopChan <- err
		}()

		// Wait for completion or timeout
		select {
		case <-stopChan:
			// Operation completed
		case <-time.After(5 * time.Second):
			logger.Debug("Warning: Server stop operation timed out after 5 seconds")
		}
	}

	// Create debug context
	debugContext := DebugContext{
		Timestamp: time.Now(),
		Operation: "close",
	}

	// Get exit code
	exitCode := 0
	if detachErr != nil {
		exitCode = 1
	}

	// Create close response
	response := &CloseResponse{
		Status:   "success",
		Context:  debugContext,
		ExitCode: exitCode,
		Summary:  fmt.Sprintf("Debug session closed with exit code %d", exitCode),
	}

	logger.Debug("Close response: %+v", response)
	return response, detachErr
}

// DebugSourceFile compiles and debugs a Go source file
func (c *Client) DebugSourceFile(sourceFile string, args []string) DebugSourceResponse {
	if c.client != nil {
		return c.createDebugSourceResponse(nil, sourceFile, "", args, fmt.Errorf("debug session already active"))
	}

	// Detect and configure correct architecture
	if err := detectAndConfigureArchitecture(); err != nil {
		logger.Warn("Architecture configuration warning: %v", err)
	}

	// Ensure source file exists
	absPath, err := filepath.Abs(sourceFile)
	if err != nil {
		return c.createDebugSourceResponse(nil, sourceFile, "", args, fmt.Errorf("failed to get absolute path: %v", err))
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return c.createDebugSourceResponse(nil, sourceFile, "", args, fmt.Errorf("source file not found: %s", absPath))
	}

	// Generate a unique debug binary name
	debugBinary := gobuild.DefaultDebugBinaryPath("debug_binary")

	logger.Debug("Compiling source file %s to %s", absPath, debugBinary)

	// Compile the source file with output capture
	cmd, output, err := gobuild.GoBuildCombinedOutput(debugBinary, []string{absPath}, "-gcflags all=-N")
	if err != nil {
		logger.Debug("Build command: %s", cmd)
		logger.Debug("Build output: %s", string(output))
		gobuild.Remove(debugBinary)
		return c.createDebugSourceResponse(nil, sourceFile, debugBinary, args, fmt.Errorf("failed to compile source file: %v\nOutput: %s", err, string(output)))
	}

	// Launch the compiled binary with the debugger
	response := c.LaunchProgram(debugBinary, args)
	if response.Context.ErrorMessage != "" {
		gobuild.Remove(debugBinary)
		return c.createDebugSourceResponse(nil, sourceFile, debugBinary, args, fmt.Errorf(response.Context.ErrorMessage))
	}

	// Store the binary path for cleanup
	c.target = debugBinary

	return c.createDebugSourceResponse(response.Context.DelveState, sourceFile, debugBinary, args, nil)
}

// DebugTest compiles and debugs a Go test function
func (c *Client) DebugTest(testFilePath string, testName string, testFlags []string) DebugTestResponse {
	response := DebugTestResponse{
		TestName:  testName,
		TestFile:  testFilePath,
		TestFlags: testFlags,
	}
	if c.client != nil {
		return c.createDebugTestResponse(nil, &response, fmt.Errorf("debug session already active"))
	}

	// Detect and configure correct architecture
	if err := detectAndConfigureArchitecture(); err != nil {
		logger.Warn("Architecture configuration warning: %v", err)
	}

	// Ensure test file exists
	absPath, err := filepath.Abs(testFilePath)
	if err != nil {
		return c.createDebugTestResponse(nil, &response, fmt.Errorf("failed to get absolute path: %v", err))
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return c.createDebugTestResponse(nil, &response, fmt.Errorf("test file not found: %s", absPath))
	}

	// Get the directory of the test file
	testDir := filepath.Dir(absPath)
	logger.Debug("Test directory: %s", testDir)

	// Generate a unique debug binary name
	debugBinary := gobuild.DefaultDebugBinaryPath("debug.test")

	logger.Debug("Compiling test package in %s to %s", testDir, debugBinary)

	// Save current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return c.createDebugTestResponse(nil, &response, fmt.Errorf("failed to get current directory: %v", err))
	}

	// Change to test directory
	if err := os.Chdir(testDir); err != nil {
		return c.createDebugTestResponse(nil, &response, fmt.Errorf("failed to change to test directory: %v", err))
	}

	// Ensure we change back to original directory
	defer func() {
		if err := os.Chdir(currentDir); err != nil {
			logger.Error("Failed to restore original directory: %v", err)
		}
	}()

	// Compile the test package with output capture using test-specific build flags
	cmd, output, err := gobuild.GoTestBuildCombinedOutput(debugBinary, []string{testDir}, "-gcflags all=-N")
	response.BuildCommand = cmd
	response.BuildOutput = string(output)
	if err != nil {
		gobuild.Remove(debugBinary)
		return c.createDebugTestResponse(nil, &response, fmt.Errorf("failed to compile test package: %v\nOutput: %s", err, string(output)))
	}

	// Create args to run the specific test
	args := []string{
		"-test.v", // Verbose output
	}

	// Add specific test pattern if provided
	if testName != "" {
		// Escape special regex characters in the test name
		escapedTestName := regexp.QuoteMeta(testName)
		// Create a test pattern that matches exactly the provided test name
		args = append(args, fmt.Sprintf("-test.run=^%s$", escapedTestName))
	}

	// Add any additional test flags
	args = append(args, testFlags...)

	logger.Debug("Launching test binary with debugger, test name: %s, args: %v", testName, args)
	// Launch the compiled test binary with the debugger
	response2 := c.LaunchProgram(debugBinary, args)
	if response2.Context.ErrorMessage != "" {
		gobuild.Remove(debugBinary)
		return c.createDebugTestResponse(nil, &response, fmt.Errorf(response.Context.ErrorMessage))
	}

	// Store the binary path for cleanup
	c.target = debugBinary

	return c.createDebugTestResponse(response2.Context.DelveState, &response, nil)
}

// createLaunchResponse creates a response for the launch command
func (c *Client) createLaunchResponse(state *api.DebuggerState, program string, args []string, err error) LaunchResponse {
	context := c.createDebugContext(state)
	context.Operation = "launch"

	if err != nil {
		context.ErrorMessage = err.Error()
	}

	return LaunchResponse{
		Context:  &context,
		Program:  program,
		Args:     args,
		ExitCode: 0,
	}
}

// createAttachResponse creates a response for the attach command
func (c *Client) createAttachResponse(state *api.DebuggerState, pid int, target string, process *Process, err error) AttachResponse {
	context := c.createDebugContext(state)
	context.Operation = "attach"

	if err != nil {
		context.ErrorMessage = err.Error()
	}

	return AttachResponse{
		Status:  "success",
		Context: &context,
		Pid:     pid,
		Target:  target,
		Process: process,
	}
}

// createDebugSourceResponse creates a response for the debug source command
func (c *Client) createDebugSourceResponse(state *api.DebuggerState, sourceFile string, debugBinary string, args []string, err error) DebugSourceResponse {
	context := c.createDebugContext(state)
	context.Operation = "debug_source"

	if err != nil {
		context.ErrorMessage = err.Error()
	}

	return DebugSourceResponse{
		Status:      "success",
		Context:     &context,
		SourceFile:  sourceFile,
		DebugBinary: debugBinary,
		Args:        args,
	}
}

// createDebugTestResponse creates a response for the debug test command
func (c *Client) createDebugTestResponse(state *api.DebuggerState, response *DebugTestResponse, err error) DebugTestResponse {
	context := c.createDebugContext(state)
	context.Operation = "debug_test"
	response.Context = &context

	if err != nil {
		context.ErrorMessage = err.Error()
		response.Status = "error"
	}

	return *response
}
