package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// cmdSummary prints an overview of the merged catalog.
func cmdSummary(r *Registry, args []string) {
	parseNoArgs("summary", args)

	groups := r.groups()

	mdProviders, llProviders := 0, 0
	for _, p := range r.Providers {
		if p.Sources[sourceModelsDev] {
			mdProviders++
		}
		if p.Sources[sourceLiteLLM] {
			llProviders++
		}
	}
	mdModels, llModels := 0, 0
	for _, m := range r.Models {
		switch m.Source {
		case sourceModelsDev:
			mdModels++
		case sourceLiteLLM:
			llModels++
		}
	}

	fmt.Println("Sources:")
	fmt.Printf("  %-12s %d providers, %d model records\n", sourceModelsDev, mdProviders, mdModels)
	fmt.Printf("  %-12s %d providers, %d model records\n", sourceLiteLLM, llProviders, llModels)
	fmt.Println()
	fmt.Println("Merged totals:")
	fmt.Printf("  Providers: %d\n", len(r.Providers))
	fmt.Printf("  Labs:      %d\n", len(r.Labs))
	fmt.Printf("  Models:    %d unique (%d records)\n", len(groups), len(r.Models))
	fmt.Println()

	fmt.Println("Top labs by model count:")
	labs := r.sortedLabs()
	for i, l := range labs {
		if i >= 12 {
			break
		}
		fmt.Printf("  %-20s %d\n", l.Name, uniqueModelCount(l.Models))
	}
}

// cmdProviders lists providers, optionally filtered by a substring.
func cmdProviders(r *Registry, args []string) {
	filter := parseFilter("providers", args)

	providers := r.sortedProviders()
	const idW, nameW, srcW = 24, 26, 18
	fmt.Printf("%-*s %-*s %-*s %s\n", idW, "ID", nameW, "NAME", srcW, "SOURCES", "MODELS")
	fmt.Printf("%-*s %-*s %-*s %s\n", idW, dashes(2), nameW, dashes(4), srcW, dashes(7), dashes(6))
	for _, p := range providers {
		if filter != "" && !matches(filter, p.ID, p.Name) {
			continue
		}
		fmt.Printf("%-*s %-*s %-*s %d\n",
			idW, truncate(p.ID, idW-1),
			nameW, truncate(p.Name, nameW-1),
			srcW, truncate(sourcesString(p.Sources), srcW-1),
			len(p.Models))
	}
}

// cmdProvider shows one provider and the models it serves.
func cmdProvider(r *Registry, args []string) {
	id := parseSingle("provider", "<id>", args)
	p := r.Providers[id]
	if p == nil {
		p = findProviderInsensitive(r, id)
	}
	if p == nil {
		fmt.Fprintf(os.Stderr, "error: no provider %q (try \"models providers\")\n", id)
		os.Exit(1)
	}

	fmt.Printf("Provider:  %s\n", p.ID)
	fmt.Printf("Name:      %s\n", p.Name)
	fmt.Printf("Sources:   %s\n", sourcesString(p.Sources))
	if p.Doc != "" {
		fmt.Printf("Docs:      %s\n", p.Doc)
	}
	if p.NPM != "" {
		fmt.Printf("NPM:       %s\n", p.NPM)
	}
	if len(p.Env) > 0 {
		fmt.Printf("Env:       %s\n", strings.Join(p.Env, ", "))
	}
	fmt.Printf("Models:    %d\n\n", len(p.Models))

	models := append([]*Model(nil), p.Models...)
	sort.Slice(models, func(i, j int) bool { return models[i].Name < models[j].Name })
	printModelTable(models)
}

// cmdLabs lists labs (model creators) and their model counts.
func cmdLabs(r *Registry, args []string) {
	filter := parseFilter("labs", args)

	labs := r.sortedLabs()
	const nameW, mW, pW = 24, 10, 10
	fmt.Printf("%-*s %-*s %-*s %s\n", nameW, "LAB", mW, "MODELS", pW, "PROVIDERS", "SOURCES")
	fmt.Printf("%-*s %-*s %-*s %s\n", nameW, dashes(3), mW, dashes(6), pW, dashes(9), dashes(7))
	for _, l := range labs {
		if filter != "" && !matches(filter, l.Name) {
			continue
		}
		provs := uniqueStrings(collect(l.Models, func(m *Model) string { return m.Provider }))
		srcs := uniqueStrings(collect(l.Models, func(m *Model) string { return m.Source }))
		fmt.Printf("%-*s %-*d %-*d %s\n",
			nameW, truncate(l.Name, nameW-1),
			mW, uniqueModelCount(l.Models),
			pW, len(provs),
			strings.Join(srcs, ","))
	}
}

