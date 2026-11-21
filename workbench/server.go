package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed ui
var uiFS embed.FS

type Server struct {
	Cfg      Config
	mu       sync.RWMutex
	profiles []Profile
}

func NewServer(cfg Config, profiles []Profile) *Server {
	if len(profiles) == 0 {
		profiles = DefaultProfiles()
	}
	return &Server{Cfg: cfg, profiles: profiles}
}

func (s *Server) profile(name string) (Profile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(uiFS, "ui")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/profiles", s.handleProfiles)
	mux.HandleFunc("/api/cases", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, BuiltinCases()) })
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/compare", s.handleCompare)
	mux.HandleFunc("/api/suite", s.handleSuite)
	return guard(mux)
}

// guard makes this safe to leave running on a developer machine: it rejects
// requests whose Host is not a loopback name (DNS rebinding) and requires JSON
// for anything that changes or runs something (blocks cross-site form posts).
func guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "localhost" && host != "127.0.0.1" && host != "::1" && host != "[::1]" {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'none'")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "use POST"})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return false
	}
	return true
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var ps []Profile
		if !decode(w, r, &ps) {
			return
		}
		for _, p := range ps {
			if p.Name == "" || p.TenantID == "" {
				writeJSON(w, 400, map[string]string{"error": "every profile needs a name and tenant_id"})
				return
			}
		}
		s.mu.Lock()
		s.profiles = ps
		s.mu.Unlock()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, 200, s.profiles)
}

type serviceStatus struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Up      bool   `json:"up"`
	Detail  string `json:"detail"`
	Enabled bool   `json:"enabled"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	get := func(url string, out any) error {
		return httpClient{c: http.DefaultClient}.do(ctx, http.MethodGet, url, nil, out)
	}
	st := []serviceStatus{
		{Name: "sentinel", URL: s.Cfg.SentinelURL, Enabled: s.Cfg.SentinelURL != ""},
		{Name: "anchor", URL: s.Cfg.AnchorURL, Enabled: s.Cfg.AnchorURL != ""},
		{Name: "ollama", URL: s.Cfg.OllamaURL, Enabled: s.Cfg.OllamaURL != ""},
	}
	var wg sync.WaitGroup
	check := func(i int, f func() (bool, string)) {
		if !st[i].Enabled {
			st[i].Detail = "not configured"
			return
		}
		wg.Add(1)
		go func() { defer wg.Done(); st[i].Up, st[i].Detail = f() }()
	}
	check(0, func() (bool, string) {
		var h struct{ Status, Version string }
		if err := get(s.Cfg.SentinelURL+"/v1/health", &h); err != nil {
			return false, err.Error()
		}
		return true, "v" + h.Version
	})
	check(1, func() (bool, string) {
		if err := get(s.Cfg.AnchorURL+"/v1/health", nil); err != nil {
			return false, err.Error()
		}
		return true, "ok"
	})
	check(2, func() (bool, string) {
		var t struct {
			Models []struct{ Name string } `json:"models"`
		}
		if err := get(s.Cfg.OllamaURL+"/api/tags", &t); err != nil {
			return false, err.Error()
		}
		for _, m := range t.Models {
			if m.Name == s.Cfg.OllamaModel {
				return true, s.Cfg.OllamaModel
			}
		}
		return false, fmt.Sprintf("model %q is not installed", s.Cfg.OllamaModel)
	})
	wg.Wait()
	writeJSON(w, 200, st)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var rq RunRequest
	if !decode(w, r, &rq) {
		return
	}
	p, ok := s.profile(rq.Profile)
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "unknown profile"})
		return
	}
	writeJSON(w, 200, Pipeline{s.Cfg}.Run(r.Context(), p, rq))
}

type compareReq struct {
	Prompt   string   `json:"prompt"`
	Profiles []string `json:"profiles"`
	Context  []string `json:"context"`
	Observe  bool     `json:"observe"`
	Answer   string   `json:"answer"`
}

type compareResp struct {
	Answer     string      `json:"shared_answer"`
	Results    []RunResult `json:"results"`
	Divergence string      `json:"first_divergence"`
}

// handleCompare runs the same prompt under several profiles with ONE model
// answer shared by all of them, so any difference comes from the controls and
// not from the model's randomness.
func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	var rq compareReq
	if !decode(w, r, &rq) {
		return
	}
	if len(rq.Profiles) < 2 || len(rq.Profiles) > 4 {
		writeJSON(w, 400, map[string]string{"error": "choose 2 to 4 profiles"})
		return
	}
	var ps []Profile
	for _, n := range rq.Profiles {
		p, ok := s.profile(n)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "unknown profile " + n})
			return
		}
		ps = append(ps, p)
	}
	answer := rq.Answer
	if answer == "" && s.Cfg.OllamaURL != "" {
		a, err := s.Cfg.generate(r.Context(), ps[0].System, rq.Prompt, rq.Context)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "model: " + err.Error()})
			return
		}
		answer = a
	}
	resp := compareResp{Answer: answer}
	pl := Pipeline{s.Cfg}
	for _, p := range ps {
		resp.Results = append(resp.Results, pl.Run(r.Context(), p, RunRequest{
			Prompt: rq.Prompt, Profile: p.Name, Context: rq.Context, Observe: rq.Observe, Answer: answer}))
	}
	for i := 1; i < len(resp.Results); i++ {
		if d := FirstDivergence(resp.Results[0], resp.Results[i]); d != "" {
			resp.Divergence = d
			break
		}
	}
	writeJSON(w, 200, resp)
}

type suiteReq struct {
	Profiles []string `json:"profiles"`
	Observe  bool     `json:"observe"`
	UseLLM   bool     `json:"use_llm"`
	Cases    []string `json:"cases"`
}

func (s *Server) handleSuite(w http.ResponseWriter, r *http.Request) {
	var rq suiteReq
	if !decode(w, r, &rq) {
		return
	}
	if len(rq.Profiles) == 0 {
		writeJSON(w, 400, map[string]string{"error": "choose at least one profile"})
		return
	}
	want := map[string]bool{}
	for _, c := range rq.Cases {
		want[c] = true
	}
	var ps []Profile
	for _, n := range rq.Profiles {
		p, ok := s.profile(n)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "unknown profile " + n})
			return
		}
		ps = append(ps, p)
	}
	pl := Pipeline{s.Cfg}
	var rows []Row
	for _, c := range BuiltinCases() {
		if len(want) > 0 && !want[c.ID] {
			continue
		}
		answer := c.CannedAnswer
		if rq.UseLLM && s.Cfg.OllamaURL != "" {
			a, err := s.Cfg.generate(r.Context(), ps[0].System, c.Prompt, c.Context)
			if err != nil {
				writeJSON(w, 502, map[string]string{"error": "model: " + err.Error()})
				return
			}
			answer = a
		}
		row := Row{Case: c, Raw: answer}
		for _, p := range ps {
			row.Cells = append(row.Cells, cellOf(pl.Run(r.Context(), p, RunRequest{
				Prompt: c.Prompt, Profile: p.Name, Context: c.Context, Observe: rq.Observe, Answer: answer})))
		}
		rows = append(rows, row)
	}
	writeJSON(w, 200, rows)
}
