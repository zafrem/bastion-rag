package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runScore implements `workbench score`: run a dataset against the live
// services, print the report, and optionally fail when a quality bar is missed.
func runScore(cfg Config, profiles []Profile, args []string) int {
	fs := flag.NewFlagSet("score", flag.ExitOnError)
	data := fs.String("data", "", "dataset file: JSONL, CSV, or one prompt per line (default: the built-in dataset)")
	profs := fs.String("profiles", "", "comma-separated profiles for cases that do not name their own (default: the first profile)")
	cats := fs.String("categories", "", "comma-separated categories to run (default: all)")
	conc := fs.Int("concurrency", 4, "runs in parallel")
	repeat := fs.Int("repeat", 1, "run the whole set this many times (load testing)")
	llm := fs.Bool("llm", false, "let the live model answer cases that have no model_answer")
	out := fs.String("out", "", "directory to write report.json and report.md")
	gate := fs.Bool("gate", false, "exit 1 if the quality bar is missed")
	g := DefaultGate()
	fs.Float64Var(&g.MinPassRate, "min-pass", g.MinPassRate, "gate: minimum pass rate")
	fs.Float64Var(&g.MinRecall, "min-recall", g.MinRecall, "gate: minimum recall")
	fs.Float64Var(&g.MaxFPR, "max-fpr", g.MaxFPR, "gate: maximum false-positive rate")
	fs.IntVar(&g.MaxLeaks, "max-leaks", g.MaxLeaks, "gate: maximum leaks")
	_ = fs.Parse(args)

	cases, err := LoadDatasetFile(*data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dataset:", err)
		return 2
	}
	if probs := ValidateDataset(cases, nameSet(profiles)); len(probs) > 0 {
		fmt.Fprintln(os.Stderr, "invalid dataset:\n  "+strings.Join(probs, "\n  "))
		return 2
	}
	all := profileMap(profiles)
	chosen := []string{profiles[0].Name}
	if *profs != "" {
		chosen = strings.Split(*profs, ",")
	}
	use := map[string]Profile{}
	for _, n := range chosen {
		p, ok := all[strings.TrimSpace(n)]
		if !ok {
			fmt.Fprintln(os.Stderr, "unknown profile", n)
			return 2
		}
		use[p.Name] = p
	}
	for i := range cases {
		if len(cases[i].Profiles) == 0 {
			cases[i].Profiles = chosen
		}
		for _, pn := range cases[i].Profiles {
			if p, ok := all[pn]; ok {
				use[pn] = p
			}
		}
	}
	if !*llm {
		cfg.OllamaURL = ""
	}
	opt := ScoreOptions{Profiles: use, Concurrency: *conc, Repeat: *repeat}
	if *cats != "" {
		opt.Categories = strings.Split(*cats, ",")
	}
	name := "built-in dataset"
	if *data != "" {
		name = filepath.Base(*data)
	}
	fmt.Fprintf(os.Stderr, "running %d runs against %s ...\n", CountRuns(cases, opt), cfg.SentinelURL)
	rep := Score(context.Background(), Pipeline{cfg}, name, cases, opt)
	md := rep.Markdown(g)
	fmt.Println(md)

	if *out != "" {
		if err := os.MkdirAll(*out, 0o750); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		js, _ := json.MarshalIndent(rep, "", "  ")
		_ = os.WriteFile(filepath.Join(*out, "report.json"), js, 0o640)
		_ = os.WriteFile(filepath.Join(*out, "report.md"), []byte(md), 0o640)
		fmt.Fprintln(os.Stderr, "wrote", filepath.Join(*out, "report.{json,md}"))
	}
	if *gate && rep.LabeledRuns > 0 {
		if v := g.Violations(rep); len(v) > 0 {
			fmt.Fprintln(os.Stderr, "gate failed:", strings.Join(v, "; "))
			return 1
		}
	}
	return 0
}