// cmdLab shows one lab and the models attributed to it.
func cmdLab(r *Registry, args []string) {
	name := parseSingle("lab", "<name>", args)
	lab := r.Labs[name]
	if lab == nil {
		lab = findLabInsensitive(r, name)
	}
	if lab == nil {
		fmt.Fprintf(os.Stderr, "error: no lab %q (try \"models labs\")\n", name)
		os.Exit(1)
	}

	provs := uniqueStrings(collect(lab.Models, func(m *Model) string { return m.Provider }))
	fmt.Printf("Lab:       %s\n", lab.Name)
	fmt.Printf("Models:    %d unique (%d records)\n", uniqueModelCount(lab.Models), len(lab.Models))
	fmt.Printf("Providers: %s\n\n", strings.Join(provs, ", "))

	// Show one row per unique model created by this lab.
	models := representativeModels(lab.Models)
	sort.Slice(models, func(i, j int) bool { return models[i].Name < models[j].Name })
	printModelTable(models)
}

// cmdModels lists models, optionally filtered by a substring.
func cmdModels(r *Registry, args []string) {
	filter := parseFilter("models", args)

	groups := r.groups()
	const nameW, labW, ctxW, costW = 38, 18, 9, 16
	fmt.Printf("%-*s %-*s %-*s %-*s %s\n", nameW, "NAME", labW, "LAB", ctxW, "CONTEXT", costW, "IN/OUT $/M", "PROVIDERS")
	fmt.Printf("%-*s %-*s %-*s %-*s %s\n", nameW, dashes(4), labW, dashes(3), ctxW, dashes(7), costW, dashes(9), dashes(9))
	count := 0
	for _, g := range groups {
		m := g.Rep
		if filter != "" && !matches(filter, m.Name, m.ID, m.Lab) {
			continue
		}
		fmt.Printf("%-*s %-*s %-*s %-*s %d\n",
			nameW, truncate(m.Name, nameW-1),
			labW, truncate(m.Lab, labW-1),
			ctxW, formatContext(m.Context),
			costW, formatCostPair(m.Cost),
			len(g.Providers))
		count++
	}
	if count == 0 {
		fmt.Println("(no models matched)")
	}
}

// cmdModel shows full detail for one model, merging all matching records.
func cmdModel(r *Registry, args []string) {
	query := parseSingle("model", "<id>", args)

	g := findGroup(r, query)
	if g == nil {
		fmt.Fprintf(os.Stderr, "error: no model matching %q (try \"models models %s\")\n", query, query)
		os.Exit(1)
	}

	m := g.Rep
	fmt.Printf("Name:        %s\n", m.Name)
	fmt.Printf("ID:          %s\n", m.ID)
	if m.Family != "" {
		fmt.Printf("Family:      %s\n", m.Family)
	}
	fmt.Printf("Lab:         %s\n", m.Lab)
	fmt.Printf("Providers:   %s\n", strings.Join(g.Providers, ", "))
	fmt.Printf("Sources:     %s\n", strings.Join(g.Sources, ", "))
	if m.ReleaseDate != "" {
		fmt.Printf("Released:    %s\n", m.ReleaseDate)
	}
	if m.Knowledge != "" {
		fmt.Printf("Knowledge:   %s\n", m.Knowledge)
	}
	fmt.Printf("Open weights:%s\n", yesNo(m.OpenWeights))

	fmt.Println()
	fmt.Println("Capabilities:")
	fmt.Printf("  Reasoning:   %s\n", yesNo(m.Reasoning))
	fmt.Printf("  Tool call:   %s\n", yesNo(m.ToolCall))
	fmt.Printf("  Attachment:  %s\n", yesNo(m.Attachment))
	if len(m.InputModes) > 0 {
		fmt.Printf("  Input:       %s\n", strings.Join(m.InputModes, ", "))
	}
	if len(m.OutputModes) > 0 {
		fmt.Printf("  Output:      %s\n", strings.Join(m.OutputModes, ", "))
	}

	fmt.Println()
	fmt.Println("Limits:")
	fmt.Printf("  Context:     %s\n", formatContext(m.Context))
	fmt.Printf("  Max output:  %s\n", formatContext(m.OutputLimit))

	fmt.Println()
	fmt.Println("Cost (USD per million tokens):")
	fmt.Printf("  Input:       %s\n", formatCost(m.Cost.Input))
	fmt.Printf("  Output:      %s\n", formatCost(m.Cost.Output))
	fmt.Printf("  Cache read:  %s\n", formatCost(m.Cost.CacheRead))
	fmt.Printf("  Cache write: %s\n", formatCost(m.Cost.CacheWrite))

	// Per-record breakdown so per-provider/source differences are visible.
	if len(g.Records) > 1 {
		fmt.Println()
		fmt.Println("Records:")
		recs := append([]*Model(nil), g.Records...)
		sort.Slice(recs, func(i, j int) bool {
			if recs[i].Source != recs[j].Source {
				return recs[i].Source < recs[j].Source
			}
			return recs[i].Provider < recs[j].Provider
		})
		const srcW, provW, ctxW = 12, 22, 9
		fmt.Printf("  %-*s %-*s %-*s %-*s %s\n", srcW, "SOURCE", provW, "PROVIDER", ctxW, "CONTEXT", 16, "IN/OUT $/M", "ID")
		for _, rec := range recs {
			fmt.Printf("  %-*s %-*s %-*s %-*s %s\n",
				srcW, rec.Source,
				provW, truncate(rec.Provider, provW-1),
				ctxW, formatContext(rec.Context),
				16, formatCostPair(rec.Cost),
				rec.ID)
		}
	}
}

