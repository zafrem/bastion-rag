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

func mockLLMReady(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

func TestIntegration(t *testing.T) {
	// If mock-llm is not already running (e.g. started by scripts/run-integration.sh),
	// start it in its own process group so all descendants are killed on teardown.
	if !mockLLMReady("http://localhost:11435/health", 500*time.Millisecond) {
		cmd := exec.Command("go", "run", "mock-llm/main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start mock LLM: %v", err)
		}
		defer func() {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			cmd.Wait()
		}()

		if !mockLLMReady("http://localhost:11435/health", 10*time.Second) {
			t.Fatal("Mock LLM did not become ready within 10s")
		}
	}

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
