package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// The default dataset ships inside the binary; -data points at another file.
//
//go:embed data/cases.jsonl
var defaultDataset []byte

// Case is one labelled example: an input, the model output under test, and what
// the correct behaviour is. Expectations describe how Bastion SHOULD behave, not
// how it currently does, so a failing case is a finding.
type Case struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Lang     string   `json:"lang,omitempty"`
	Profiles []string `json:"profiles,omitempty"` // default: internal-staff
	// Input
	Prompt  string   `json:"prompt"`
	Context []string `json:"context,omitempty"`
	// Output under test: stands in for the model so scoring is repeatable.
	// Optional: when empty, the live model writes one (use_llm) so real prompts can be run as-is.
	ModelAnswer string `json:"model_answer,omitempty"`
	// Ground truth
	Expect Expect `json:"expect"`
	Notes  string `json:"notes,omitempty"`
}

// Expect lists the checks a case is scored on. Unset fields are not checked.
type Expect struct {
	Outcome        string   `json:"outcome,omitempty"`    // "delivered" or "blocked"
	BlockedAt      string   `json:"blocked_at,omitempty"` // "sentinel-in" or "sentinel-out"
	Unchanged      *bool    `json:"unchanged,omitempty"`  // delivered text equals the model's text
	MustNotContain []string `json:"must_not_contain,omitempty"`
	MustContain    []string `json:"must_contain,omitempty"`
	Bias           string   `json:"bias,omitempty"`      // "flagged" or "clean" (Anchor)
	Grounding      string   `json:"grounding,omitempty"` // "high" (>=0.5) or "low" (<0.5)
}

// Categories the dataset may use, with what each one measures.
var Categories = map[string]string{
	"benign":    "Harmless traffic that must pass untouched (false-positive check)",
	"injection": "Prompt-injection and jailbreak attempts that must be blocked",
	"pii":       "Personal data or secrets in a model answer that must not reach the user",
	"bias":      "Biased answers Anchor should flag, and neutral near-misses it should not",
	"grounding": "Answers checked against supplied documents",
	"content":   "Harmful content in an answer",
	"metadata":  "Callers whose request metadata is malformed",
	"policy":    "Cases where the policy profile decides the outcome: the same input is expected to be treated differently by audience",
	"unlabeled": "Real traffic without labels: measured for behaviour and latency, not graded",
}

// ExpectedIntervene reports whether the controls should act on this case, and
// whether the case states that clearly enough to count in the confusion matrix.
func (c Case) ExpectedIntervene() (want, known bool) {
	e := c.Expect
	switch {
	case e.Outcome == "blocked", e.Bias == "flagged", len(e.MustNotContain) > 0,
		e.Unchanged != nil && !*e.Unchanged:
		return true, true
	case e.Unchanged != nil && *e.Unchanged, e.Bias == "clean":
		return false, true
	}
	return false, false
}

func (c Case) profileNames() []string {
	if len(c.Profiles) == 0 {
		return []string{"internal-staff"}
	}
	return c.Profiles
}

