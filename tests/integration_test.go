package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	// Start Mock LLM in its own process group so all descendants are killed on teardown.
	// "go run" compiles and then forks a child process; killing only the "go run" PID
	// leaves that child orphaned and still holding the stdout pipe, which causes the
	// "I/O incomplete" failure. Killing the whole process group avoids this.
	cmd := exec.Command("go", "run", "mock-llm/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start mock LLM: %v", err)
	}
	defer func() {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Wait()
	}()

	// Wait for mock LLM to start
	time.Sleep(2 * time.Second)

	// Test /api/generate
	prompt := "Hello, mock LLM!"
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  "llama3",
		"prompt": prompt,
		"stream": false,
	})

	resp, err := http.Post("http://localhost:11435/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to send request to mock LLM: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Mock LLM Response: %s\n", string(body))

	var genResp map[string]interface{}
	if err := json.Unmarshal(body, &genResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expected := fmt.Sprintf("This is a mock response to your prompt: %s", prompt)
	if genResp["response"] != expected {
		t.Errorf("Expected response %q, got %q", expected, genResp["response"])
	}
}
