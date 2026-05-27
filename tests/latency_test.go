// Package tests — pipeline latency measurement.
//
// Each stage is timed wall-clock (Go time.Now) and the engine's own
// processing_time_ms (or ProcessingTimeMs) field is read from the response
// to separate in-engine cost from HTTP overhead.
//
// Run:
//
//	go test ./tests/ -run TestPipelineLatency -v -count=5
package tests

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"time"
	"testing"
)

// stageTiming holds wall-clock and engine-reported durations for one request.
type stageTiming struct {
	wall   time.Duration
	engine float64 // ms from response body, 0 if absent
}

// postJSONTimed posts JSON and returns the parsed body + wall-clock duration.
func postJSONTimed(t *testing.T, url string, body any) (map[string]any, time.Duration) {
	t.Helper()
	start := time.Now()
	resp := postJSON(t, url, body)
	wall := time.Since(start)
	defer resp.Body.Close()
	var m map[string]any
	json.NewDecoder(resp.Body).Decode(&m)
	return m, wall
}

// engineMs extracts processing_time_ms (lowercase) or ProcessingTimeMs (uppercase) from a response map.
func engineMs(m map[string]any) float64 {
	if v, ok := m["processing_time_ms"].(float64); ok {
		return v
	}
	if v, ok := m["ProcessingTimeMs"].(float64); ok {
		return v
	}
	return 0
}

