package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Stage statuses. "would_block" only appears in observe mode, where a blocked
// stage is recorded but the request keeps flowing so later stages can be seen.
const (
	StPassed     = "passed"
	StModified   = "modified"
	StBlocked    = "blocked"
	StWouldBlock = "would_block"
	StFlagged    = "flagged"
	StSkipped    = "skipped"
	StError      = "error"
	StReused     = "reused"
)

type Stage struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Module     string         `json:"module"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary"`
	DurationMs float64        `json:"duration_ms"`
	Detail     map[string]any `json:"detail,omitempty"`
	Notice     map[string]any `json:"notice,omitempty"`
}

type RunRequest struct {
	Prompt  string   `json:"prompt"`
	Profile string   `json:"profile"`
	Context []string `json:"context"`
	// Observe keeps the pipeline running past a block, so you can see what the
	// controls would have prevented.
	Observe bool `json:"observe"`
	// Answer, when set, replaces the LLM call (used to compare profiles on an identical answer).
	Answer string `json:"answer"`
}

type RunResult struct {
	Profile     string  `json:"profile"`
	Prompt      string  `json:"prompt"`
	Stages      []Stage `json:"stages"`
	RawAnswer   string  `json:"raw_answer"`
	FinalAnswer string  `json:"final_answer"`
	// Outcome: delivered | blocked | error
	Outcome   string `json:"outcome"`
	BlockedAt string `json:"blocked_at,omitempty"`
	// WouldBlockAt is the first stage that would have blocked, in observe mode.
	WouldBlockAt string `json:"would_block_at,omitempty"`
	// Altered is true when the user receives different text than the model wrote.
	Altered bool    `json:"altered"`
	TotalMs float64 `json:"total_ms"`
}

type Pipeline struct{ Cfg Config }

func ms(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000 }

