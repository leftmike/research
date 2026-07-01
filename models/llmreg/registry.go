package llmreg

import (
	"sort"
	"strings"
)

// Source identifiers.
const (
	SourceModelsDev = "models.dev"
	SourceLiteLLM   = "litellm"
)

// Cost holds token pricing in US dollars per million tokens.
type Cost struct {
	Input      float64
	Output     float64
	CacheRead  float64
	CacheWrite float64
}

func (c Cost) empty() bool {
	return c.Input == 0 && c.Output == 0 && c.CacheRead == 0 && c.CacheWrite == 0
}

// Model is a single model record from one source as served by one provider.
type Model struct {
	ID          string // identifier as it appears in the source
	Key         string // normalized identifier for cross-source matching
	Name        string
	Family      string
	Lab         string
	Provider    string // provider id this record belongs to
	Source      string // SourceModelsDev or SourceLiteLLM
	Mode        string // litellm "mode" (chat, embedding, ...); empty for models.dev
	InputModes  []string
	OutputModes []string
	Reasoning   bool
	ToolCall    bool
	Attachment  bool
	OpenWeights bool
	Knowledge   string
	ReleaseDate string
	LastUpdated string
	Context     int
	OutputLimit int
	Cost        Cost
}

// Provider is an API provider that serves models.
type Provider struct {
	ID      string
	Name    string
	Doc     string
	NPM     string
	Env     []string
	Sources map[string]bool
	Models  []*Model
}

// Lab is the organization that created a set of models.
type Lab struct {
	Name   string
	Models []*Model
}

// Registry is the merged catalog assembled from all sources.
type Registry struct {
	Providers map[string]*Provider
	Labs      map[string]*Lab
	Models    []*Model
}

func newRegistry() *Registry {
	return &Registry{
		Providers: map[string]*Provider{},
		Labs:      map[string]*Lab{},
	}
}

// addModel records a model and indexes it under its provider and lab.
func (r *Registry) addModel(m *Model) {
	if m.Lab == "" {
		m.Lab = inferLab(m.Family, m.ID, m.Provider)
	}
	if m.Key == "" {
		m.Key = NormalizeID(m.ID)
	}
	r.Models = append(r.Models, m)

	p := r.Providers[m.Provider]
	if p == nil {
		p = &Provider{ID: m.Provider, Name: m.Provider, Sources: map[string]bool{}}
		r.Providers[m.Provider] = p
	}
	p.Sources[m.Source] = true
	p.Models = append(p.Models, m)

	lab := r.Labs[m.Lab]
	if lab == nil {
		lab = &Lab{Name: m.Lab}
		r.Labs[m.Lab] = lab
	}
	lab.Models = append(lab.Models, m)
}

// upsertProvider merges provider metadata (name, docs, env) into the registry.
func (r *Registry) upsertProvider(id, name, doc, npm string, env []string, source string) {
	p := r.Providers[id]
	if p == nil {
		p = &Provider{ID: id, Sources: map[string]bool{}}
		r.Providers[id] = p
	}
	if name != "" {
		p.Name = name
	}
	if p.Name == "" {
		p.Name = id
	}
	if doc != "" {
		p.Doc = doc
	}
	if npm != "" {
		p.NPM = npm
	}
	if len(env) > 0 {
		p.Env = env
	}
	p.Sources[source] = true
}

