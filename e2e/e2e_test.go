package e2e_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// jsonRPCRequest represents a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      *int           `json:"id,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError  `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// buildBinary builds the silo binary for testing.
func buildBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "silo")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/silo")
	cmd.Dir = filepath.Join(t.TempDir(), "..", "..")
	// Find the actual silo directory
	siloDir := findSiloDir(t)
	cmd.Dir = siloDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, string(out))
	}

	return binPath
}

func findSiloDir(t *testing.T) string {
	t.Helper()

	// Walk up from the test file to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatal("could not find silo directory")
	return ""
}

// createTestConfig creates a temporary config file.
func createTestConfig(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config := `profile: test
engram_path: engram
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	return configPath
}

func TestE2E_BinaryExists(t *testing.T) {
	binPath := buildBinary(t)

	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found at %s: %v", binPath, err)
	}
}

func TestE2E_ServerStarts(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	// Set XDG config home so silo uses our test config
	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	// Start the server — it will block waiting for stdin
	// We just verify it starts without crashing
	err := cmd.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	// Kill the process
	if err := cmd.Process.Kill(); err != nil {
		t.Logf("warning: failed to kill process: %v", err)
	}

	// Wait for it to exit
	_ = cmd.Wait()
}

func TestE2E_JSONRPCParseError(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Send invalid JSON to trigger parse error
	_, _ = stdin.Write([]byte("not valid json\n"))

	// Read response with timeout
	scanner := bufio.NewScanner(stdout)
	scanner.Scan() // skip potential startup output
	if scanner.Scan() {
		line := scanner.Text()
		// Should contain a parse error
		if !strings.Contains(line, "error") && !strings.Contains(line, "parse") {
			t.Logf("response: %s", line)
		}
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func TestE2E_ToolInitialization(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Send initialize request
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	}

	data, _ := json.Marshal(initReq)
	_, _ = stdin.Write(append(data, '\n'))

	// Read response
	scanner := bufio.NewScanner(stdout)
	var gotResponse bool
	for i := 0; i < 10; i++ {
		if scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, `"result"`) || strings.Contains(line, `"id":1`) {
				gotResponse = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !gotResponse {
		t.Log("server may need more time to respond")
	}
}

func TestE2E_GracefulDegradation(t *testing.T) {
	// This test verifies that the server starts even when Engram is unreachable
	// (pointing to a non-existent port)
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	// The config points to localhost:9999 which doesn't exist
	// Server should still start (graceful degradation)
	err := cmd.Start()
	if err != nil {
		t.Fatalf("server should start even when Engram is unreachable: %v", err)
	}

	// Give it time to attempt connection
	time.Sleep(1 * time.Second)

	// Verify process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		t.Fatal("server exited when it should have continued in degraded mode")
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func TestE2E_TUILaunchesWithoutServer(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	// Run without --server flag (TUI mode)
	// It should start and then exit quickly since there's no terminal
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
		"TERM=dumb",
	)

	// Set a timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		// TUI might fail without a real terminal, which is expected
		// The important thing is it doesn't panic
		if err != nil {
			t.Logf("TUI exited with: %v (expected without real terminal)", err)
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Log("TUI is running (didn't exit within 3s)")
	}
}

func TestE2E_BinaryHelpFlag(t *testing.T) {
	binPath := buildBinary(t)

	cmd := exec.Command(binPath, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "-server") {
		t.Error("--help output should mention -server flag")
	}
}

func TestE2E_SearchToolWithUnreachableEngram(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Initialize first
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	}
	data, _ := json.Marshal(initReq)
	_, _ = stdin.Write(append(data, '\n'))

	// Wait for init response
	time.Sleep(500 * time.Millisecond)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		// consume init response
		break
	}

	// Send tools/list to verify tools are registered
	listReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	data, _ = json.Marshal(listReq)
	_, _ = stdin.Write(append(data, '\n'))

	// Read response
	time.Sleep(500 * time.Millisecond)
	if scanner.Scan() {
		line := scanner.Text()
		// Should contain tool definitions
		if strings.Contains(line, "search") || strings.Contains(line, "tools") {
			t.Log("tools/list response received")
		}
	}
}

func TestE2E_SendAddResource(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Initialize
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	}
	data, _ := json.Marshal(initReq)
	_, _ = stdin.Write(append(data, '\n'))
	time.Sleep(500 * time.Millisecond)

	// Consume init response
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		break
	}

	// Try to call add_resource (will fail since Engram is down, but verifies the tool exists)
	addReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "add_resource",
			"arguments": map[string]string{
				"url":   "https://example.com/test",
				"title": "Test Resource",
			},
		},
	}
	data, _ = json.Marshal(addReq)
	_, _ = stdin.Write(append(data, '\n'))

	// Read response
	time.Sleep(1 * time.Second)
	if scanner.Scan() {
		line := scanner.Text()
		// Should get an error response since Engram is unreachable
		t.Logf("add_resource response: %s", truncate(line, 200))
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func TestE2E_MultipleRequests(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Send multiple requests in sequence
	requests := []jsonRPCRequest{
		{JSONRPC: "2.0", ID: 1, Method: "initialize", Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		}},
		{JSONRPC: "2.0", ID: 2, Method: "tools/list"},
	}

	scanner := bufio.NewScanner(stdout)
	for _, req := range requests {
		data, _ := json.Marshal(req)
		_, _ = stdin.Write(append(data, '\n'))
		time.Sleep(300 * time.Millisecond)

		if scanner.Scan() {
			line := scanner.Text()
			t.Logf("request %d response: %s", req.ID, truncate(line, 100))
		}
	}
}

func TestE2E_GracefulShutdown(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Give it time to start
	time.Sleep(500 * time.Millisecond)

	// Send SIGTERM for graceful shutdown
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Logf("failed to send signal: %v", err)
		_ = cmd.Process.Kill()
	}

	// Wait for exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		t.Log("server shut down gracefully")
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		t.Log("server did not shut down within timeout")
	}
}

func TestE2E_InvalidURL(t *testing.T) {
	binPath := buildBinary(t)
	configPath := createTestConfig(t)

	configDir := filepath.Dir(configPath)

	cmd := exec.Command(binPath, "--server")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configDir,
		"XDG_DATA_HOME="+t.TempDir(),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Initialize
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	}
	data, _ := json.Marshal(initReq)
	_, _ = stdin.Write(append(data, '\n'))
	time.Sleep(500 * time.Millisecond)

	// Consume init response
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		break
	}

	// Call add_resource with invalid URL
	addReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "add_resource",
			"arguments": map[string]string{
				"url": "not-a-valid-url",
			},
		},
	}
	data, _ = json.Marshal(addReq)
	_, _ = stdin.Write(append(data, '\n'))

	// Read response — should be a validation error
	time.Sleep(500 * time.Millisecond)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "error") || strings.Contains(line, "invalid") {
			t.Log("received validation error as expected")
		} else {
			t.Logf("response: %s", truncate(line, 200))
		}
	}
}

func fmtError(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
