package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds where each service lives. An empty URL means "not configured":
// that stage is skipped, so the workbench works with any subset of modules.
type Config struct {
	SentinelURL, SentinelToken string
	AnchorURL, AnchorToken     string
	OllamaURL, OllamaModel     string
	LLMTimeout                 time.Duration
	MaxTokens                  int
}

type httpClient struct {
	c     *http.Client
	token string
}

func (h httpClient) do(ctx context.Context, method, url string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if h.token != "" {
		req.Header.Set("Authorization", "Bearer "+h.token)
	}
	resp, err := h.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	// 4xx from a validator (e.g. Sentinel's 403 on a block) still carries a result.
	if resp.StatusCode >= 500 || resp.StatusCode == 401 || resp.StatusCode == 404 {
		return fmt.Errorf("%s %s: HTTP %d: %s", method, url, resp.StatusCode, trim(string(raw), 200))
	}
	if out == nil {
		return nil
	}
	if resp.StatusCode >= 400 && resp.StatusCode != 403 {
		return fmt.Errorf("%s %s: HTTP %d: %s", method, url, resp.StatusCode, trim(string(raw), 200))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s %s: bad JSON (HTTP %d): %w", method, url, resp.StatusCode, err)
	}
	return nil
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// ── Sentinel ────────────────────────────────────────────────────────────────

type sentinelInResp struct {
	Status string `json:"status"`
	Checks struct {
		PromptInjection struct {
			Status          string   `json:"status"`
			RiskScore       float64  `json:"risk_score"`
			MatchedPatterns []string `json:"matched_patterns"`
		} `json:"prompt_injection"`
		MetadataValidation struct {
			Status       string   `json:"status"`
			FormatErrors []string `json:"format_errors"`
		} `json:"metadata_validation"`
	} `json:"checks"`
	UserNotice       map[string]any `json:"user_notice"`
	ProcessingTimeMs float64        `json:"processing_time_ms"`
}

func (c Config) sentinelInput(ctx context.Context, query string, meta map[string]string, strict bool) (sentinelInResp, error) {
	var out sentinelInResp
	err := httpClient{c: http.DefaultClient, token: c.SentinelToken}.do(ctx, http.MethodPost,
		c.SentinelURL+"/v1/sentinel/validate/input",
		map[string]any{"query": query, "metadata": meta,
			"options": map[string]any{"strict_mode": strict, "include_details": true}}, &out)
	return out, err
}

type sentinelOutResp struct {
	Status            string `json:"Status"`
	ValidatedResponse string `json:"ValidatedResponse"`
	Checks            struct {
		PIICheck struct {
			Status    string `json:"Status"`
			Incidents []struct {
				RuleIDs       []string `json:"RuleIDs"`
				OriginalValue string   `json:"OriginalValue"`
				ActionTaken   string   `json:"ActionTaken"`
			} `json:"Incidents"`
			RedactionsApplied int `json:"RedactionsApplied"`
		} `json:"PIICheck"`
		HallucinationCheck struct {
			Status         string   `json:"Status"`
			GroundingScore float64  `json:"GroundingScore"`
			Ungrounded     []string `json:"UngroundedClaims"`
		} `json:"HallucinationCheck"`
		ContentCheck struct {
			Status     string   `json:"Status"`
			Violations []string `json:"Violations"`
		} `json:"ContentCheck"`
		PermissionCheck struct {
			Status           string `json:"Status"`
			UserAccessLevel  string `json:"UserAccessLevel"`
			ResponseLevel    string `json:"ResponseAccessLevel"`
			BoundaryViolated bool   `json:"BoundaryViolated"`
		} `json:"PermissionCheck"`
	} `json:"Checks"`
	ErrorMessage     string         `json:"ErrorMessage"`
	Notice           map[string]any `json:"Notice"`
	ProcessingTimeMs float64        `json:"ProcessingTimeMs"`
}

func (c Config) sentinelOutput(ctx context.Context, p Profile, answer, query string, docs []string) (sentinelOutResp, error) {
	var out sentinelOutResp
	err := httpClient{c: http.DefaultClient, token: c.SentinelToken}.do(ctx, http.MethodPost,
		c.SentinelURL+"/v1/sentinel/validate/output",
		map[string]any{
			"llm_response": answer,
			"user":         map[string]any{"UserID": p.UserID, "TenantID": p.TenantID, "Department": p.Department, "AccessLevel": p.AccessLevel},
			"retrieval":    map[string]any{"Query": query, "SourceDocuments": docs},
		}, &out)
	return out, err
}

// ── Anchor ──────────────────────────────────────────────────────────────────

type anchorResp struct {
	Status       string `json:"status"`
	BiasAnalysis *struct {
		OverallScore float64 `json:"overall_score"`
		Severity     string  `json:"severity"`
		Categories   []struct {
			Category string   `json:"category"`
			Score    float64  `json:"score"`
			Evidence []string `json:"evidence"`
		} `json:"categories"`
	} `json:"bias_analysis"`
	Quality *struct {
		GroundingScore float64 `json:"grounding_score"`
		Overall        float64 `json:"overall_quality"`
	} `json:"quality"`
	Warnings          []string `json:"warnings"`
	ExceptionsApplied []string `json:"exceptions_applied"`
	ProcessingTimeMs  float64  `json:"processing_time_ms"`
}

func (c Config) anchorVerify(ctx context.Context, p Profile, answer, query string, docs []string) (anchorResp, error) {
	var out anchorResp
	if docs == nil {
		docs = []string{} // Anchor rejects null
	}
	err := httpClient{c: http.DefaultClient, token: c.AnchorToken}.do(ctx, http.MethodPost,
		c.AnchorURL+"/v1/anchor/verify",
		map[string]any{"request_id": uuidV4(), "tenant_id": p.TenantID, "llm_response": answer,
			"original_query": query, "source_documents": docs,
			"user": map[string]any{"user_id": p.UserID, "tenant_id": p.TenantID}}, &out)
	return out, err
}

// ── Ollama ──────────────────────────────────────────────────────────────────

// generate asks the local model. Thinking is disabled: reasoning models such as
// qwen3.5 otherwise spend the whole token budget on hidden thinking and return
// an empty answer.
func (c Config) generate(ctx context.Context, system, prompt string, docs []string) (string, error) {
	user := prompt
	if len(docs) > 0 {
		user = "Context:\n" + strings.Join(docs, "\n\n") + "\n\nQuestion: " + prompt
	}
	msgs := []map[string]string{}
	if system != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": system})
	}
	msgs = append(msgs, map[string]string{"role": "user", "content": user})

	ctx, cancel := context.WithTimeout(ctx, c.LLMTimeout)
	defer cancel()
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error string `json:"error"`
	}
	err := httpClient{c: http.DefaultClient}.do(ctx, http.MethodPost, c.OllamaURL+"/api/chat",
		map[string]any{"model": c.OllamaModel, "messages": msgs, "stream": false, "think": false,
			"options": map[string]any{"num_predict": c.MaxTokens, "temperature": 0.2}}, &out)
	if err != nil {
		return "", err
	}
	if out.Error != "" {
		return "", fmt.Errorf("ollama: %s", out.Error)
	}
	return strings.TrimSpace(out.Message.Content), nil
}
