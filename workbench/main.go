// Command workbench is a demo and test bench for the Bastion pipeline.
//
// It talks to Sentinel, Anchor and a local Ollama model over HTTP only, so each
// module stays independent: nothing here imports module code, and any service
// whose URL is left empty is simply skipped.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
}

func main() {
	addr := flag.String("addr", env("WORKBENCH_ADDR", "127.0.0.1:8090"), "listen address (keep it on loopback)")
	profiles := flag.String("profiles", env("WORKBENCH_PROFILES", ""), "optional JSON file of profiles")
	flag.Parse()

	maxTok, _ := strconv.Atoi(env("OLLAMA_MAX_TOKENS", "300"))
	cfg := Config{
		SentinelURL: env("SENTINEL_URL", "http://localhost:8080"), SentinelToken: os.Getenv("SENTINEL_TOKEN"),
		AnchorURL: env("ANCHOR_URL", "http://localhost:8083"), AnchorToken: os.Getenv("ANCHOR_TOKEN"),
		OllamaURL: env("OLLAMA_URL", "http://localhost:11434"), OllamaModel: env("OLLAMA_MODEL", "qwen3.5:4b"),
		LLMTimeout: 120 * time.Second, MaxTokens: maxTok,
	}

	var ps []Profile
	if *profiles != "" {
		b, err := os.ReadFile(*profiles)
		if err != nil {
			log.Fatalf("profiles: %v", err)
		}
		if err := json.Unmarshal(b, &ps); err != nil {
			log.Fatalf("profiles: %v", err)
		}
	}

	srv := &http.Server{Addr: *addr, Handler: NewServer(cfg, ps).Handler(), ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Minute}
	log.Printf("workbench on http://%s  (sentinel=%q anchor=%q ollama=%q model=%q)", *addr,
		cfg.SentinelURL, cfg.AnchorURL, cfg.OllamaURL, cfg.OllamaModel)
	log.Fatal(srv.ListenAndServe())
}
