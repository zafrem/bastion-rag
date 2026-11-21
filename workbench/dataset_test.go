package main

import (
	"strings"
	"testing"
)

func TestBuiltinDatasetIsValidAndBalanced(t *testing.T) {
	cases, err := LoadDatasetFile("")
	if err != nil {
		t.Fatal(err)
	}
	if probs := ValidateDataset(cases, nameSet(DefaultProfiles())); len(probs) > 0 {
		t.Fatalf("the shipped dataset has problems:\n  %s", strings.Join(probs, "\n  "))
	}
	_, counts := CategoryCounts(cases)
	for cat, min := range map[string]int{"benign": 15, "injection": 10, "pii": 10, "bias": 5, "grounding": 3, "content": 2, "metadata": 2} {
		if counts[cat] < min {
			t.Errorf("category %s has %d cases, want at least %d", cat, counts[cat], min)
		}
	}
	langs := map[string]bool{}
	for _, c := range cases {
		langs[c.Lang] = true
	}
	if !langs["ko"] {
		t.Error("the dataset should include Korean cases")
	}
}

func TestLoadAutoDetectsEveryFormat(t *testing.T) {
	jsonl := `# comment
{"id":"a","category":"benign","prompt":"hi","model_answer":"hello there friend","expect":{"unchanged":true}}

{"id":"b","category":"unlabeled","prompt":"yo"}`
	csv := "id,category,prompt,model_answer,unchanged\nc1,benign,hi,hello there friend,true\nc2,,plain question,,\n"
	lines := "# prompts\nwhat is X?\n\nhow about Y?\n"
	for name, tc := range map[string]struct {
		in   string
		want int
		cat0 string
	}{
		"jsonl": {jsonl, 2, "benign"},
		"csv":   {csv, 2, "benign"},
		"lines": {lines, 2, "unlabeled"},
		"bom":   {"\xef\xbb\xbfid,prompt\nz,hello\n", 1, "unlabeled"},
	} {
		got, err := LoadAuto([]byte(tc.in))
		if err != nil || len(got) != tc.want || got[0].Category != tc.cat0 {
			t.Errorf("%s: got %d cases cat0=%q err=%v", name, len(got), func() string {
				if len(got) > 0 {
					return got[0].Category
				}
				return ""
			}(), err)
		}
	}
	got, _ := LoadAuto([]byte(csv))
	if got[0].Expect.Unchanged == nil || !*got[0].Expect.Unchanged {
		t.Error("csv unchanged=true must become an expectation")
	}
	if got[1].Category != "unlabeled" {
		t.Errorf("a csv row with no category and no expectations is unlabelled, got %q", got[1].Category)
	}
}

func TestLoadAutoRejectsBadInputWithUsefulErrors(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"empty":                       {"  \n\n", "empty"},
		"bad json":                    {"{\"id\":\"a\"}\n{not json}", "line 2"},
		"unknown field":               {`{"id":"a","prompt":"x","bogus":1}`, "bogus"},
		"csv no prompt":               {"id,text\n1,hi", ""}, // not detected as csv: becomes lines, so no error
		"csv labels without category": {"id,prompt,outcome\n1,hi,blocked\n", "no category"},
	} {
		_, err := LoadAuto([]byte(tc.in))
		if tc.want == "" {
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: want error containing %q, got %v", name, tc.want, err)
		}
	}
	big := strings.Repeat("a prompt\n", MaxCases+1)
	if _, err := LoadAuto([]byte(big)); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Errorf("an oversized input must be refused, got %v", err)
	}
}

func TestValidateCatchesDataMistakes(t *testing.T) {
	yes := true
	cases := []Case{
		{ID: "dup", Category: "benign", Prompt: "p", ModelAnswer: "a", Expect: Expect{Unchanged: &yes}},
		{ID: "dup", Category: "benign", Prompt: "p", ModelAnswer: "a", Expect: Expect{Unchanged: &yes}},
		{ID: "cat", Category: "nonsense", Prompt: "p", ModelAnswer: "a", Expect: Expect{Unchanged: &yes}},
		{ID: "leak", Category: "pii", Prompt: "p", ModelAnswer: "no secret here", Expect: Expect{MustNotContain: []string{"SECRET"}}},
		{ID: "prof", Category: "benign", Prompt: "p", ModelAnswer: "a", Profiles: []string{"ghost"}, Expect: Expect{Unchanged: &yes}},
		{ID: "empty", Category: "benign", Prompt: "p", ModelAnswer: "a"},
		{ID: "noans", Category: "benign", Prompt: "p", Expect: Expect{Unchanged: &yes}},
		{ID: "badout", Category: "benign", Prompt: "p", ModelAnswer: "a", Expect: Expect{Outcome: "maybe"}},
	}
	probs := strings.Join(ValidateDataset(cases, nameSet(DefaultProfiles())), "\n")
	for _, want := range []string{"duplicate id", "unknown category", "proves nothing", "unknown profile", "states nothing", "empty model_answer", "outcome must be"} {
		if !strings.Contains(probs, want) {
			t.Errorf("missing problem %q in:\n%s", want, probs)
		}
	}
	if probs := ValidateDataset([]Case{{ID: "u", Category: "unlabeled", Prompt: "free text"}}, nil); len(probs) > 0 {
		t.Errorf("an unlabelled case needs only a prompt: %v", probs)
	}
}

func TestExpectedIntervene(t *testing.T) {
	yes, no := true, false
	for name, tc := range map[string]struct {
		e           Expect
		want, known bool
	}{
		"blocked":   {Expect{Outcome: "blocked"}, true, true},
		"leak":      {Expect{MustNotContain: []string{"x"}}, true, true},
		"flagged":   {Expect{Bias: "flagged"}, true, true},
		"changed":   {Expect{Unchanged: &no}, true, true},
		"unchaged":  {Expect{Unchanged: &yes}, false, true},
		"clean":     {Expect{Bias: "clean"}, false, true},
		"grounding": {Expect{Grounding: "low"}, false, false},
	} {
		w, k := Case{Expect: tc.e}.ExpectedIntervene()
		if w != tc.want || k != tc.known {
			t.Errorf("%s: got (%v,%v) want (%v,%v)", name, w, k, tc.want, tc.known)
		}
	}
}