// LoadDataset parses JSON Lines (one case per line; blank lines and # comments are skipped).
func LoadDataset(raw []byte) ([]Case, error) {
	var cases []Case
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var c Case
		dec := json.NewDecoder(strings.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		cases = append(cases, c)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}

func LoadDatasetFile(path string) ([]Case, error) {
	if path == "" {
		return LoadDataset(defaultDataset)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadAuto(b)
}

// MaxCases bounds one run so a huge paste cannot exhaust memory.
const MaxCases = 50000

// LoadAuto detects the format of real-world input:
//
//	JSON Lines  first line starts with "{"            (full labelled cases)
//	CSV         first line has a "prompt" column      (export from a spreadsheet or database)
//	plain text  anything else: one prompt per line    (raw traffic, unlabelled)
func LoadAuto(raw []byte) ([]Case, error) {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")) // BOM from spreadsheet exports
	first := ""
	for _, ln := range strings.Split(string(raw), "\n") {
		if t := strings.TrimSpace(ln); t != "" && !strings.HasPrefix(t, "#") {
			first = t
			break
		}
	}
	var cases []Case
	var err error
	switch {
	case first == "":
		return nil, fmt.Errorf("the input is empty")
	case strings.HasPrefix(first, "{"):
		cases, err = LoadDataset(raw)
	case strings.Contains(strings.ToLower(first), "prompt") && strings.Contains(first, ","):
		cases, err = LoadCSV(raw)
	default:
		cases = LoadLines(raw)
	}
	if err != nil {
		return nil, err
	}
	if len(cases) > MaxCases {
		return nil, fmt.Errorf("%d cases is over the limit of %d", len(cases), MaxCases)
	}
	return cases, nil
}

// LoadLines treats each non-empty line as one unlabelled prompt.
func LoadLines(raw []byte) []Case {
	var cases []Case
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		cases = append(cases, Case{ID: fmt.Sprintf("line-%d", len(cases)+1), Category: "unlabeled", Prompt: t})
	}
	return cases
}

// LoadCSV reads a header row then one case per row. Recognised columns (any
// order; only prompt is required): id, category, lang, profiles, prompt,
// context, model_answer, outcome, blocked_at, unchanged, must_not_contain,
// must_contain, bias, grounding, notes. List cells use "|" as the separator.
func LoadCSV(raw []byte) ([]Case, error) {
	rd := csv.NewReader(bytes.NewReader(raw))
	rd.FieldsPerRecord = -1
	rd.LazyQuotes = true
	rows, err := rd.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("csv: needs a header row and at least one data row")
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	if _, ok := col["prompt"]; !ok {
		return nil, fmt.Errorf("csv: no \"prompt\" column")
	}
	get := func(row []string, name string) string {
		if i, ok := col[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}
	list := func(s string) []string {
		var out []string
		for _, p := range strings.Split(s, "|") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	var cases []Case
	for n, row := range rows[1:] {
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		c := Case{ID: get(row, "id"), Category: get(row, "category"), Lang: get(row, "lang"),
			Profiles: list(get(row, "profiles")), Prompt: get(row, "prompt"), Context: list(get(row, "context")),
			ModelAnswer: get(row, "model_answer"), Notes: get(row, "notes")}
		if c.ID == "" {
			c.ID = fmt.Sprintf("row-%d", n+1)
		}
		e := &c.Expect
		e.Outcome, e.BlockedAt = get(row, "outcome"), get(row, "blocked_at")
		e.Bias, e.Grounding = get(row, "bias"), get(row, "grounding")
		e.MustNotContain, e.MustContain = list(get(row, "must_not_contain")), list(get(row, "must_contain"))
		switch strings.ToLower(get(row, "unchanged")) {
		case "true", "yes", "1":
			t := true
			e.Unchanged = &t
		case "false", "no", "0":
			f := false
			e.Unchanged = &f
		}
		if c.Category == "" {
			if e.Outcome == "" && e.BlockedAt == "" && e.Unchanged == nil && e.Bias == "" &&
				e.Grounding == "" && len(e.MustNotContain) == 0 && len(e.MustContain) == 0 {
				c.Category = "unlabeled"
			} else {
				return nil, fmt.Errorf("csv row %d (%s): has expectations but no category", n+2, c.ID)
			}
		}
		cases = append(cases, c)
	}
	return cases, nil
}

// ValidateDataset checks structure, so a typo in the data fails loudly instead
// of silently scoring nothing.
func ValidateDataset(cases []Case, profileNames map[string]bool) []string {
	var problems []string
	seen := map[string]bool{}
	for _, c := range cases {
		bad := func(msg string) { problems = append(problems, c.ID+": "+msg) }
		switch {
		case c.ID == "":
			problems = append(problems, "a case has no id")
			continue
		case seen[c.ID]:
			bad("duplicate id")
		}
		seen[c.ID] = true
		if _, ok := Categories[c.Category]; !ok {
			bad("unknown category " + c.Category)
		}
		if strings.TrimSpace(c.Prompt) == "" {
			bad("empty prompt")
		}
		if c.Category != "unlabeled" {
			if strings.TrimSpace(c.ModelAnswer) == "" {
				bad("empty model_answer (only unlabeled cases may omit it)")
			}
			if _, known := c.ExpectedIntervene(); !known && c.Expect.Grounding == "" && c.Expect.Outcome == "" &&
				len(c.Expect.MustContain) == 0 {
				bad("expect states nothing checkable (use category \"unlabeled\" for unlabelled data)")
			}
		}
		if o := c.Expect.Outcome; o != "" && o != "delivered" && o != "blocked" {
			bad("outcome must be delivered or blocked")
		}
		if b := c.Expect.BlockedAt; b != "" && b != "sentinel-in" && b != "sentinel-out" {
			bad("blocked_at must be sentinel-in or sentinel-out")
		}
		if b := c.Expect.Bias; b != "" && b != "flagged" && b != "clean" {
			bad("bias must be flagged or clean")
		}
		if g := c.Expect.Grounding; g != "" && g != "high" && g != "low" {
			bad("grounding must be high or low")
		}
		// A leak check is only meaningful if the leak is really in the model's answer.
		for _, s := range c.Expect.MustNotContain {
			if !strings.Contains(c.ModelAnswer, s) {
				bad(fmt.Sprintf("must_not_contain %q is not in model_answer, so the check proves nothing", s))
			}
		}
		for _, s := range c.Expect.MustContain {
			if !strings.Contains(c.ModelAnswer, s) {
				bad(fmt.Sprintf("must_contain %q is not in model_answer", s))
			}
		}
		for _, p := range c.profileNames() {
			if profileNames != nil && !profileNames[p] {
				bad("unknown profile " + p)
			}
		}
	}
	return problems
}

// CategoryCounts returns how many cases each category has, in stable order.
func CategoryCounts(cases []Case) (names []string, counts map[string]int) {
	counts = map[string]int{}
	for _, c := range cases {
		counts[c.Category]++
	}
	for n := range counts {
		names = append(names, n)
	}
	sort.Strings(names)
	return
}
