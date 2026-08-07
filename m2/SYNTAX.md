# m2 — a language for describing diagrams

m2 is a declarative language for describing diagrams: boxes, arrows, containers,
and the styling that goes with them. It is a redesign of
[d2](https://d2lang.com/), keeping what makes d2 pleasant — text that reads like
the picture it produces — while cutting the grammar down to something you can
hold in your head and write a parser for in an afternoon.

The whole language is **three statement types**:

| Statement | Example | Meaning |
| --- | --- | --- |
| Declaration | `db: "Postgres" [shape: cylinder]` | create or update an element |
| Edge | `api -> db: "query"` | connect elements |
| Attributes | `[direction: right]` | configure the enclosing scope |

Statements are separated by `;` or a newline — the two are interchangeable.

There are no keywords, no selectors, and no reserved attribute names, which
follows from the one structural idea worth holding on to:

> **Attributes live in `[...]`. Children live in `{...}`. They are separate
> namespaces and never collide.**

In d2, `shape`, `style`, `label`, `near`, `width` and about twenty other words
are reserved *keys* sharing a namespace with your node names. In m2 an element
may be named `style`, `shape`, or `label`, because attributes are never written
as children.

This document specifies the syntax and the evaluation semantics. It does not
specify layout or rendering beyond what the syntax must express.

- [1. A first look](#1-a-first-look)
- [2. Lexical structure](#2-lexical-structure)
- [3. Elements](#3-elements)
- [4. Attributes](#4-attributes)
- [5. Reuse](#5-reuse)
- [6. Edges](#6-edges)
- [7. Containers and paths](#7-containers-and-paths)
- [8. Structured kinds](#8-structured-kinds)
- [9. Evaluation semantics](#9-evaluation-semantics)
- [10. Attribute reference](#10-attribute-reference)
- [11. Grammar](#11-grammar)
- [12. Differences from d2](#12-differences-from-d2)
- [13. Worked example](#13-worked-example)

---

## 1. A first look

```m2
// architecture.m2
[title: "Ingest pipeline", direction: right]

edge:  "Edge Proxy" [shape: hexagon]
queue: "Kafka"      [shape: queue, fill: #FFF3BF]

workers: "Workers" {
  [direction: down]
  parse:  "Parser"
  enrich: "Enricher"
  parse -> enrich
}

store: "Postgres" [shape: cylinder]

edge -> queue: "events"
queue -> workers.parse
workers.enrich -> store: "upsert" [stroke-dash: 3]
```

Every line above is one of the three statement types. That is the entire
inventory — the rest of this document is what may appear inside them.

---

## 2. Lexical structure

### 2.1 Files

An m2 source file uses the extension `.m2`, is encoded in UTF-8, and is
newline-separated. A file is a sequence of statements evaluated in the **root
scope**, which is itself an element: the diagram.

### 2.2 Statement separators

`;` and a newline are the same token. Either ends a statement; consecutive
separators are collapsed, so blank lines and trailing `;` are harmless.

```m2
a; b; c                  // three declarations
a
b
c                        // the same three
```

A statement continues across newlines while it is inside `[]`, `{}`, `()`, or a
triple-quoted string, so lists may be broken up freely:

```m2
servers [
  shape: hexagon,
  fill: #F1F3F5,
]
```

An opening `{` must appear on the same line as the statement that introduces it.
That keeps "a newline ends a statement" true with no lookahead.

### 2.3 Comments

```m2
// line comment, runs to end of line

/* block comment,
   may span lines and nest */
```

Comments are lexical — they are stripped before parsing and are not statements.
m2 uses `//` rather than d2's `#` so that `#` is free for hex color literals.
`#4C6EF5` is a value, not a comment.

### 2.4 Names

A **bare name** is a run of characters drawn from letters, digits, and
`_ - + # ! ? @ % '`, plus internal whitespace (runs of whitespace collapse to a
single space; leading and trailing whitespace is trimmed).

A bare name ends at any of `: ; , . { } [ ] ( )`, an arrow token (`->`, `<-`,
`<->`, `--`), a comment opener, or end of line.

A bare name may not *begin* with `--` or `->`. There are no other restrictions:
a path is just names joined by `.`, with no anchors or sigils to avoid.

```m2
Order Service            // legal: internal spaces are fine
#cache                   // legal: # is only a color prefix inside a value
-conn                    // legal: a single leading - is not an arrow
"api.example.com"        // must be quoted: bare . is the path separator
```

Names are case-sensitive. Two names differing only in internal whitespace runs
are the same name.

### 2.5 Strings

```m2
"double quoted, with \" \\ \n \t escapes"
'single quoted, raw — no escapes'
```

Triple-quoted blocks carry multi-line text and may be tagged with a content
type. The common indentation prefix of the block is stripped.

```m2
readme: """md
  ## Ingest
  Events are **batched** every 5s.
"""

snippet: """go
  func main() { fmt.Println("hi") }
"""
```

An untagged `"""..."""` is plain text. Recognized tags are `md`, `tex`, and any
language name for syntax-highlighted code.

### 2.6 Values

| Form | Examples |
| --- | --- |
| Word | `cylinder`, `right`, `top-left` |
| Number | `2`, `0.5`, `-3` |
| Percentage | `50%` |
| Color | `#4C6EF5`, `#FFF3BF80`, `#eee`, `red`, `transparent` |
| String | `"Postgres"`, `'raw'`, `"""md ... """` |
| Path | `store`, `templates.store` |
| List | `(4, 8)`, `(a.b, c.d)` |

Bare words and paths lex identically; which one a value is depends on the
attribute key. `shape: cylinder` reads `cylinder` as a word, `like: base` reads
`base` as a path. Quoting is always allowed and always means the same thing.

---

## 3. Elements

An **element** is a node, a container, or the diagram itself.

```m2
db                                    // name only; label defaults to the name
db: "Postgres"                        // name + label
db: "Postgres" [shape: cylinder]      // + attributes
db: "Postgres" [shape: cylinder] {    // + children
  wal: "WAL"
}
```

Every part after the name is optional, and the order is fixed:
`name [: label] [attrs] [block]`.

### 3.1 Labels

The value after `:` is the element's label, exactly equivalent to the `label`
attribute. These are the same declaration:

```m2
db: "Postgres"
db [label: "Postgres"]
```

An unquoted label runs to the end of the statement, with a trailing `[...]`
group stripped if it parses as an attribute list. Quote the label when that rule
would guess wrong:

```m2
note: pick one of A, B, C          // label is: pick one of A, B, C
note: "counts as [1, 2]"           // quoted, so the brackets are literal
```

Set `[label: none]` to draw the shape with no text.

### 3.2 Redeclaration merges

Naming an element again updates it rather than creating a second one. This is
how attributes get attached after the fact:

```m2
db: "Postgres"
db [shape: cylinder]      // same element, now a cylinder
db [fill: #E7F5FF]        // still the same element
```

Merging is per attribute: a later value replaces an earlier one for the same
key and leaves other keys alone. Declaration *order* is fixed at first mention,
which is what layout uses for tie-breaking.

### 3.3 Implicit creation

A name that resolves nowhere (§7.1) is created in the current scope:

```m2
cache -> db               // creates both, if they do not already exist
```

This is convenient, and it is also how typos become extra boxes. Set `[strict]`
on the root to require every reference to resolve to a declared element:

```m2
[strict]

api
api -> db                 // error: undeclared element "db"
```

---

## 4. Attributes

Attributes are a comma-separated list in square brackets. A trailing comma is
allowed.

```m2
db [shape: cylinder, fill: #E7F5FF, stroke-width: 2]
```

### 4.1 Flags

An attribute with no value is a boolean set to true. Prefix `!` to set it false.

```m2
title [bold, italic]
child [!shadow]
```

### 4.2 Attributes as a statement

An attribute list standing alone as a statement applies to whatever scope
contains it — a container, or the root.

```m2
[direction: right]        // the diagram flows left to right

cluster: "us-east-1" {
  [direction: down]       // this container flows top to bottom
  [fill: #F8F9FA]
  a -> b
}
```

This is the only way to set root attributes, and it is why m2 needs no top-level
reserved keys.

### 4.3 One flat namespace

d2 splits attributes across `style.*` and bare keys. m2 has a single flat
namespace: `fill`, not `style.fill`. Where a value is naturally compound, it is
a list.

```m2
box [pad: (8, 12), fill: #FFF, stroke: #868E96]
```

---

## 5. Reuse

Two attribute keys cover everything d2 spends four keyword statements on. Both
are ordinary attributes, so neither adds a statement type.

### 5.1 `like` — copy attributes from another element

```m2
db:    "Postgres" [shape: cylinder, fill: #E7F5FF]
cache: "Redis"    [like: db]
```

`like` copies every attribute of the named element except its `label`, then
applies the target's own attributes on top. Inline always wins:

```m2
warm: "Redis" [like: db, fill: #FFF5F5]     // cylinder, but pink
```

A list copies several sources left to right, later winning:

```m2
stripe: "Stripe" [like: (svc, external)]
```

`like` resolves transitively — a source may itself have a `like` — and a cycle
is an error. Only attributes are copied, never children.

It works on edges too, so a shared emphasis style needs no second mechanism:

```m2
hot [hidden, stroke: #E03131, stroke-width: 3]

orders -> billing: "authorize" [like: hot]
billing -> stripe: "charge"    [like: hot]
```

### 5.2 Templates

Because `like` points at an ordinary element, a reusable style is just an
element you do not draw. A hidden container hides its whole subtree, which gives
templates somewhere to live:

```m2
t [hidden] {
  svc      [shape: rect, radius: 4, fill: #FFF, stroke: #4C6EF5]
  store    [shape: cylinder, fill: #E7F5FF, stroke: #1971C2]
  external [stroke-dash: 4, opacity: 0.8]
}

gw:     "API Gateway" [like: t.svc]
pg:     "Postgres"    [like: t.store]
stripe: "Stripe"      [like: (t.svc, t.external)]
```

This is d2's `classes` block, minus the block, minus the keyword, minus the
separate lookup namespace. A template is an element; `t.svc` is a path like any
other.

### 5.3 `src` — children from another file

```m2
aws: "AWS" [src: "./aws.m2"]

myapp -> aws.rds
```

The named file is parsed and its root becomes this element's children. Paths
resolve relative to the importing file, and cycles are an error. Attributes
declared alongside `src` apply to the container itself:

```m2
aws: "AWS" [src: "./aws.m2", fill: #F8F9FA, collapsed]
```

There is no merge-into-current-scope import. Everything a file brings in arrives
under a name you chose, so nothing can be shadowed by surprise.

---

## 6. Edges

```m2
a -> b                    // directed
a <- b                    // directed, reversed
a <-> b                   // bidirectional
a -- b                    // undirected line
```

Labels and attributes attach exactly as they do to elements:

```m2
api -> db: "SELECT"
api -> db: "SELECT" [stroke: #E03131, stroke-dash: 3]
```

Restating an edge declares a *second* edge between the same pair, which is how
parallel connections are drawn. Edges have no names; style them where you
declare them.

### 6.1 Chains

```m2
ingest -> parse -> enrich -> store
```

A chain declares one edge per arrow. A trailing label or attribute list applies
to every edge in the chain:

```m2
a -> b -> c: "sync" [stroke-width: 2]
```

### 6.2 Groups

Parenthesized endpoints fan out. `(a, b) -> (c, d)` declares the four edges of
the cross product, in row-major order.

```m2
(web, mobile, cli) -> gateway: "HTTPS"
gateway -> (auth, catalog, orders)
```

### 6.3 Self edges

```m2
scheduler -> scheduler: "tick"
```

### 6.4 Arrowheads

The arrow token sets direction; `src-head` and `dst-head` set the drawn shape,
including crow's-foot notation for entity diagrams.

```m2
users.org_id -> orgs.id [src-head: many, dst-head: one]
Dog -> Animal [dst-head: triangle]              // UML inheritance
Wheel -- Car  [src-head: filled-diamond]        // UML composition
```

Head values: `none`, `arrow`, `triangle`, `diamond`, `filled-diamond`,
`circle`, `filled-circle`, `cross`, `one`, `many`, `zero-or-one`,
`one-or-many`, `zero-or-many`.

### 6.5 Endpoint labels

```m2
client -> server: "TCP" [src-label: "ephemeral", dst-label: ":443"]
```

---

## 7. Containers and paths

A block makes an element a container.

```m2
vpc: "Production VPC" {
  public: "Public subnet" {
    lb: "Load balancer"
  }
  private: "Private subnet" {
    app: "App servers"
  }
}
```

Elements are addressed by dotted path:

```m2
vpc.public.lb -> vpc.private.app
```

Edges may cross container boundaries freely, in either direction.

### 7.1 Scope resolution

A path has no anchors and no sigils — it is names joined by `.`. Its first
segment resolves in the current scope; if nothing matches, the search continues
outward through each enclosing scope to the root, and the nearest match wins.
Later segments are then looked up strictly inside what the first one found.

```m2
dns: "Route 53"

vpc {
  public { lb }
  private {
    app
    app -> public.lb: "upstream"   // public: found one scope out, in vpc
    app -> dns: "resolve"          // dns: found at the root
  }
}
```

**Resolution searches outward; creation never does.** An implicitly created name
(§3.3) always lands in the current scope, so writing an edge can never
accidentally reach out and modify an enclosing container. The two rules together
mean an outward match only ever happens against something already declared.

A nearer declaration shadows a farther one. To reach a shadowed outer element,
give a path long enough to be unambiguous from where you are, or declare the
statement in an outer scope:

```m2
db: "Primary"

shard {
  db: "Shard-local"     // shadows the outer db inside this container
  worker
  worker -> db          // the shard-local one
}

reporter
reporter -> db          // at the root: the primary
```

With `[strict]` set, a path that resolves in no enclosing scope is an error
rather than a new element, which turns a typo into a message instead of a box.

### 7.2 Declaring into a container

A dotted declaration creates or updates in place, without opening a block:

```m2
vpc.private.cache: "Redis" [shape: cylinder]
```

Missing intermediate segments are created as empty containers, unless `strict`
is set.

### 7.3 Container edges

An edge may name a container as an endpoint. It attaches to the container
boundary rather than to any child:

```m2
internet -> vpc: "443"
```

---

## 8. Structured kinds

`shape` controls an element's outline. `kind` controls how its **children are
interpreted**. It defaults to `node`, or to `container` when the element has
children.

| `kind` | Children are |
| --- | --- |
| `node` | (none) |
| `container` | nested elements |
| `sequence` | participants and ordered messages |
| `table` | columns |
| `class` | fields and methods |
| `fragment` | a sequence-diagram fragment (`loop`, `alt`, `opt`, `par`) |
| `note` | free text attached to another element |
| `layer` | a separate board, not drawn inline |

This replaces d2's overloading of `shape` with `sequence_diagram`, `sql_table`,
and `class` — a shape is a shape.

### 8.1 Sequence

Inside `kind: sequence`, participants appear in declaration order and edges are
messages in source order.

```m2
login: "Login flow" [kind: sequence] {
  user: "User" [shape: person]
  app:  "Web App"
  auth: "Auth API"

  user -> app:  "GET /login"
  app  -> auth: "POST /token" [span]
  auth -> app:  "200 {jwt}" [stroke-dash: 3]

  retry: "3 attempts" [kind: fragment, op: loop] {
    app -> auth: "POST /token"
  }

  app -> user: "302 /home"

  n: "tokens expire in 15m" [kind: note, near: auth]
}
```

`[span]` activates the destination until its reply. `op` selects the fragment
type; an `alt` fragment's branches are its child fragments.

### 8.2 Table

Inside `kind: table`, each child is a column and its label is the column type.

```m2
users [kind: table] {
  id:      uuid        [key: primary]
  email:   text        [unique]
  org_id:  uuid        [key: foreign]
  created: timestamptz
}

orgs [kind: table] {
  id:   uuid [key: primary]
  name: text
}

users.org_id -> orgs.id [src-head: many, dst-head: one]
```

Columns are addressable as paths, which is what makes the foreign-key edge land
on the right rows.

### 8.3 Class

Inside `kind: class`, each child is a member: an optional visibility sigil
(`+` public, `-` private, `#` protected, `~` package), a name, and a label
holding the type or return type.

```m2
Store [kind: class] {
  +name:  string
  -conn:  *sql.DB
  #cache: map[string]Entry

  +Save(e Entry): error
  +Load(id string): (Entry, error)
}

Store -> Repository [dst-head: triangle]
```

Member labels are free text to end of statement, so type syntax passes through
untouched. Quote the label if it ends with something that would parse as an
attribute list.

### 8.4 Layers

A `kind: layer` container is a board of its own: a renderer presents it as a
separate page, tab, or slide instead of drawing it inside its parent. Use it to
drill into a component without cluttering the overview.

```m2
api -> db

internals: "Inside the API" [kind: layer] {
  router -> handler -> repo
}
```

A layer is an ordinary container in every other respect — it nests, it takes
attributes, and paths reach into it.

### 8.5 Notes and text

```m2
n: """md
  **Caution:** this path is not idempotent.
""" [kind: note, near: workers.enrich]

banner: "Draft" [shape: text, font-size: 32, opacity: 0.3]
```

---

## 9. Evaluation semantics

### 9.1 Order

1. Statements are evaluated top to bottom in each scope.
2. A `src` file is parsed and evaluated when its declaration is reached.
3. A `like` is resolved when its declaration is reached, against the source's
   attributes *at that moment* — so a source restyled later does not
   retroactively change elements that already copied it.

### 9.2 Merging

Declaring an existing element merges attributes into it (§3.2). Declaring an
edge always creates a new edge, even between the same pair.

### 9.3 Precedence

When several sources set the same attribute on one element, the winner is the
highest of:

1. **Inline** — written in the element's own `[...]`
2. **Copied** — pulled in by `like`, later sources beating earlier
3. **Inherited** — from the enclosing scope, for inheritable attributes only
   (`font`, `font-size`, `font-color`, `direction`)
4. **Default**

Within one level, later beats earlier.

### 9.4 Errors

These are errors, not warnings: a reference that cannot resolve under `strict`;
a `like` or `src` cycle; an unknown attribute key; a value of the wrong type.

An unknown *value* for a known key is an error too — `[shape: octagon]` fails
rather than silently drawing a rectangle.

---

## 10. Attribute reference

### 10.1 Any element

| Key | Values | Notes |
| --- | --- | --- |
| `label` | string, `none` | same as the `:` form |
| `like` | path or list of paths | copy attributes, §5.1 |
| `src` | string | children from a file, §5.3 |
| `kind` | see §8 | how children are read |
| `shape` | see below | outline |
| `icon` | string (path or URL) | |
| `link` | string | clickable target |
| `tooltip` | string | |
| `near` | path, or `top-left` … `bottom-right` | pin position |
| `width`, `height` | number | in points |
| `pad` | number, `(v, h)`, or `(t, r, b, l)` | |
| `hidden` | flag | keeps it in the model, omits it and its subtree from the render |
| `opacity` | 0–1 or % | |
| `z` | number | draw order |

Shapes: `rect`, `round`, `circle`, `oval`, `diamond`, `hexagon`, `cylinder`,
`queue`, `package`, `document`, `note`, `parallelogram`, `step`, `cloud`,
`person`, `text`, `image`.

### 10.2 Fill and stroke

| Key | Values |
| --- | --- |
| `fill` | color |
| `fill-pattern` | `none`, `dots`, `lines`, `grain` |
| `stroke` | color |
| `stroke-width` | number |
| `stroke-dash` | number (dash length; `0` is solid) |
| `radius` | number (corner radius) |
| `shadow` | flag |
| `depth` | `none`, `3d`, `stack` |

### 10.3 Text

| Key | Values |
| --- | --- |
| `font` | `sans`, `serif`, `mono`, or a family name |
| `font-size` | number |
| `font-color` | color |
| `bold`, `italic`, `underline` | flags |
| `align` | `left`, `center`, `right` |

`font`, `font-size`, and `font-color` are inherited by children.

### 10.4 Containers

| Key | Values |
| --- | --- |
| `direction` | `right`, `left`, `down`, `up` (inherited) |
| `layout` | `auto`, `grid`, `free` |
| `columns`, `rows` | number (with `layout: grid`) |
| `gap` | number, or `(row, col)` |
| `collapsed` | flag |
| `op` | `loop`, `alt`, `opt`, `par` (with `kind: fragment`) |

### 10.5 Edges

| Key | Values |
| --- | --- |
| `label` | string |
| `like` | path or list of paths — §5.1 |
| `src-label`, `dst-label` | string |
| `src-head`, `dst-head` | see §6.4 |
| `stroke`, `stroke-width`, `stroke-dash` | as §10.2 |
| `curve` | `straight`, `orthogonal`, `curved` |
| `weight` | number — layout pull, default `1` |
| `animate` | flag |
| `span` | flag (`kind: sequence` only) |

### 10.6 Root

| Key | Values |
| --- | --- |
| `title` | string |
| `direction` | as §10.4 |
| `theme`, `dark-theme` | theme name |
| `background` | color |
| `sketch` | flag — hand-drawn rendering |
| `strict` | flag — see §3.3 |
| `layout`, `pad`, `gap`, `font` | as above |

---

## 11. Grammar

EBNF. Whitespace between tokens is insignificant; a newline is a separator
(§2.2). Comments are stripped by the lexer.

```ebnf
diagram     = [ statement ] { separator [ statement ] } ;
separator   = ";" | newline ;

statement   = declaration | edge | attrs ;

declaration = path [ ":" label ] [ attrs ] [ block ] ;

edge        = endpoints { arrow endpoints } [ ":" label ] [ attrs ] ;
endpoints   = path | "(" path { "," path } [ "," ] ")" ;
arrow       = "->" | "<-" | "<->" | "--" ;

attrs       = "[" [ attr { "," attr } [ "," ] ] "]" ;
attr        = name ":" value
            | name                          (* flag, true  *)
            | "!" name ;                    (* flag, false *)

block       = "{" [ statement ] { separator [ statement ] } "}" ;

path        = segment { "." segment } ;
segment     = name | string ;

value       = number | percent | color | string | list | path ;
list        = "(" [ value { "," value } [ "," ] ] ")" ;

label       = string | free-text ;
name        = bare-name | string ;
```

`free-text` is everything to the end of the statement, minus a trailing `attrs`
group if one parses (§3.1). A bare word is a single-segment `path`; the
attribute key decides whether to read it as an enum or a reference (§2.6).
`bare-name` and the string forms are defined in §2.4 and §2.5.

### 11.1 No reserved words

There are no keywords. Every attribute key — `shape`, `label`, `near`, `like`,
`style` — is available as an element name, because attribute keys are only ever
read inside `[...]`:

```m2
shape: "Shape"          // an element named shape
shape -> label          // an edge between two ordinary elements
shape [shape: circle]   // the element named shape, drawn as a circle
```

---

## 12. Differences from d2

Each of these is a deliberate departure, listed with the problem it solves.

| # | d2 | m2 | Why |
| --- | --- | --- | --- |
| 1 | Attributes are children: `x.shape: circle` | Attributes are bracketed: `x [shape: circle]` | Element names and attribute keys stop competing for one namespace. No reserved keys, no accidental node called `width`. |
| 2 | `x.style.fill: red` | `x [fill: red]` | One flat attribute namespace; nothing to remember about which keys live under `style`. |
| 3 | `#` starts a comment | `//` and `/* */` start comments | Frees `#` for hex colors, the most common literal in a diagram. |
| 4 | Undeclared names silently become nodes | Same by default, `[strict]` to forbid it | Keeps the fast path fast, makes typo-proof diagrams possible. |
| 5 | Names resolve outward, and an unresolved one is created wherever the search stopped | Resolution searches outward, creation is always local, `[strict]` turns a miss into an error | Reaching an outer container is the common case and stays syntax-free; the hazard was never the search, it was a failed search quietly inventing a node. |
| 6 | `a -> b` only | `(a, b) -> (c, d)` cross products | Fan-in and fan-out without repeating yourself. |
| 7 | `classes` block, `vars` block, globs (`*.style.fill`), `@`/`...@` imports | `[like: path]` and `[src: "file"]` | Four mechanisms with three syntaxes become two attribute keys. A style bundle is an element; importing binds under a name you chose. |
| 8 | `shape: sequence_diagram`, `shape: sql_table`, `shape: class` | `kind: sequence`, `kind: table`, `kind: class` | Separates "what outline" from "how are children interpreted". |
| 9 | `\|md ... \|` text blocks | `"""md ... """` | Familiar from other languages; no delimiter collision with tables or code containing `\|`. |
| 10 | layers, scenarios, steps | `kind: layer` | One concept, expressed with machinery the language already has. |
| 11 | `(a -> b)[0]` positional edge references | none — style edges where you declare them | Positional indexes shift when you insert an edge, and the feature only existed to patch edges after the fact. |

What m2 keeps from d2, on purpose: dotted paths, `{}` containers, the four
arrow tokens, `key: label` reading as "a box with this text", edges that cross
container boundaries, and the property that a diagram's source is diffable and
reviewable line by line.

---

## 13. Worked example

```m2
// checkout.m2 — order checkout, end to end
[title: "Checkout", direction: right, font: sans, strict]

t [hidden] {
  svc      [shape: rect, radius: 4, fill: #FFF, stroke: #4C6EF5]
  store    [shape: cylinder, fill: #E7F5FF, stroke: #1971C2]
  external [stroke-dash: 4, stroke: #868E96, opacity: 0.8]
  money    [stroke: #E03131, stroke-width: 3]
}

client: "Browser"   [shape: person]
edge:   "CDN + WAF" [shape: hexagon, fill: #F1F3F5]

platform: "Platform" {
  [direction: down, fill: #F8F9FA, pad: 16]

  gw:      "API Gateway"     [like: t.svc]
  cart:    "Cart Service"    [like: t.svc]
  orders:  "Order Service"   [like: t.svc]
  billing: "Billing Service" [like: t.svc]

  bus: "Event Bus" [shape: queue, fill: #FFF3BF]

  gw -> cart
  gw -> orders
  orders -> billing: "authorize" [like: t.money]
  (cart, orders, billing) -> bus: "emit" [stroke-dash: 2, weight: 0.5]
}

data: "Data" {
  [direction: down]
  pg:    "Postgres" [like: t.store]
  redis: "Redis"    [like: t.store]
}

stripe:   "Stripe" [like: (t.svc, t.external)]
sendgrid: "Email"  [like: (t.svc, t.external)]

client -> edge:        "HTTPS"
edge   -> platform.gw: "mTLS" [stroke-width: 2]

platform.cart    -> data.redis: "session"
platform.orders  -> data.pg:    "orders"
platform.billing -> stripe:     "charge" [like: t.money]
platform.bus     -> sendgrid:   "receipt"

n: """md
  **SLO:** p99 checkout < 800ms
  Stripe timeouts fall back to the async queue.
""" [kind: note, near: platform.billing]

billing-detail: "Inside Billing" [kind: layer] {
  [direction: down]

  flow: "Authorize" [kind: sequence] {
    orders:  "Order Service"
    billing: "Billing Service"
    stripe:  "Stripe"
    pg:      "Postgres"

    orders  -> billing: "POST /authorize" [span]
    billing -> pg:      "insert attempt"
    billing -> stripe:  "PaymentIntent" [span]

    retry: "up to 3, backoff 2^n" [kind: fragment, op: loop] {
      billing -> stripe: "PaymentIntent"
    }

    stripe  -> billing: "requires_action" [stroke-dash: 3]
    billing -> orders:  "202 + redirect"
  }

  ledger [kind: table] {
    id:       uuid [key: primary]
    order_id: uuid [key: foreign]
    amount_c: bigint
    status:   text
    created:  timestamptz
  }
}
```

Nothing in that diagram is outside the three statement types. The `t` block is a
declaration, `[title: ..., strict]` is an attribute list, `client -> edge` is an
edge — and the styling, reuse, layering, sequence chart, and table are all
attribute values carried by those same three forms.