// ---- shared printing helpers ----

func printModelTable(models []*Model) {
	const nameW, ctxW, costW = 40, 9, 16
	fmt.Printf("%-*s %-*s %-*s %s\n", nameW, "NAME", ctxW, "CONTEXT", costW, "IN/OUT $/M", "CAPS")
	fmt.Printf("%-*s %-*s %-*s %s\n", nameW, dashes(4), ctxW, dashes(7), costW, dashes(9), dashes(4))
	for _, m := range models {
		fmt.Printf("%-*s %-*s %-*s %s\n",
			nameW, truncate(m.Name, nameW-1),
			ctxW, formatContext(m.Context),
			costW, formatCostPair(m.Cost),
			capsString(m))
	}
}

func capsString(m *Model) string {
	var caps []string
	if m.Reasoning {
		caps = append(caps, "reason")
	}
	if m.ToolCall {
		caps = append(caps, "tools")
	}
	if m.Attachment {
		caps = append(caps, "vision")
	}
	if m.OpenWeights {
		caps = append(caps, "open")
	}
	return strings.Join(caps, ",")
}

func formatContext(n int) string {
	switch {
	case n == 0:
		return "-"
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%dK", n/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func formatCost(c float64) string {
	if c == 0 {
		return "-"
	}
	return fmt.Sprintf("$%g", c)
}

func formatCostPair(c Cost) string {
	if c.Input == 0 && c.Output == 0 {
		return "-"
	}
	return fmt.Sprintf("%g / %g", c.Input, c.Output)
}

func sourcesString(s map[string]bool) string {
	var out []string
	for _, name := range []string{sourceModelsDev, sourceLiteLLM} {
		if s[name] {
			out = append(out, name)
		}
	}
	return strings.Join(out, ",")
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func dashes(n int) string { return strings.Repeat("-", n) }

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

func matches(filter string, fields ...string) bool {
	f := strings.ToLower(filter)
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), f) {
			return true
		}
	}
	return false
}

// uniqueModelCount counts distinct models (by normalized key) in a slice.
func uniqueModelCount(ms []*Model) int {
	seen := map[string]bool{}
	for _, m := range ms {
		seen[m.Key] = true
	}
	return len(seen)
}

// representativeModels returns one record per unique key, preferring models.dev.
func representativeModels(ms []*Model) []*Model {
	byKey := map[string]*Model{}
	order := []string{}
	for _, m := range ms {
		cur, ok := byKey[m.Key]
		if !ok {
			byKey[m.Key] = m
			order = append(order, m.Key)
			continue
		}
		if m.Source == sourceModelsDev && cur.Source != sourceModelsDev {
			byKey[m.Key] = m
		}
	}
	out := make([]*Model, 0, len(order))
	for _, k := range order {
		out = append(out, byKey[k])
	}
	return out
}

// findGroup locates a model group by exact id, then exact key, then substring.
func findGroup(r *Registry, query string) *ModelGroup {
	groups := r.groups()
	key := normalizeID(query)
	q := strings.ToLower(query)

	var substr *ModelGroup
	for _, g := range groups {
		for _, rec := range g.Records {
			if rec.ID == query {
				return g
			}
		}
		if g.Key == key {
			return g
		}
		if substr == nil && (strings.Contains(strings.ToLower(g.Rep.Name), q) || strings.Contains(g.Key, q)) {
			substr = g
		}
	}
	return substr
}

func findProviderInsensitive(r *Registry, id string) *Provider {
	for pid, p := range r.Providers {
		if strings.EqualFold(pid, id) {
			return p
		}
	}
	return nil
}

func findLabInsensitive(r *Registry, name string) *Lab {
	for ln, l := range r.Labs {
		if strings.EqualFold(ln, name) {
			return l
		}
	}
	return nil
}

// ---- argument parsing ----

func parseNoArgs(name string, args []string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Parse(args)
}

func parseFilter(name string, args []string) string {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Parse(args)
	return strings.Join(fs.Args(), " ")
}

func parseSingle(name, arg string, args []string) string {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintf(os.Stderr, "usage: models %s %s\n", name, arg) }
	fs.Parse(args)
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	return strings.Join(fs.Args(), " ")
}
