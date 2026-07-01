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
		r.upsertProvider(pid, p.Name, p.Doc, p.NPM, p.Env, SourceModelsDev)
		for _, m := range p.Models {
			r.addModel(&Model{
				ID:          m.ID,
				Name:        nonEmpty(m.Name, m.ID),
				Family:      m.Family,
				Provider:    pid,
				Source:      SourceModelsDev,
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

// ---- litellm schema ----

type llModel struct {
	Provider           string  `json:"litellm_provider"`
	Mode               string  `json:"mode"`
	MaxInputTokens     int     `json:"max_input_tokens"`
	MaxOutputTokens    int     `json:"max_output_tokens"`
	MaxTokens          int     `json:"max_tokens"`
	InputCostPerToken  float64 `json:"input_cost_per_token"`
	OutputCostPerToken float64 `json:"output_cost_per_token"`
	CacheReadCost      float64 `json:"cache_read_input_token_cost"`
	CacheCreationCost  float64 `json:"cache_creation_input_token_cost"`
	SupportsReasoning  bool    `json:"supports_reasoning"`
	SupportsFunctions  bool    `json:"supports_function_calling"`
	SupportsVision     bool    `json:"supports_vision"`
	SupportsPDFInput   bool    `json:"supports_pdf_input"`
}

// loadLiteLLM parses litellm's model_prices_and_context_window.json into the
// registry. The "sample_spec" documentation entry is skipped.
func loadLiteLLM(data []byte, r *Registry) error {
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse litellm: %w", err)
	}

	for id, raw := range entries {
		if id == "sample_spec" {
			continue
		}
		var m llModel
		if err := json.Unmarshal(raw, &m); err != nil {
			// Skip malformed entries rather than failing the whole load.
			continue
		}
		provider := nonEmpty(m.Provider, "unknown")
		r.upsertProvider(provider, "", "", "", nil, SourceLiteLLM)

		context := m.MaxInputTokens
		if context == 0 {
			context = m.MaxTokens
		}
		var in, out []string
		if m.SupportsVision || m.SupportsPDFInput {
			in = []string{"text", "image"}
		}
		r.addModel(&Model{
			ID:          id,
			Name:        id,
			Provider:    provider,
			Source:      SourceLiteLLM,
			Mode:        m.Mode,
			InputModes:  in,
			OutputModes: out,
			Reasoning:   m.SupportsReasoning,
			ToolCall:    m.SupportsFunctions,
			Attachment:  m.SupportsVision || m.SupportsPDFInput,
			Context:     context,
			OutputLimit: m.MaxOutputTokens,
			Cost: Cost{
				Input:      perMillion(m.InputCostPerToken),
				Output:     perMillion(m.OutputCostPerToken),
				CacheRead:  perMillion(m.CacheReadCost),
				CacheWrite: perMillion(m.CacheCreationCost),
			},
		})
	}
	return nil
}

// perMillion converts a per-token cost to a per-million-token cost so both
// sources report pricing in the same units.
func perMillion(perToken float64) float64 {
	return perToken * 1_000_000
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// BuildRegistry fetches the selected sources and merges them into a registry.
// Only the requested catalogs are downloaded.
func BuildRegistry(opts FetchOptions, useMD, useLL bool) (*Registry, error) {
	r := newRegistry()

	if useMD {
		mdData, err := fetchJSON(modelsDevURL, "modelsdev.json", opts)
		if err != nil {
			return nil, err
		}
		if err := loadModelsDev(mdData, r); err != nil {
			return nil, err
		}
	}

	if useLL {
		llData, err := fetchJSON(litellmURL, "litellm.json", opts)
		if err != nil {
			return nil, err
		}
		if err := loadLiteLLM(llData, r); err != nil {
			return nil, err
		}
	}

	return r, nil
}
