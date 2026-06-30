# models

A command-line tool for exploring LLM **model**, **provider**, and **lab**
information merged from two public catalogs:

- [models.dev](https://models.dev) — `https://models.dev/api.json`
- [litellm](https://github.com/BerriAI/litellm) —
  `model_prices_and_context_window.json`

Both sources describe the *provider* (the API serving a model) but not the
*lab* (the organization that created it), so the lab is inferred from a curated
family/keyword mapping (see `lab.go`); unrecognized long-tail models are
reported as `Unknown`.

## Build

```
go build -o models .
```

## Usage

```
models [global flags] <command> [args]
```

### Commands

| Command              | Description                                              |
|----------------------|----------------------------------------------------------|
| `summary`            | Overview: source, provider, lab, and model counts.       |
| `providers [SUBSTR]` | List providers (optionally filtered by substring).       |
| `provider <id>`      | Show one provider and the models it serves.              |
| `labs [SUBSTR]`      | List labs (model creators) and their model counts.       |
| `lab <name>`         | Show one lab and its models.                             |
| `models [SUBSTR]`    | List models (optionally filtered by substring).          |
| `model <id>`         | Full detail for one model, merged across sources.        |

If the first argument is not one of the commands above, it is matched against
providers. When exactly one provider matches it is used, and an optional second
argument selects a model within it:

```
models <provider>           # show the matching provider (like "provider")
models <provider> <model>   # show a model within that provider
```

If the argument matches several providers (or the model specifier matches
several models in the provider), the candidates are listed so you can narrow
the query.

### Global flags

- `-refresh` — ignore cached data and re-download from the sources.
- `-no-cache` — do not read or write the on-disk cache.

Downloaded data is cached under the user cache directory and refreshed every
24 hours. If a download fails, a stale cache copy is used as a fallback.

### Examples

```
models summary
models providers anthropic
models provider anthropic
models labs
models lab Anthropic
models models grok-4
models model claude-opus-4-5
models openai                  # provider shortcut
models openai gpt-4o           # model within a provider
```

A model's detail view merges every matching record across both sources and all
providers, with a per-record breakdown so per-provider price and context
differences are visible.
