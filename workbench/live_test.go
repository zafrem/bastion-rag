package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// Live tests run the real Sentinel and Anchor (and optionally Ollama).
//
//	WORKBENCH_LIVE=1                run against real services
//	WORKBENCH_LIVE_LLM=1            also call the real model
//	WORKBENCH_FAIL_ON_KNOWN=1       fail on documented known issues instead of only logging them
//
// Service URLs come from SENTINEL_URL / ANCHOR_URL / OLLAMA_URL / OLLAMA_MODEL.
func liveConfig(t *testing.T) Config {
	t.Helper()
	if os.Getenv("WORKBENCH_LIVE") == "" {
		t.Skip("set WORKBENCH_LIVE=1 to run against real services")
	}
	cfg := Config{
		SentinelURL: env("SENTINEL_URL", "http://localhost:8080"), SentinelToken: os.Getenv("SENTINEL_TOKEN"),
		AnchorURL: env("ANCHOR_URL", "http://localhost:8083"), AnchorToken: os.Getenv("ANCHOR_TOKEN"),
		OllamaModel: env("OLLAMA_MODEL", "qwen3.5:9b"), LLMTimeout: 180 * time.Second, MaxTokens: 200,
	}
	if os.Getenv("WORKBENCH_LIVE_LLM") != "" {
		cfg.OllamaURL = env("OLLAMA_URL", "http://localhost:11434")
	}
	return cfg
}

func liveCase(t *testing.T, cfg Config, caseID, profileName string) RunResult {
	t.Helper()
	for _, c := range BuiltinCases() {
		if c.ID == caseID {
			return Pipeline{cfg}.Run(context.Background(), profile(profileName),
				RunRequest{Prompt: c.Prompt, Context: c.Context, Answer: c.CannedAnswer})
		}
	}
	t.Fatalf("no case %s", caseID)
	return RunResult{}
}

func TestLive_InjectionIsBlockedForEveryProfile(t *testing.T) {
	cfg := liveConfig(t)
	for _, id := range []string{"injection", "roleplay"} {
		for _, p := range DefaultProfiles() {
			r := liveCase(t, cfg, id, p.Name)
			if r.Outcome != "blocked" || r.BlockedAt != "sentinel-in" {
				t.Errorf("%s/%s: want blocked at sentinel-in, got %s at %q", id, p.Name, r.Outcome, r.BlockedAt)
			}
		}
	}
}

func TestLive_PersonalDataNeverReachesTheUser(t *testing.T) {
	cfg := liveConfig(t)
	for _, p := range []string{"internal-staff", "external-customer"} {
		r := liveCase(t, cfg, "pii-answer", p)
		if r.Outcome != "delivered" {
			t.Fatalf("%s: %s", p, r.Outcome)
		}
		if strings.Contains(r.FinalAnswer, "john.doe@example.com") || strings.Contains(r.FinalAnswer, "010-1234-5678") {
			t.Errorf("%s: personal data leaked: %q", p, r.FinalAnswer)
		}
		r = liveCase(t, cfg, "korean-rrn", p)
		if strings.Contains(r.FinalAnswer, "900101-1234567") {
			t.Errorf("%s: resident registration number leaked: %q", p, r.FinalAnswer)
		}
	}
}

func TestLive_MalformedMetadataIsRejectedBeforeAnythingElse(t *testing.T) {
	cfg := liveConfig(t)
	r := liveCase(t, cfg, "benign-days", "external-sloppy")
	if r.Outcome != "blocked" || r.BlockedAt != "sentinel-in" {
		t.Fatalf("got %s at %q", r.Outcome, r.BlockedAt)
	}
	if ok := liveCase(t, cfg, "benign-days", "external-customer"); ok.BlockedAt == "sentinel-in" {
		t.Errorf("the same prompt with correct metadata must pass the input check")
	}
}

func TestLive_AnchorFlagsBias(t *testing.T) {
	cfg := liveConfig(t)
	r := liveCase(t, cfg, "bias", "internal-staff")
	if status(r, "anchor") != StFlagged {
		t.Errorf("want flagged, got %s (%s)", status(r, "anchor"), r.Stages[4].Summary)
	}
}

func TestLive_UngroundedAnswerIsDetected(t *testing.T) {
	cfg := liveConfig(t)
	r := liveCase(t, cfg, "ungrounded", "internal-staff")
	g, _ := r.Stages[3].Detail["grounding_score"].(float64)
	if g > 0.5 {
		t.Errorf("an answer contradicting its documents should score low on grounding, got %.2f", g)
	}
}

// KNOWN ISSUE: Sentinel's output PII rules latitude_01 / longitude_01 redact
// ordinary integers, so "within 14 days" is delivered as "within [DECIMAL
// LATITUDE ...] days". The same prompt is fine on the raw model, which makes
// internal results look different from the engine used directly.
func TestLive_KnownIssue_OrdinaryNumbersAreRedacted(t *testing.T) {
	cfg := liveConfig(t)
	for _, id := range []string{"benign-days", "benign-range"} {
		r := liveCase(t, cfg, id, "internal-staff")
		if !r.Altered {
			continue
		}
		msg := id + ": a harmless answer was altered: " + r.Stages[3].Summary + " -> " + r.FinalAnswer
		if os.Getenv("WORKBENCH_FAIL_ON_KNOWN") != "" {
			t.Error(msg)
		} else {
			t.Log("KNOWN ISSUE: " + msg)
		}
	}
}

func TestLive_RealModelAnswersWithThinkingOff(t *testing.T) {
	cfg := liveConfig(t)
	if cfg.OllamaURL == "" {
		t.Skip("set WORKBENCH_LIVE_LLM=1 to call the real model")
	}
	ans, err := cfg.generate(context.Background(), "Answer in one short sentence.", "What colour is the sky on a clear day?", nil)
	if err != nil || strings.TrimSpace(ans) == "" {
		t.Fatalf("the model must return text (err=%v, answer=%q)", err, ans)
	}
	t.Logf("model said: %s", ans)
}

func TestLive_FullPipelineWithRealModel(t *testing.T) {
	cfg := liveConfig(t)
	if cfg.OllamaURL == "" {
		t.Skip("set WORKBENCH_LIVE_LLM=1 to call the real model")
	}
	r := Pipeline{cfg}.Run(context.Background(), profile("internal-staff"),
		RunRequest{Prompt: "In one sentence, what is a refund?",
			Context: []string{"A refund returns money to the buyer."}})
	if r.Outcome != "delivered" || r.RawAnswer == "" {
		t.Fatalf("outcome=%s answer=%q stages=%+v", r.Outcome, r.RawAnswer, r.Stages)
	}
	for _, s := range r.Stages {
		if s.Status == StSkipped || s.Status == StError {
			t.Errorf("stage %s: %s (%s)", s.ID, s.Status, s.Summary)
		}
	}
}
