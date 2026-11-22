package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Transform describes how much the text the model receives differs from what the user typed, for one case.
// It is computed locally from the same function that builds the model's input, so it cannot drift from
// what is actually sent.
type Transform struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Docs     int    `json:"docs"`

	RawChars       int `json:"raw_chars"`
	RawWords       int `json:"raw_words"`
	AugmentedChars int `json:"augmented_chars"`
	AugmentedWords int `json:"augmented_words"`

	// Expansion is augmented size divided by raw size (1.0 means unchanged).
	Expansion float64 `json:"expansion"`
	// Shares of the augmented message, by characters. They add up to 1.
	ShareQuestion float64 `json:"share_question"`
	ShareContext  float64 `json:"share_context"`
	ShareTemplate float64 `json:"share_template"`
	// QuestionCovered is the fraction of the question's content words that also occur in the documents:
	// a rough sign that the context is about what was asked.
	QuestionCovered float64 `json:"question_covered"`
	// Unchanged is true when no documents were added, so the model sees exactly what was typed.
	Unchanged bool `json:"unchanged"`
}

var stop = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`the a an and or but if of to in on at by for with from as is are was were be been it its this that
		these those i you he she we they me my your our their what which who how why when where do does did can could
		should would will not no yes about into than then so there here have has had please tell give show`) {
		stop[w] = true
	}
}

func contentWords(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if utf8.RuneCountInString(w) >= 3 && !stop[w] {
			out[w] = true
		}
	}
	return out
}

func wordCount(s string) int { return len(strings.Fields(s)) }

// MeasureTransform compares the raw prompt with the message actually sent to the model.
func MeasureTransform(c Case) Transform {
	aug := userMessage(c.Prompt, c.Context)
	t := Transform{ID: c.ID, Category: c.Category, Docs: len(c.Context),
		RawChars: utf8.RuneCountInString(c.Prompt), RawWords: wordCount(c.Prompt),
		AugmentedChars: utf8.RuneCountInString(aug), AugmentedWords: wordCount(aug),
		Unchanged: aug == c.Prompt}
	if t.RawChars > 0 {
		t.Expansion = float64(t.AugmentedChars) / float64(t.RawChars)
	}
	if t.AugmentedChars > 0 {
		ctx := 0
		for _, d := range c.Context {
			ctx += utf8.RuneCountInString(d)
		}
		t.ShareQuestion = float64(t.RawChars) / float64(t.AugmentedChars)
		t.ShareContext = float64(ctx) / float64(t.AugmentedChars)
		t.ShareTemplate = 1 - t.ShareQuestion - t.ShareContext
	}
	if len(c.Context) > 0 {
		q, d := contentWords(c.Prompt), contentWords(strings.Join(c.Context, " "))
		if len(q) > 0 {
			hit := 0
			for w := range q {
				if d[w] {
					hit++
				}
			}
			t.QuestionCovered = float64(hit) / float64(len(q))
		}
	}
	return t
}

// TransformSummary aggregates a set of measurements.
type TransformSummary struct {
	Cases           int     `json:"cases"`
	WithContext     int     `json:"with_context"`
	Unchanged       int     `json:"unchanged"`
	MedianExpansion float64 `json:"median_expansion"` // among cases with context
	MaxExpansion    float64 `json:"max_expansion"`
	MeanShareCtx    float64 `json:"mean_share_context"`
	MeanCovered     float64 `json:"mean_question_covered"`
	RawChars        int     `json:"raw_chars_total"`
	AugmentedChars  int     `json:"augmented_chars_total"`
}

func Summarize(ts []Transform) TransformSummary {
	s := TransformSummary{Cases: len(ts)}
	var exp []float64
	for _, t := range ts {
		s.RawChars += t.RawChars
		s.AugmentedChars += t.AugmentedChars
		if t.Unchanged {
			s.Unchanged++
			continue
		}
		s.WithContext++
		exp = append(exp, t.Expansion)
		s.MeanShareCtx += t.ShareContext
		s.MeanCovered += t.QuestionCovered
		if t.Expansion > s.MaxExpansion {
			s.MaxExpansion = t.Expansion
		}
	}
	if n := len(exp); n > 0 {
		sort.Float64s(exp)
		s.MedianExpansion = exp[n/2]
		s.MeanShareCtx /= float64(n)
		s.MeanCovered /= float64(n)
	}
	return s
}

