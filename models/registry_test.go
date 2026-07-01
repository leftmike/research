package main

import "testing"

func TestNormalizeID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"claude-opus-4-5", "claude-opus-4-5"},
		{"anthropic.claude-opus-4-1-20250805-v1:0", "claude-opus-4-1"},
		{"bedrock/anthropic.claude-3-5-sonnet-20240620-v1:0", "claude-3-5-sonnet"},
		{"us.anthropic.claude-3-haiku-20240307-v1:0", "claude-3-haiku"},
		{"gpt-4o", "gpt-4o"},
		{"gpt-4o-2024-08-06", "gpt-4o"},
		{"gemini-1.5-pro-latest", "gemini-1.5-pro"},
		{"meta.llama3-70b-instruct-v1:0", "llama3-70b-instruct"},
	}
	for _, c := range cases {
		if got := normalizeID(c.in); got != c.want {
			t.Errorf("normalizeID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInferLab(t *testing.T) {
	cases := []struct {
		family, id, provider, want string
	}{
		{"claude-opus", "claude-opus-4-5", "anthropic", "Anthropic"},
		{"gpt", "gpt-4o", "openai", "OpenAI"},
		{"gemini-flash", "gemini-2.0-flash", "google", "Google"},
		{"llama", "llama-3.1-70b", "together", "Meta"},
		{"", "claude-3-5-sonnet-20240620", "bedrock", "Anthropic"},
		{"", "gpt-4-turbo", "azure", "OpenAI"},
		{"", "mystery-model", "openai", "OpenAI"}, // provider fallback
		{"", "mystery-model", "togetherai", "Unknown"},
	}
	for _, c := range cases {
		if got := inferLab(c.family, c.id, c.provider); got != c.want {
			t.Errorf("inferLab(%q,%q,%q) = %q, want %q", c.family, c.id, c.provider, got, c.want)
		}
	}
}

func TestParseSources(t *testing.T) {
	cases := []struct {
		in             string
		md, ll, wantOK bool
	}{
		{"all", true, true, true},
		{"", true, true, true},
		{"models.dev", true, false, true},
		{"MD", true, false, true},
		{"litellm", false, true, true},
		{"ll", false, true, true},
		{"bogus", false, false, false},
	}
	for _, c := range cases {
		md, ll, err := parseSources(c.in)
		if (err == nil) != c.wantOK {
			t.Errorf("parseSources(%q) ok = %v, want %v (err=%v)", c.in, err == nil, c.wantOK, err)
			continue
		}
		if err == nil && (md != c.md || ll != c.ll) {
			t.Errorf("parseSources(%q) = (%v,%v), want (%v,%v)", c.in, md, ll, c.md, c.ll)
		}
	}
}

func TestRegistryMergeAndGroups(t *testing.T) {
	r := newRegistry()
	r.upsertProvider("anthropic", "Anthropic", "https://docs", "@ai-sdk/anthropic", []string{"ANTHROPIC_API_KEY"}, sourceModelsDev)
	r.addModel(&Model{
		ID: "claude-opus-4-5", Name: "Claude Opus 4.5", Family: "claude-opus",
		Provider: "anthropic", Source: sourceModelsDev, Context: 200000,
		Cost: Cost{Input: 5, Output: 25},
	})
	// Same logical model from litellm via a different provider key.
	r.addModel(&Model{
		ID: "anthropic.claude-opus-4-5-20251101-v1:0", Name: "anthropic.claude-opus-4-5-20251101-v1:0",
		Provider: "bedrock_converse", Source: sourceLiteLLM, Context: 200000,
		Cost: Cost{Input: 5, Output: 25},
	})

	groups := r.groups()
	if len(groups) != 1 {
		t.Fatalf("expected 1 merged group, got %d", len(groups))
	}
	g := groups[0]
	if g.Rep.Source != sourceModelsDev {
		t.Errorf("representative should be models.dev record, got %q", g.Rep.Source)
	}
	if len(g.Providers) != 2 {
		t.Errorf("expected 2 providers, got %v", g.Providers)
	}
	if len(g.Sources) != 2 {
		t.Errorf("expected 2 sources, got %v", g.Sources)
	}
	if g.Rep.Lab != "Anthropic" {
		t.Errorf("expected lab Anthropic, got %q", g.Rep.Lab)
	}
}