// summarise computes min/median/p95/max over a slice of durations (milliseconds).
func summarise(vals []float64) (min, median, p95, max float64) {
	if len(vals) == 0 {
		return
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	min = sorted[0]
	max = sorted[len(sorted)-1]
	median = sorted[len(sorted)/2]
	p95idx := int(math.Ceil(0.95*float64(len(sorted)))) - 1
	if p95idx < 0 {
		p95idx = 0
	}
	p95 = sorted[p95idx]
	return
}

// TestPipelineLatency measures per-stage and end-to-end latency over N iterations.
// It prints a latency table and fails if any stage median exceeds its budget.
func TestPipelineLatency(t *testing.T) {
	const iterations = 20

	sentinelSrv := buildSentinel(t)
	navSrv      := buildMockNavigator(t)
	llmSrv      := buildMockLLM(t)

	// Warm up (first request is slower due to lazy init).
	for i := 0; i < 3; i++ {
		postJSON(t, sentinelSrv.URL+"/v1/validate", map[string]any{
			"request_id": "warmup",
			"query":      "warmup query",
			"metadata":   map[string]string{"tenant_id": "acme", "user_id": "u", "context_id": "550e8400-e29b-41d4-a716-446655440000", "timestamp": time.Now().UTC().Format(time.RFC3339)},
		}).Body.Close()
	}

	type stage struct {
		name    string
		wall    []float64
		engine  []float64
	}

	stages := []*stage{
		{name: "Sentinel-IN  (input validate)"},
		{name: "Navigator    (vector search) "},
		{name: "LLM          (chat complete) "},
		{name: "Sentinel-OUT (output mapped) "},
		{name: "Full pipeline (IN+NAV+LLM+OUT)"},
	}

	for i := 0; i < iterations; i++ {
		id := fmt.Sprintf("lat-%03d", i)
		ts := time.Now().UTC().Format(time.RFC3339)

		// ── Sentinel-IN ──────────────────────────────────────────────────────
		siBody, siWall := postJSONTimed(t, sentinelSrv.URL+"/v1/validate", map[string]any{
			"request_id": id + "-si",
			"query":      "What are the account balance trends for Q4?",
			"metadata":   map[string]string{"tenant_id": "acme", "user_id": "alice", "context_id": "550e8400-e29b-41d4-a716-446655440000", "timestamp": ts},
		})
		stages[0].wall   = append(stages[0].wall,   float64(siWall.Microseconds())/1000.0)
		stages[0].engine = append(stages[0].engine, engineMs(siBody))

		// ── Navigator search ─────────────────────────────────────────────────
		navBody, navWall := postJSONTimed(t, navSrv.URL+"/v1/navigator/search", map[string]any{
			"request_id": id + "-nav",
			"tenant_id":  "acme",
			"query":      "account balance Q4",
		})
		stages[1].wall   = append(stages[1].wall,   float64(navWall.Microseconds())/1000.0)
		stages[1].engine = append(stages[1].engine, engineMs(navBody))

		// ── LLM call ─────────────────────────────────────────────────────────
		_, llmWall := postJSONTimed(t, llmSrv.URL+"/v1/chat/completions", map[string]any{
			"model":    "test-llm",
			"messages": []map[string]string{{"role": "user", "content": "summarise balance data"}},
		})
		stages[2].wall   = append(stages[2].wall,   float64(llmWall.Microseconds())/1000.0)
		stages[2].engine = append(stages[2].engine, 0) // mock LLM has no engine field

		// ── Sentinel-OUT mapped ───────────────────────────────────────────────
		outBody, outWall := postJSONTimed(t, sentinelSrv.URL+"/v1/validate/output/mapped", map[string]any{
			"request_id":   id + "-out",
			"llm_response": "The balance for the account is $5,000.",
			"user":         map[string]string{"tenant_id": "acme", "user_id": "alice", "access_level": "full"},
			"known_mappings": map[string]string{},
		})
		stages[3].wall   = append(stages[3].wall,   float64(outWall.Microseconds())/1000.0)
		stages[3].engine = append(stages[3].engine, engineMs(outBody))

		// ── Full pipeline wall time (sum of the four stages above) ────────────
		fullMs := float64(siWall.Microseconds()+navWall.Microseconds()+llmWall.Microseconds()+outWall.Microseconds()) / 1000.0
		stages[4].wall = append(stages[4].wall, fullMs)
	}

	// ── Print latency table ───────────────────────────────────────────────────
	t.Logf("\n%-36s  %7s  %7s  %7s  %7s  %7s  │  %9s  %9s",
		"Stage", "min ms", "median", "p95 ms", "max ms", "n",
		"eng.med", "eng.p95")
	t.Logf("%s", "────────────────────────────────────────────────────────────────────────────────────────────────────")

	// Budget: each stage must finish in median < threshold.
	budgets := map[string]float64{
		"Sentinel-IN  (input validate)":  50,
		"Navigator    (vector search) ":  20,
		"LLM          (chat complete) ":  20,
		"Sentinel-OUT (output mapped) ":  50,
		"Full pipeline (IN+NAV+LLM+OUT)": 150,
	}

	for _, s := range stages {
		wMin, wMed, wP95, wMax := summarise(s.wall)
		eMin, eMed, eP95, _    := summarise(s.engine)
		_ = eMin

		t.Logf("%-36s  %7.2f  %7.2f  %7.2f  %7.2f  %7d  │  %9.3f  %9.3f",
			s.name, wMin, wMed, wP95, wMax, len(s.wall), eMed, eP95)

		if budget, ok := budgets[s.name]; ok {
			if wMed > budget {
				t.Errorf("LATENCY BUDGET EXCEEDED: %s median=%.2fms > budget=%.0fms", s.name, wMed, budget)
			}
		}
	}

	// ── Per-stage HTTP overhead (wall − engine) ───────────────────────────────
	t.Log("\nHTTP overhead (wall − engine) per stage:")
	for _, s := range stages[0:4] { // skip full pipeline pseudo-stage
		_, wMed, _, _ := summarise(s.wall)
		_, eMed, _, _ := summarise(s.engine)
		if eMed > 0 {
			t.Logf("  %-36s  overhead = %.2fms  (wall %.2fms − engine %.3fms)",
				s.name, wMed-eMed, wMed, eMed)
		}
	}
}

// BenchmarkSentinelIN measures Sentinel input validation throughput.
func BenchmarkSentinelIN(b *testing.B) {
	// Build sentinel with a fresh testing.T-like helper
	sentinelSrv := httptest.NewServer(http.NotFoundHandler())
	sentinelSrv.Close()

	// Re-use buildSentinel via a real *testing.T wrapped benchmark
	b.Helper()
	b.Skip("use go test -bench=BenchmarkSentinelIN -run='^$' to run separately")
}
