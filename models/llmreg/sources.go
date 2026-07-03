package llmreg

import (
	"encoding/json"
	"fmt"
)

// ---- models.dev schema ----

type mdProvider struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Doc    string             `json:"doc"`
	NPM    string             `json:"npm"`
	Env    []string           `json:"env"`
	Models map[string]mdModel `json:"models"`
}

type mdModel struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Family      string       `json:"family"`
	Attachment  bool         `json:"attachment"`
	Reasoning   bool         `json:"reasoning"`
	ToolCall    bool         `json:"tool_call"`
	Knowledge   string       `json:"knowledge"`
	ReleaseDate string       `json:"release_date"`
	LastUpdated string       `json:"last_updated"`
	OpenWeights bool         `json:"open_weights"`
	Modalities  mdModalities `json:"modalities"`
	Limit       mdLimit      `json:"limit"`
	Cost        mdCost       `json:"cost"`
}

type mdModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type mdLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

type mdCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

// loadModelsDev parses models.dev's api.json into the registry.
func loadModelsDev(data []byte, r *Registry) error {
	var providers map[string]mdProvider
	if err := json.Unmarshal(data, &providers); err != nil {
		return fmt.Errorf("parse models.dev: %w", err)
	}

	for pid, p := range providers {
		r.upsertProvider(pid, p.Name, p.Doc, p.NPM, p.Env)
		for _, m := range p.Models {
			r.addModel(&Model{
				ID:          m.ID,
				Name:        nonEmpty(m.Name, m.ID),
				Family:      m.Family,
				Provider:    pid,
				InputModes:  m.Modalities.Input,
				OutputModes: m.Modalities.Output,
				Reasoning:   m.Reasoning,
				ToolCall:    m.ToolCall,
				Attachment:  m.Attachment,
				OpenWeights: m.OpenWeights,
				Knowledge:   m.Knowledge,
				ReleaseDate: m.ReleaseDate,
				LastUpdated: m.LastUpdated,
				Context:     m.Limit.Context,
				OutputLimit: m.Limit.Output,
				Cost: Cost{
					Input:      m.Cost.Input,
					Output:     m.Cost.Output,
					CacheRead:  m.Cost.CacheRead,
					CacheWrite: m.Cost.CacheWrite,
				},
			})
		}
	}
	return nil
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// BuildRegistry fetches models.dev and loads it into a registry.
func BuildRegistry(opts FetchOptions) (*Registry, error) {
	r := newRegistry()

	mdData, err := fetchJSON(modelsDevURL, "modelsdev.json", opts)
	if err != nil {
		return nil, err
	}
	if err := loadModelsDev(mdData, r); err != nil {
		return nil, err
	}

	return r, nil
}