// SortedProviders returns providers sorted by id.
func (r *Registry) SortedProviders() []*Provider {
	out := make([]*Provider, 0, len(r.Providers))
	for _, p := range r.Providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// SortedLabs returns labs sorted by descending model count, then name.
func (r *Registry) SortedLabs() []*Lab {
	out := make([]*Lab, 0, len(r.Labs))
	for _, l := range r.Labs {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		ci, cj := UniqueModelCount(out[i].Models), UniqueModelCount(out[j].Models)
		if ci != cj {
			return ci > cj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ModelGroup collapses model records that share a normalized key into one
// representative entry, preferring a models.dev record when present, and merges
// the set of providers and sources that offer it.
type ModelGroup struct {
	Key       string
	Rep       *Model
	Records   []*Model
	Providers []string
	Sources   []string
}

// Groups returns the merged model groups, one per normalized key.
func (r *Registry) Groups() []*ModelGroup {
	byKey := map[string]*ModelGroup{}
	order := []string{}
	for _, m := range r.Models {
		g := byKey[m.Key]
		if g == nil {
			g = &ModelGroup{Key: m.Key}
			byKey[m.Key] = g
			order = append(order, m.Key)
		}
		g.Records = append(g.Records, m)
		// Prefer a richer models.dev record as the representative.
		if g.Rep == nil || (m.Source == SourceModelsDev && g.Rep.Source != SourceModelsDev) {
			g.Rep = m
		}
	}

	out := make([]*ModelGroup, 0, len(order))
	for _, k := range order {
		g := byKey[k]
		g.Providers = UniqueStrings(Collect(g.Records, func(m *Model) string { return m.Provider }))
		g.Sources = UniqueStrings(Collect(g.Records, func(m *Model) string { return m.Source }))
		// Fill representative gaps (cost/limits/flags) from sibling records.
		fillGaps(g)
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rep.Name < out[j].Rep.Name })
	return out
}

// GroupByKey returns the merged model group with the given normalized key.
func (r *Registry) GroupByKey(key string) *ModelGroup {
	for _, g := range r.Groups() {
		if g.Key == key {
			return g
		}
	}
	return nil
}

// fillGaps copies useful fields into the representative from sibling records
// when the representative is missing them, so the merged view is complete.
func fillGaps(g *ModelGroup) {
	rep := g.Rep
	for _, m := range g.Records {
		if m == rep {
			continue
		}
		if rep.Cost.empty() && !m.Cost.empty() {
			rep.Cost = m.Cost
		}
		if rep.Context == 0 {
			rep.Context = m.Context
		}
		if rep.OutputLimit == 0 {
			rep.OutputLimit = m.OutputLimit
		}
		if rep.Knowledge == "" {
			rep.Knowledge = m.Knowledge
		}
		if rep.ReleaseDate == "" {
			rep.ReleaseDate = m.ReleaseDate
		}
		rep.Reasoning = rep.Reasoning || m.Reasoning
		rep.ToolCall = rep.ToolCall || m.ToolCall
		rep.Attachment = rep.Attachment || m.Attachment
		rep.OpenWeights = rep.OpenWeights || m.OpenWeights
	}
}

// Collect maps f over ms and returns the results in order.
func Collect(ms []*Model, f func(*Model) string) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, f(m))
	}
	return out
}

// UniqueStrings returns the distinct, non-empty values of in, sorted.
func UniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// UniqueModelCount counts distinct models (by normalized key) in a slice.
func UniqueModelCount(ms []*Model) int {
	seen := map[string]bool{}
	for _, m := range ms {
		seen[m.Key] = true
	}
	return len(seen)
}

// RepresentativeModels returns one record per unique key, preferring models.dev.
func RepresentativeModels(ms []*Model) []*Model {
	byKey := map[string]*Model{}
	order := []string{}
	for _, m := range ms {
		cur, ok := byKey[m.Key]
		if !ok {
			byKey[m.Key] = m
			order = append(order, m.Key)
			continue
		}
		if m.Source == SourceModelsDev && cur.Source != SourceModelsDev {
			byKey[m.Key] = m
		}
	}
	out := make([]*Model, 0, len(order))
	for _, k := range order {
		out = append(out, byKey[k])
	}
	return out
}

// NormalizeID reduces a source-specific model id to a key that is comparable
// across sources, stripping provider prefixes and version/date suffixes.
func NormalizeID(id string) string {
	s := strings.ToLower(strings.TrimSpace(id))

	// Drop a path-style provider prefix: "bedrock/anthropic.claude" -> "anthropic.claude".
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	// Drop dotted vendor/region prefixes: "us.anthropic.claude-opus" -> "claude-opus".
	for {
		i := strings.Index(s, ".")
		if i < 0 || !knownVendorPrefix[s[:i]] {
			break
		}
		s = s[i+1:]
	}

	// Strip a trailing ":<n>" version marker (e.g. "-v1:0").
	if i := strings.LastIndex(s, ":"); i >= 0 {
		s = s[:i]
	}
	// Strip a trailing "-v<n>" version marker.
	s = trimVersionSuffix(s)
	// Strip a trailing date "-20250805" or "-2024-08-06".
	s = trimDateSuffix(s)
	// Strip a trailing "-latest" pointer.
	s = strings.TrimSuffix(s, "-latest")

	return strings.TrimRight(s, "-")
}

var knownVendorPrefix = map[string]bool{
	"anthropic": true, "amazon": true, "meta": true, "mistral": true,
	"cohere": true, "ai21": true, "us": true, "eu": true, "apac": true,
	"google": true, "deepseek": true, "qwen": true, "stability": true,
}

func trimVersionSuffix(s string) string {
	i := strings.LastIndex(s, "-v")
	if i < 0 {
		return s
	}
	rest := s[i+2:]
	if rest == "" {
		return s
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return s
		}
	}
	return s[:i]
}

func trimDateSuffix(s string) string {
	// Compact form: "-20250805".
	if i := strings.LastIndex(s, "-"); i >= 0 && isDigits(s[i+1:], 8) {
		return s[:i]
	}
	// Dashed form: "-2024-08-06".
	parts := strings.Split(s, "-")
	if n := len(parts); n >= 4 &&
		isDigits(parts[n-3], 4) && isDigits(parts[n-2], 2) && isDigits(parts[n-1], 2) {
		return strings.Join(parts[:n-3], "-")
	}
	return s
}

func isDigits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
