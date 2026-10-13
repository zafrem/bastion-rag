package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Message   Message   `json:"message"`
	Done      bool      `json:"done"`
}

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/generate", handleGenerate)
	mux.HandleFunc("/api/chat", handleChat)
	mux.HandleFunc("/api/embeddings", handleEmbeddings)
	mux.HandleFunc("/api/tags", handleTags)
	mux.HandleFunc("/health", handleHealth)

	port := 11435
	fmt.Printf("Mock LLM listening on :%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), loggerMiddleware(mux)))
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))

		log.Printf("RECEIVED: %s %s | Body: %s", r.Method, r.URL.Path, string(body))
		
		next.ServeHTTP(w, r)
		
		log.Printf("COMPLETED: %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := GenerateResponse{
		Model:     req.Model,
		CreatedAt: time.Now(),
		Response:  fmt.Sprintf("This is a mock response to your prompt: %s", req.Prompt),
		Done:      true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lastMsg := ""
	if len(req.Messages) > 0 {
		lastMsg = req.Messages[len(req.Messages)-1].Content
	}

	resp := ChatResponse{
		Model:     req.Model,
		CreatedAt: time.Now(),
		Message: Message{
			Role:    "assistant",
			Content: fmt.Sprintf("I received your message: %s", lastMsg),
		},
		Done: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	var req EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return a dummy 1024-dimension vector (common for BGE-M3)
	embedding := make([]float64, 1024)
	for i := range embedding {
		embedding[i] = 0.1 // Just some dummy data
	}

	resp := EmbeddingResponse{
		Embedding: embedding,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleTags(w http.ResponseWriter, r *http.Request) {
	models := map[string]interface{}{
		"models": []map[string]string{
			{"name": "llama3:latest"},
			{"name": "bge-m3:latest"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
