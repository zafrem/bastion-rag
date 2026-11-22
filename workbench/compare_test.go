package main

import (
	"math"
	"strings"
	"testing"
)

func TestUserMessage_ExactFormat(t *testing.T) {
	if got := userMessage("Q?", nil); got != "Q?" {
		t.Errorf("no documents: the question must be sent untouched, got %q", got)
	}
	want := "Context:\nA\n\nB\n\nQuestion: Q?"
	if got := userMessage("Q?", []string{"A", "B"}); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestMeasureTransform_NoDocumentsIsUnchanged(t *testing.T) {
	m := MeasureTransform(Case{ID: "x", Prompt: "What is AI?"})
	if !m.Unchanged || m.Expansion != 1 || m.Strain != 0 || m.ShareQuestion != 1 || m.ShareContext != 0 {
		t.Errorf("unexpected: %+v", m)
	}
}

func TestMeasureTransform_SharesAddUpAndQuestionSurvives(t *testing.T) {
	c := Case{ID: "x", Prompt: "When is the office open?", Context: []string{"The office is open from nine to five."}}
	m := MeasureTransform(c)
	if m.Unchanged || m.Expansion <= 1 || math.Abs(m.Strain-(m.Expansion-1)) > 1e-9 {
		t.Fatalf("documents must enlarge the message: %+v", m)
	}
	if sum := m.ShareQuestion + m.ShareContext + m.ShareTemplate; math.Abs(sum-1) > 1e-9 {
		t.Errorf("shares add to %v, want 1", sum)
	}
	if m.ShareTemplate <= 0 {
		t.Errorf("the Context/Question labels are template text, share should be positive: %+v", m)
	}
	if !strings.HasSuffix(userMessage(c.Prompt, c.Context), c.Prompt) {
		t.Error("the question must be appended verbatim")
	}
	if m.QuestionCovered <= 0 || m.QuestionCovered > 1 {
		t.Errorf("'office' and 'open' occur in the document, coverage should be in (0,1]: %v", m.QuestionCovered)
	}
}

func TestMeasureTransform_CountsCharactersNotBytes(t *testing.T) {
	m := MeasureTransform(Case{ID: "k", Prompt: "영업시간이 어떻게 되나요?"})
	if m.RawChars != 14 {
		t.Errorf("Korean text should count by character, got %d", m.RawChars)
	}
}

func TestSummarize(t *testing.T) {
	s := Summarize([]Transform{
		{Unchanged: true, RawChars: 10, AugmentedChars: 10},
		{Strain: 1, ShareContext: 0.4, QuestionCovered: 0.5, RawChars: 10, AugmentedChars: 20},
		{Strain: 3, ShareContext: 0.6, QuestionCovered: 1, RawChars: 10, AugmentedChars: 40},
	})
	if s.Cases != 3 || s.Unchanged != 1 || s.WithContext != 2 || s.MaxStrain != 3 || s.MedianStrain != 3 ||
		math.Abs(s.MeanShareCtx-0.5) > 1e-9 || math.Abs(s.MeanCovered-0.75) > 1e-9 {
		t.Errorf("%+v", s)
	}
}