func (p Pipeline) Run(ctx context.Context, pr Profile, rq RunRequest) RunResult {
	start := time.Now()
	res := RunResult{Profile: pr.Name, Prompt: rq.Prompt, Outcome: "delivered"}
	blocked := false // true once a stage blocked (enforce mode)

	block := func(s *Stage, id string) {
		if rq.Observe {
			s.Status = StWouldBlock
			if res.WouldBlockAt == "" {
				res.WouldBlockAt = id
			}
			return
		}
		s.Status = StBlocked
		blocked = true
		res.Outcome, res.BlockedAt = "blocked", id
	}
	skipped := func(id, name, module, why string) {
		res.Stages = append(res.Stages, Stage{ID: id, Name: name, Module: module, Status: StSkipped, Summary: why})
	}
	fail := func(s Stage, err error) {
		s.Status, s.Summary = StError, err.Error()
		res.Stages = append(res.Stages, s)
		res.Outcome, res.BlockedAt = "error", s.ID // fail closed: never deliver unchecked text
		blocked = true
	}

	// 1 ── Sentinel input
	in := Stage{ID: "sentinel-in", Name: "Input check", Module: "sentinel"}
	switch {
	case p.Cfg.SentinelURL == "":
		skipped(in.ID, in.Name, in.Module, "Sentinel not configured")
	default:
		t := time.Now()
		r, err := p.Cfg.sentinelInput(ctx, rq.Prompt, pr.inputMetadata(time.Now()), pr.Strict)
		in.DurationMs = ms(t)
		if err != nil {
			fail(in, err)
			break
		}
		in.Detail = map[string]any{
			"injection_score": r.Checks.PromptInjection.RiskScore,
			"patterns":        r.Checks.PromptInjection.MatchedPatterns,
			"metadata_errors": r.Checks.MetadataValidation.FormatErrors,
		}
		if r.Profile != "" {
			in.Detail["policy_profile"] = r.Profile
		}
		in.Notice = r.UserNotice
		switch r.Status {
		case "BLOCKED":
			block(&in, in.ID)
			in.Summary = inputReason(r)
		default:
			in.Status, in.Summary = StPassed, fmt.Sprintf("injection score %.2f, metadata ok", r.Checks.PromptInjection.RiskScore)
		}
		res.Stages = append(res.Stages, in)
	}
	if res.Outcome == "error" {
		return finish(res, start)
	}

	// 2 ── Context (supplied by the caller; Navigator/Vault are not required)
	if len(rq.Context) > 0 {
		res.Stages = append(res.Stages, Stage{ID: "context", Name: "Context", Module: "navigator", Status: StPassed,
			Summary: fmt.Sprintf("%d supplied document(s)", len(rq.Context))})
	} else {
		skipped("context", "Context", "navigator", "no context supplied (grounding is not tested)")
	}

	// 3 ── LLM
	llm := Stage{ID: "llm", Name: "Model", Module: "ollama"}
	switch {
	case blocked:
		skipped(llm.ID, llm.Name, llm.Module, "not called: blocked upstream")
	case rq.Answer != "":
		res.RawAnswer = rq.Answer
		llm.Status, llm.Summary = StReused, "same answer reused so only the controls differ"
		res.Stages = append(res.Stages, llm)
	case p.Cfg.OllamaURL == "":
		skipped(llm.ID, llm.Name, llm.Module, "Ollama not configured")
	default:
		t := time.Now()
		ans, err := p.Cfg.generate(ctx, pr.System, rq.Prompt, rq.Context)
		llm.DurationMs = ms(t)
		if err != nil {
			fail(llm, err)
			break
		}
		if ans == "" {
			fail(llm, fmt.Errorf("model returned an empty answer"))
			break
		}
		res.RawAnswer = ans
		llm.Status, llm.Summary = StPassed, fmt.Sprintf("%s, %d characters", p.Cfg.OllamaModel, len(ans))
		res.Stages = append(res.Stages, llm)
	}
	if res.Outcome == "error" || res.RawAnswer == "" {
		if res.Outcome == "delivered" {
			if p.Cfg.OllamaURL == "" && rq.Answer == "" {
				res.Outcome = "input_only" // only the input check could run: there is no model answer
			} else {
				res.Outcome = "blocked" // nothing to deliver
			}
		}
		p.fillRemaining(&res, true)
		return finish(res, start)
	}

	text := res.RawAnswer

	// 4 ── Sentinel output
	out := Stage{ID: "sentinel-out", Name: "Output check", Module: "sentinel"}
	if p.Cfg.SentinelURL == "" {
		skipped(out.ID, out.Name, out.Module, "Sentinel not configured")
	} else if blocked {
		skipped(out.ID, out.Name, out.Module, "not run: blocked upstream")
	} else {
		t := time.Now()
		r, err := p.Cfg.sentinelOutput(ctx, pr, text, rq.Prompt, rq.Context)
		out.DurationMs = ms(t)
		if err != nil {
			fail(out, err)
		} else {
			rules := map[string]int{}
			for _, i := range r.Checks.PIICheck.Incidents {
				for _, id := range i.RuleIDs {
					rules[id]++
				}
			}
			out.Detail = map[string]any{
				"pii_rules":       rules,
				"redactions":      r.Checks.PIICheck.RedactionsApplied,
				"grounding_score": r.Checks.HallucinationCheck.GroundingScore,
				"content":         r.Checks.ContentCheck.Violations,
				"permission":      r.Checks.PermissionCheck.Status,
				"response_level":  r.Checks.PermissionCheck.ResponseLevel,
				"user_level":      r.Checks.PermissionCheck.UserAccessLevel,
			}
			if r.Profile != "" {
				out.Detail["policy_profile"] = r.Profile
			}
			out.Notice = r.Notice
			switch r.Status {
			case "BLOCKED":
				block(&out, out.ID)
				out.Summary = outputReason(r)
			case "SANITIZED":
				out.Status, out.Summary = StModified, outputReason(r)
				text = r.ValidatedResponse
			default:
				out.Status, out.Summary = StPassed, "no changes"
			}
			res.Stages = append(res.Stages, out)
		}
	}

	// 5 ── Anchor
	an := Stage{ID: "anchor", Name: "Bias & quality", Module: "anchor"}
	switch {
	case p.Cfg.AnchorURL == "":
		skipped(an.ID, an.Name, an.Module, "Anchor not configured")
	case blocked:
		skipped(an.ID, an.Name, an.Module, "not run: blocked upstream")
	default:
		t := time.Now()
		r, err := p.Cfg.anchorVerify(ctx, pr, text, rq.Prompt, rq.Context)
		an.DurationMs = ms(t)
		if err != nil {
			// Anchor is advisory: report it but still deliver what Sentinel cleared.
			an.Status, an.Summary = StError, err.Error()
		} else {
			an.Detail = map[string]any{"warnings": r.Warnings}
			if r.BiasAnalysis != nil {
				an.Detail["bias_score"] = r.BiasAnalysis.OverallScore
				cats := []string{}
				for _, c := range r.BiasAnalysis.Categories {
					cats = append(cats, c.Category)
				}
				an.Detail["bias_categories"] = cats
			}
			if r.Quality != nil {
				an.Detail["quality"] = r.Quality.Overall
			}
			an.Detail["exceptions_applied"] = r.ExceptionsApplied
			if r.Status == "PASSED" {
				an.Status, an.Summary = StPassed, "no bias or quality concerns"
			} else {
				an.Status = StFlagged
				an.Summary = r.Status + ": " + strings.Join(firstN(r.Warnings, 2), "; ")
			}
		}
		res.Stages = append(res.Stages, an)
	}
	if res.Outcome == "error" {
		p.fillRemaining(&res, false)
		return finish(res, start)
	}

	if res.Outcome == "delivered" {
		res.FinalAnswer = text
	}
	return finish(res, start)
}