func compareMarkdown(name string, ts []Transform, sum TransformSummary, examples int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Raw input vs model input: %s\n\n", name)
	b.WriteString("What the user typed compared with the message the model actually receives once documents are added. ")
	b.WriteString("The message is built by the same function the pipeline uses, so these numbers describe real traffic shape.\n\n")
	b.WriteString("| Measure | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| Cases | %d |\n| Sent to the model unchanged (no documents) | %d |\n| Augmented with documents | %d |\n", sum.Cases, sum.Unchanged, sum.WithContext)
	if sum.WithContext > 0 {
		fmt.Fprintf(&b, "| Median growth of an augmented message | **%.1fx** |\n| Largest growth | %.1fx |\n", sum.MedianExpansion, sum.MaxExpansion)
		fmt.Fprintf(&b, "| Share of an augmented message that is retrieved text (mean) | **%s** |\n", pct(sum.MeanShareCtx))
		fmt.Fprintf(&b, "| Question words that also appear in the documents (mean) | %s |\n", pct(sum.MeanCovered))
	}
	fmt.Fprintf(&b, "| Characters typed / sent in total | %d / %d |\n", sum.RawChars, sum.AugmentedChars)
	b.WriteString("\nThe question itself is never rewritten: it is appended verbatim after the documents, so the change is addition, not transformation.\n")

	var aug []Transform
	for _, t := range ts {
		if !t.Unchanged {
			aug = append(aug, t)
		}
	}
	sort.Slice(aug, func(i, j int) bool { return aug[i].Expansion > aug[j].Expansion })
	if len(aug) > 0 {
		b.WriteString("\n## Augmented cases, most expanded first\n\n| Case | Docs | Typed | Sent | Growth | Retrieved text | Question covered |\n|---|---|---|---|---|---|---|\n")
		for i, t := range aug {
			if examples > 0 && i >= examples+10 {
				break
			}
			fmt.Fprintf(&b, "| %s | %d | %d | %d | %.1fx | %s | %s |\n", t.ID, t.Docs, t.RawChars, t.AugmentedChars, t.Expansion, pct(t.ShareContext), pct(t.QuestionCovered))
		}
	}
	return b.String()
}

// runCompare implements `workbench compare`. It needs no running services.
func runCompare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	data := fs.String("data", "", "dataset file (default: the built-in dataset)")
	out := fs.String("out", "", "directory to write compare.json and compare.md")
	show := fs.String("show", "", "case id: print its raw prompt and the message the model receives, side by side")
	_ = fs.Parse(args)

	cases, err := LoadDatasetFile(*data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dataset:", err)
		return 2
	}
	if *show != "" {
		for _, c := range cases {
			if c.ID == *show {
				fmt.Printf("── RAW (what the user typed) ──\n%s\n\n── MODEL INPUT (what the model receives) ──\n%s\n", c.Prompt, userMessage(c.Prompt, c.Context))
				return 0
			}
		}
		fmt.Fprintln(os.Stderr, "no case with id", *show)
		return 2
	}
	ts := make([]Transform, 0, len(cases))
	for _, c := range cases {
		ts = append(ts, MeasureTransform(c))
	}
	sum := Summarize(ts)
	name := "built-in dataset"
	if *data != "" {
		name = filepath.Base(*data)
	}
	md := compareMarkdown(name, ts, sum, 5)
	fmt.Println(md)
	if *out != "" {
		if err := os.MkdirAll(*out, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		js, _ := json.MarshalIndent(map[string]any{"dataset": name, "summary": sum, "cases": ts}, "", "  ")
		if err := os.WriteFile(filepath.Join(*out, "compare.json"), js, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if err := os.WriteFile(filepath.Join(*out, "compare.md"), []byte(md), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	return 0
}
