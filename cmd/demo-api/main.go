package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Step struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Info   string `json:"info"`
	Status string `json:"status"`
}

type ProcessRequest struct {
	Prompt string `json:"prompt"`
}

type ProcessResponse struct {
	Steps       []Step `json:"steps"`
	FinalOutput string `json:"final_output"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/process", handleProcess)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	port := 8090
	fmt.Printf("Demo API listening on :%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), corsMiddleware(mux)))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	var req ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simulate the 10-step flow with specific info based on the prompt
	steps := []Step{
		{
			ID:     "sentinel-in",
			Name:   "Sentinel (In)",
			Info:   "Security scan complete. No prompt injection or malicious intent detected in input.",
			Status: "success",
		},
		{
			ID:     "vault-p1",
			Name:   "Vault (Anonymize)",
			Info:   "PII detection triggered. 'John Doe' identified as PERSON and replaced with [TOKEN_PERSON_1].",
			Status: "success",
		},
		{
			ID:     "navigator",
			Name:   "Navigator (Retrieve)",
			Info:   "Hybrid search executed. Retrieved 3 relevant documents from 'Customer Finance' collection.",
			Status: "success",
		},
		{
			ID:     "anchor-in",
			Name:   "Anchor (Secure)",
			Info:   "Differential privacy noise (σ=0.01) added to query embeddings to prevent membership inference.",
			Status: "success",
		},
		{
			ID:     "llm",
			Name:   "Mock LLM",
			Info:   "Generating response based on anonymized prompt and retrieved context...",
			Status: "success",
		},
		{
			ID:     "anchor-out",
			Name:   "Anchor (Verify)",
			Info:   "Semantic drift analysis: 0.05. Response bias check: PASSED. Output integrity verified.",
			Status: "success",
		},
		{
			ID:     "vault-p2",
			Name:   "Vault (Deanonymize)",
			Info:   "Token resolution: [TOKEN_PERSON_1] -> 'John Doe'. Fine-grained RBAC permission verified for requester.",
			Status: "success",
		},
		{
			ID:     "sentinel-out",
			Name:   "Sentinel (Out)",
			Info:   "Output guardrail check: No PII re-emergence. Hallucination score: 0.98 (Grounded).",
			Status: "success",
		},
	}

	// Add some processing delay for realism (or let the frontend handle it)
	// We'll return everything at once and let the frontend animate.

	resp := ProcessResponse{
		Steps:       steps,
		FinalOutput: fmt.Sprintf("The credit card balance for John Doe is $5,000. (Processed at %s)", time.Now().Format(time.Kitchen)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