// fillRemaining marks stages that never ran, so the UI always shows all five.
func (p Pipeline) fillRemaining(res *RunResult, fromLLM bool) {
	have := map[string]bool{}
	for _, s := range res.Stages {
		have[s.ID] = true
	}
	for _, d := range []struct{ id, name, mod string }{
		{"sentinel-in", "Input check", "sentinel"}, {"context", "Context", "navigator"}, {"llm", "Model", "ollama"},
		{"sentinel-out", "Output check", "sentinel"}, {"anchor", "Bias & quality", "anchor"},
	} {
		if !have[d.id] {
			res.Stages = append(res.Stages, Stage{ID: d.id, Name: d.name, Module: d.mod, Status: StSkipped, Summary: "not run"})
		}
	}
	order := map[string]int{"sentinel-in": 0, "context": 1, "llm": 2, "sentinel-out": 3, "anchor": 4}
	for i := range res.Stages {
		for j := i + 1; j < len(res.Stages); j++ {
			if order[res.Stages[j].ID] < order[res.Stages[i].ID] {
				res.Stages[i], res.Stages[j] = res.Stages[j], res.Stages[i]
			}
		}
	}
}

func finish(res RunResult, start time.Time) RunResult {
	if len(res.Stages) < 5 {
		(Pipeline{}).fillRemaining(&res, false)
	}
	res.Altered = res.Outcome == "delivered" && res.FinalAnswer != "" && res.FinalAnswer != res.RawAnswer
	res.TotalMs = ms(start)
	return res
}

func firstN(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func inputReason(r sentinelInResp) string {
	var why []string
	if r.Checks.PromptInjection.Status == "BLOCKED" {
		why = append(why, fmt.Sprintf("injection (score %.2f, %s)", r.Checks.PromptInjection.RiskScore,
			strings.Join(r.Checks.PromptInjection.MatchedPatterns, ",")))
	}
	if r.Checks.MetadataValidation.Status == "BLOCKED" {
		why = append(why, "metadata: "+strings.Join(r.Checks.MetadataValidation.FormatErrors, "; "))
	}
	if len(why) == 0 {
		return "blocked"
	}
	return strings.Join(why, " + ")
}

func outputReason(r sentinelOutResp) string {
	var why []string
	if n := r.Checks.PIICheck.RedactionsApplied; n > 0 {
		rules := map[string]bool{}
		for _, i := range r.Checks.PIICheck.Incidents {
			for _, id := range i.RuleIDs {
				rules[id] = true
			}
		}
		ids := []string{}
		for id := range rules {
			ids = append(ids, id)
		}
		sortStrings(ids)
		why = append(why, fmt.Sprintf("%d redaction(s) by %s", n, strings.Join(ids, ",")))
	}
	if r.Checks.PermissionCheck.BoundaryViolated {
		why = append(why, "access level exceeded")
	}
	if len(r.Checks.ContentCheck.Violations) > 0 {
		why = append(why, "content: "+strings.Join(r.Checks.ContentCheck.Violations, ","))
	}
	if r.Checks.HallucinationCheck.Status != "" && r.Checks.HallucinationCheck.Status != "PASSED" {
		why = append(why, fmt.Sprintf("grounding %.2f", r.Checks.HallucinationCheck.GroundingScore))
	}
	if len(why) == 0 {
		return strings.ToLower(r.Status)
	}
	return strings.Join(why, " + ")
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
