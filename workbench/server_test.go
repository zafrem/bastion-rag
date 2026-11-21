package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Host = "localhost:8090"
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestCompareSharesOneAnswerAndFindsTheDivergence(t *testing.T) {
	f := newFakes(t)
	f.llmAnswer = "Fine."
	h := NewServer(f.cfg(), nil).Handler()
	w := post(t, h, "/api/compare", `{"prompt":"hello","profiles":["external-customer","external-sloppy"]}`)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	var r compareResp
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	if f.llmCalls.Load() != 1 {
		t.Errorf("the model must be called once for the whole comparison, got %d", f.llmCalls.Load())
	}
	if r.Divergence != "sentinel-in" {
		t.Errorf("first divergence: %q", r.Divergence)
	}
	if r.Results[0].Outcome != "delivered" || r.Results[1].Outcome != "blocked" {
		t.Errorf("outcomes: %s / %s", r.Results[0].Outcome, r.Results[1].Outcome)
	}
	if r.Results[0].RawAnswer != "Fine." {
		t.Errorf("shared answer not used: %q", r.Results[0].RawAnswer)
	}
}

func TestCompareNeedsTwoToFourProfiles(t *testing.T) {
	h := NewServer(newFakes(t).cfg(), nil).Handler()
	for _, body := range []string{`{"prompt":"x","profiles":["internal-staff"]}`, `{"prompt":"x","profiles":["a","b"]}`} {
		if w := post(t, h, "/api/compare", body); w.Code != 400 {
			t.Errorf("%s: want 400, got %d", body, w.Code)
		}
	}
}

func TestSuiteReturnsAMatrixWithoutCallingTheModel(t *testing.T) {
	f := newFakes(t)
	h := NewServer(f.cfg(), nil).Handler()
	w := post(t, h, "/api/suite", `{"profiles":["internal-staff","external-sloppy"],"use_llm":false}`)
	var rows []Row
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != len(BuiltinCases()) {
		t.Fatalf("rows: %d", len(rows))
	}
	for _, r := range rows {
		if len(r.Cells) != 2 {
			t.Errorf("%s: want 2 cells", r.Case.ID)
		}
	}
	if f.llmCalls.Load() != 0 {
		t.Error("use_llm=false must not call the model")
	}
}

func TestGuardRejectsForeignHostsAndNonJSONPosts(t *testing.T) {
	h := NewServer(newFakes(t).cfg(), nil).Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/cases", nil)
	req.Host = "evil.example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("DNS-rebinding host must be refused, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/run", strings.NewReader("prompt=x"))
	req.Host = "localhost:8090"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 415 {
		t.Errorf("cross-site form posts must be refused, got %d", w.Code)
	}
}

func TestUnknownProfileIsABadRequest(t *testing.T) {
	h := NewServer(newFakes(t).cfg(), nil).Handler()
	if w := post(t, h, "/api/run", `{"prompt":"x","profile":"nope"}`); w.Code != 400 {
		t.Errorf("got %d", w.Code)
	}
}

func TestStatusReportsWhichServicesAreUp(t *testing.T) {
	f := newFakes(t)
	h := NewServer(f.cfg(), nil).Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.Host = "127.0.0.1:8090"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var st []serviceStatus
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if len(st) != 3 || !st[0].Up {
		t.Fatalf("%+v", st)
	}
	if st[2].Up {
		t.Error("ollama fake does not list the model, so it must be reported as not ready")
	}
}

func TestEveryBuiltinCaseHasAPromptAndAnAnswer(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range BuiltinCases() {
		if c.ID == "" || c.Prompt == "" || c.CannedAnswer == "" || c.Why == "" || seen[c.ID] {
			t.Errorf("bad case %+v", c)
		}
		seen[c.ID] = true
	}
}
