# mermaid

A self-contained terminal renderer for [Mermaid](https://mermaid.js.org/)
diagrams. It parses a Mermaid source block and paints it as Unicode
box-drawing art — no browser, no network, no external dependencies (Go standard
library only).

It is a Go port of the Rust renderer in
[`xai-org/grok-build`](https://github.com/xai-org/grok-build/blob/b189869b7755d2b482969acf6c92da3ecfeffd36/crates/codegen/xai-grok-markdown/src/mermaid.rs).

## Supported diagram types

| Type | Header | Notes |
| --- | --- | --- |
| Flowchart | `graph` / `flowchart` | `TB`/`TD`/`BT`/`LR`/`RL`, node shapes, edge heads/styles, labels, `subgraph` grouping (nested) |
| State | `stateDiagram` / `stateDiagram-v2` | `[*]` start/end, transitions, `<<choice>>`, descriptions |
| Class | `classDiagram` | compartments (attrs/methods), annotations, relations, cardinalities, generics |
| ER | `erDiagram` | entities, attributes, crow's-foot cardinalities |
| Sequence | `sequenceDiagram` | messages, self-calls, notes, dividers (`loop`/`alt`/…), `autonumber` |

Anything else — or a diagram too wide for the terminal — falls back to the raw
source in a framed box.

## Install / build

```sh
go build -o mermaid .
```

## Usage

```
usage: mermaid [-w cols] [-color auto|always|never] [-html] [-svg] [-title t] [-raw] [file ...]
```

- Reads every ` ```mermaid ` (or `~~~mermaid`) fenced block from each file, or
  from stdin when no files are given.
- With `-raw`, the whole input is treated as a single diagram (no fence
  scanning).
- `-w` sets the max render width; `0` auto-detects the terminal width (falling
  back to 100 columns, or `$COLUMNS`).
- `-color` controls ANSI colorization; `auto` colorizes only when stdout is a
  terminal.
- `-html` emits a single self-contained, theme-aware HTML page (all diagrams as
  colored `<pre>` blocks, no external assets) instead of terminal text; `-title`
  sets its page title.
- `-svg` renders diagrams as inline SVG: on its own it writes a self-contained
  SVG document (diagrams stacked vertically); combined with `-html` it embeds
  each diagram as an inline `<svg>` in the page. Text runs are pinned to the
  monospace grid with `textLength`, so box-drawing stays aligned in any font.
- Width defaults to unlimited under `-html`/`-svg` (the page scrolls, the SVG
  scales) unless `-w` is given.

### Examples

Render a diagram piped on stdin:

```sh
printf 'flowchart TD\n  A[Start] --> B{OK?}\n  B -->|yes| C[Ship]\n  B -->|no| A\n' | mermaid
```

```
 ┌───────┐ no
 │ Start │◄───┐
 └───┬───┘    │
     │        │
     ▼        │
  ╭─────╮     │
  │ OK? ├─────┘
  ╰──┬──╯
     │
     ▼yes
 ┌──────┐
 │ Ship │
 └──────┘
```

Render all Mermaid blocks embedded in a Markdown file:

```sh
mermaid README.md
```

Turn a document's diagrams into a standalone web page:

```sh
mermaid -html -title "Architecture" docs/design.md > diagrams.html
```

Export a diagram as a scalable SVG (or an HTML page with inline SVG):

```sh
mermaid -svg docs/design.md > diagrams.svg
mermaid -html -svg -title "Architecture" docs/design.md > diagrams.html
```

## Testing

```sh
go test ./...
```

Tests are ~85% statement coverage and come in three layers:

- **Golden files** (`golden_test.go`, `testdata/*.golden`) capture the exact
  rendered art for a diagram of every type plus the routing, direction-flip
  (`BT`/`RL`), edge-style (dotted/thick), edge-head, self-loop, back-edge,
  subgraph, and crossing-minimization paths — so any layout regression is
  caught. Regenerate them after an intentional change with:

  ```sh
  go test -run TestGolden -update
  ```

- **Unit tests** (`helpers_test.go`) cover label cleaning, HTML-entity decoding,
  markdown stripping, wrapping/truncation, display-width, and the ANSI color
  layer.

- **CLI tests** (`cli_test.go`) drive the `run` entry point end-to-end: stdin
  and file inputs, fence extraction, `-raw`, `-color`, multi-diagram output, and
  error paths.

The only uncovered code is `main` (a one-line call into `run`) and the raw
`ioctl` terminal-width probe, which needs a real TTY.
