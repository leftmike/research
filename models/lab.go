package main

import "strings"

// inferLab determines the originating lab (the organization that created a
// model) from its family, id, and serving provider. models.dev and litellm
// describe the serving provider but not the creator, so the lab is inferred
// from a curated mapping with a first-party-provider fallback.
func inferLab(family, id, provider string) string {
	if lab := labFromFamily(family); lab != "" {
		return lab
	}
	hay := strings.ToLower(id)
	if lab := labFromKeywords(hay); lab != "" {
		return lab
	}
	if lab := firstPartyProviderLab[strings.ToLower(provider)]; lab != "" {
		return lab
	}
	return "Unknown"
}

// labFromFamily maps a models.dev "family" value to its lab. Families are model
// lineages (e.g. "claude-opus", "gemini-flash"), so the prefix is matched.
func labFromFamily(family string) string {
	f := strings.ToLower(strings.TrimSpace(family))
	if f == "" {
		return ""
	}
	for _, fp := range familyPrefixes {
		if f == fp.prefix || strings.HasPrefix(f, fp.prefix+"-") {
			return fp.lab
		}
	}
	return ""
}

// labFromKeywords matches well-known substrings in a model id as a fallback for
// records without a family (notably litellm entries).
func labFromKeywords(id string) string {
	for _, kw := range idKeywords {
		if strings.Contains(id, kw.prefix) {
			return kw.lab
		}
	}
	return ""
}

type prefixLab struct {
	prefix string
	lab    string
}

// familyPrefixes is ordered longest-prefix-first so specific families win.
var familyPrefixes = []prefixLab{
	{"claude", "Anthropic"},
	{"gpt-oss", "OpenAI"},
	{"gpt", "OpenAI"},
	{"o-mini", "OpenAI"},
	{"o-pro", "OpenAI"},
	{"o", "OpenAI"},
	{"text-embedding", "OpenAI"},
	{"whisper", "OpenAI"},
	{"dall-e", "OpenAI"},
	{"gemini", "Google"},
	{"gemma", "Google"},
	{"veo", "Google"},
	{"imagen", "Google"},
	{"llama", "Meta"},
	{"qwen", "Alibaba"},
	{"qwq", "Alibaba"},
	{"glm", "Zhipu AI"},
	{"kimi", "Moonshot AI"},
	{"minimax", "MiniMax"},
	{"deepseek", "DeepSeek"},
	{"grok", "xAI"},
	{"nemotron", "NVIDIA"},
	{"mimo", "Xiaomi"},
	{"phi", "Microsoft"},
	{"mistral", "Mistral"},
	{"ministral", "Mistral"},
	{"devstral", "Mistral"},
	{"codestral", "Mistral"},
	{"magistral", "Mistral"},
	{"pixtral", "Mistral"},
	{"command", "Cohere"},
	{"seed", "ByteDance"},
	{"doubao", "ByteDance"},
	{"ling", "InclusionAI"},
	{"flux", "Black Forest Labs"},
	{"voyage", "Voyage AI"},
	{"nova", "Amazon"},
	{"titan", "Amazon"},
	{"ernie", "Baidu"},
	{"sonar", "Perplexity"},
	{"bge", "BAAI"},
	{"hunyuan", "Tencent"},
	{"recraft", "Recraft"},
	{"jamba", "AI21"},
	{"jais", "Inception"},
	{"granite", "IBM"},
	{"stable", "Stability AI"},
	{"sdxl", "Stability AI"},
}

// idKeywords backs labFromKeywords; tokens are matched as substrings of the id.
var idKeywords = []prefixLab{
	{"claude", "Anthropic"},
	{"gpt", "OpenAI"},
	{"o1-", "OpenAI"},
	{"o3-", "OpenAI"},
	{"o4-", "OpenAI"},
	{"chatgpt", "OpenAI"},
	{"dall-e", "OpenAI"},
	{"whisper", "OpenAI"},
	{"text-embedding-ada", "OpenAI"},
	{"davinci", "OpenAI"},
	{"babbage", "OpenAI"},
	{"gemini", "Google"},
	{"gemma", "Google"},
	{"palm", "Google"},
	{"imagen", "Google"},
	{"veo", "Google"},
	{"llama", "Meta"},
	{"qwen", "Alibaba"},
	{"qwq", "Alibaba"},
	{"glm", "Zhipu AI"},
	{"kimi", "Moonshot AI"},
	{"moonshot", "Moonshot AI"},
	{"minimax", "MiniMax"},
	{"abab", "MiniMax"},
	{"deepseek", "DeepSeek"},
	{"grok", "xAI"},
	{"nemotron", "NVIDIA"},
	{"phi-", "Microsoft"},
	{"mistral", "Mistral"},
	{"mixtral", "Mistral"},
	{"ministral", "Mistral"},
	{"codestral", "Mistral"},
	{"devstral", "Mistral"},
	{"pixtral", "Mistral"},
	{"command", "Cohere"},
	{"cohere", "Cohere"},
	{"nova-", "Amazon"},
	{"titan", "Amazon"},
	{"ernie", "Baidu"},
	{"sonar", "Perplexity"},
	{"hunyuan", "Tencent"},
	{"jamba", "AI21"},
	{"j2-", "AI21"},
	{"granite", "IBM"},
	{"flux", "Black Forest Labs"},
	{"voyage", "Voyage AI"},
	{"stable-diffusion", "Stability AI"},
	{"stability", "Stability AI"},
}

// firstPartyProviderLab maps providers that are also the creator of the models
// they serve. Used only when family and id keyword matching find nothing.
var firstPartyProviderLab = map[string]string{
	"anthropic":         "Anthropic",
	"openai":            "OpenAI",
	"azure":             "OpenAI",
	"google":            "Google",
	"google-vertex":     "Google",
	"gemini":            "Google",
	"vertex_ai":         "Google",
	"deepseek":          "DeepSeek",
	"mistral":           "Mistral",
	"codestral":         "Mistral",
	"xai":               "xAI",
	"cohere":            "Cohere",
	"cohere_chat":       "Cohere",
	"moonshotai":        "Moonshot AI",
	"zhipuai":           "Zhipu AI",
	"perplexity":        "Perplexity",
	"ai21":              "AI21",
	"alibaba":           "Alibaba",
	"alibaba-cn":        "Alibaba",
	"dashscope":         "Alibaba",
	"voyage":            "Voyage AI",
	"black_forest_labs": "Black Forest Labs",
}
