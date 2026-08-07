# m2 — a language for describing diagrams

m2 is a declarative language for describing diagrams: boxes, arrows, containers,
and the styling that goes with them. It is a redesign of
[d2](https://d2lang.com/), keeping what makes d2 pleasant — text that reads like
the picture it produces — while fixing the parts of d2's grammar that make it
hard to write tooling for and easy to get wrong.

The one structural idea to hold on to:

> **Attributes live in `[...]`. Children live in `{...}`. They are separate
> namespaces and never collide.**

In d2, `shape`, `style`, `label`, `near`, `width` and about twenty other words
are reserved *keys*, sharing a namespace with your node names. m2 has no
reserved keys at all — a node may be called `style`, `shape`, or `label`,
because attributes are never written as children.

This document specifies the syntax and the evaluation semantics. It does not
specify layout or rendering beyond what the syntax must express.

- [1. A first look](#1-a-first-look)
- [2. Lexical structure](#2-lexical-structure)
- [3. Elements](#3-elements)
- [4. Attributes](#4-attributes)
- [5. Edges](#5-edges)
- [6. Containers and paths](#6-containers-and-paths)
- [7. Selectors](#7-selectors)
- [8. Classes](#8-classes)
- [9. Variables](#9-variables)
- [10. Imports](#10-imports)
- [11. Layers and steps](#11-layers-and-steps)
- [12. Structured kinds](#12-structured-kinds)
- [13. Evaluation semantics](#13-evaluation-semantics)
- [14. Attribute reference](#14-attribute-reference)
- [15. Grammar](#15-grammar)
- [16. Differences from d2](#16-differences-from-d2)
- [17. Worked example](#17-worked-example)

---

## 1. A first look

```m2
// architecture.m2
[title: "Ingest pipeline", direction: right]

edge:  "Edge Proxy"    [shape: hexagon]
queue: "Kafka"         [shape: queue, fill: #FFF3BF]

workers: "Workers" {
  [direction: down]
  parse:   "Parser"
  enrich:  "Enricher"
  parse -> enrich
}

store: "Postgres" [shape: cylinder]

edge -> queue:      "events"
queue -> workers.parse
workers.enrich -> store: "upsert" [stroke-dash: 3]
```

Six kinds of statement cover the whole language:

| Statement | Example | Meaning |
| --- | --- | --- |
| Declaration | `db: "Postgres" [shape: cylinder]` | create or update an element |
| Edge | `api -> db: "query"` | connect elements |
| Attribute list | `[direction: right]` | set attributes on the enclosing scope |
| Selector | `.critical [stroke: #E03131]` | set attributes on everything matching |
| Keyword | `var accent: #4C6EF5` | define a class, var, import, layer, or step |
| Comment | `// ...` | ignored |

---

## 2. Lexical structure

### 2.1 Files

An m2 source file uses the extension `.m2`, is encoded in UTF-8, and is
newline-separated. A file is a sequence of statements evaluated in the **root
scope**, which is itself an element (the diagram).

### 2.2 Statement termination

A statement ends at a newline or a `;`. A statement continues across newlines
while it is inside `[]`, `{}`, `()`, or a triple-quoted string.

```m2
a; b; c                  // three declarations

servers [
  shape: hexagon,        // attribute lists may span lines
  fill: #F1F3F5,
]
```

An opening `{` must appear on the same line as the statement that introduces it.
This keeps "newline ends a statement" true without lookahead.

### 2.3 Comments

```m2
// line comment, runs to end of line

/* block comment,
   may span lines and nest */
```

m2 uses `//` and `/* */` rather than d2's `#` so that `#` is free for hex color
literals. `#4C6EF5` is a value, not a comment.

### 2.4 Names

A **bare name** is a run of characters drawn from letters, digits, and
`_ - + # ! ? @ % ' &`, plus internal whitespace (runs of whitespace collapse to
a single space, and leading/trailing whitespace is trimmed).

A bare name ends at any of `: ; , . { } [ ] ( )`, an arrow token (`->`, `<-`,
`<->`, `--`), a comment opener, or end of line.

A bare name may not *begin* with a sigil — `*`, `~`, `^`, `$`, `&` — because
those introduce selectors, path anchors, and references. It may not begin with
`--` or `->`.

```m2
Order Service            // legal: internal spaces are fine
#cache                   // legal: # is only special before hex digits in a value
-conn                    // legal: a single leading - is not an arrow
"api.example.com"        // must be quoted: bare . is the path separator
"*"                      // must be quoted: bare * is a selector
```

Names are case-sensitive. Two names that differ only in internal whitespace runs
are the same name.

### 2.5 Strings

```m2
"double quoted, with \" \\ \n \t escapes and $var interpolation"
'single quoted, no escapes, no interpolation — a raw string'
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
language name for syntax-highlighted code. Interpolation applies inside `"` and
`"""` forms; write `$$` for a literal `$`.

### 2.6 Values

| Form | Examples |
| --- | --- |
| Word | `cylinder`, `right`, `dashed` |
| Number | `2`, `0.5`, `-3` |
| Percentage | `50%` |
| Color | `#4C6EF5`, `#FFF3BF80`, `#eee`, `red`, `transparent` |
| String | `"Postgres"`, `'raw'`, `"""md ... """` |
| List | `(4, 8)`, `(critical, external)` |
| Reference | `$accent`, `&query` |

Bare words and numbers need no quotes. Anything containing a delimiter should be
quoted; quoting is always allowed and always means the same thing.

---

## 3. Elements

An **element** is a node, a container, or the diagram itself. Declaring one:

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

The value after `:` is the element's label. It is exactly equivalent to the
`label` attribute; these are the same declaration:

```m2
db: "Postgres"
db [label: "Postgres"]
```

An unquoted label runs to the end of the line, with a trailing `[...]` group
stripped if it parses as an attribute list. Quote the label when that rule would
guess wrong:

```m2
note: pick one of A, B, C          // label is: pick one of A, B, C
note: "counts as [1, 2]"           // quoted, so the brackets are literal
```

Set `[label: none]` to draw the shape with no text.

### 3.2 Redeclaration merges

Naming an element again updates it rather than creating a second one. This is
how you attach attributes after the fact, and how imported diagrams get
customized:

```m2
db: "Postgres"
db [shape: cylinder]      // same element, now a cylinder
db [fill: #E7F5FF]        // still the same element
```

Merge is shallow per attribute: a later value replaces an earlier one for the
same key, and leaves other keys alone. Declaration *order* is preserved from
the first mention, which is what layout uses for tie-breaking.

### 3.3 Implicit creation

Referring to an undeclared name in an edge creates it:

```m2
cache -> db               // creates both cache and db if they do not exist
```

This is convenient and it is also how typos become extra boxes. Set
`[strict]` on the root to require every reference to resolve to a declared
element:

```m2
[strict]

api
api -> db                 // error: undeclared element "db"
```

---

## 4. Attributes

Attributes are written as a comma-separated list in square brackets. A trailing
comma is allowed.

```m2
db [shape: cylinder, fill: #E7F5FF, stroke-width: 2]
```

### 4.1 Flags

An attribute with no value is a boolean set to true. Prefix `!` to set it false.

```m2
title [bold, italic]
child [!shadow]
```

### 4.2 Attributes of the enclosing scope

An attribute list on its own line applies to whatever scope contains it — a
container, or the root.

```m2
[direction: right]        // the diagram flows left-to-right

cluster: "us-east-1" {
  [direction: down]       // this container flows top-to-bottom
  [fill: #F8F9FA]
  a -> b
}
```

This is the only way to set root attributes, and it removes the need for
d2's top-level reserved keys.

### 4.3 No nesting

d2 splits attributes across `style.*` and bare keys. m2 has one flat namespace:
`fill`, not `style.fill`. Where a value is naturally compound, it is a list.

```m2
box [pad: (8, 12), fill: #FFF, stroke: #868E96]
```

### 4.4 Spread

`...` splices one attribute list into another. The source may be a variable or
a class name.

```m2
var boxy: [shape: rect, radius: 0, stroke-width: 2]

card  [...$boxy, fill: #FFF]
alert [...$boxy, fill: #FFF5F5, stroke: #E03131]
```

Later keys win, so `fill` above overrides any `fill` inside `$boxy`.

---

## 5. Edges

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

### 5.1 Chains

```m2
ingest -> parse -> enrich -> store
```

A chain declares one edge per arrow. A label or attribute list at the end
applies to every edge in the chain:

```m2
a -> b -> c: "sync" [stroke-width: 2]
```

### 5.2 Groups

Parenthesized endpoints fan out. `(a, b) -> (c, d)` declares the four edges of
the cross product, in row-major order.

```m2
(web, mobile, cli) -> gateway: "HTTPS"
gateway -> (auth, catalog, orders)
```

### 5.3 Self edges

```m2
scheduler -> scheduler: "tick"
```

### 5.4 Arrowheads

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

### 5.5 Naming edges

Give an edge an `id` to refer to it later. `&id` is the reference form.

```m2
api -> db: "query" [id: hot-path]

&hot-path [stroke: #E03131, stroke-width: 3, animate]
```

Without an `id`, an edge has no name — restating `api -> db` declares a
*second* edge between the same pair, which is how you draw parallel
connections.

### 5.6 Endpoint labels

```m2
client -> server: "TCP" [src-label: "ephemeral", dst-label: ":443"]
```

---

## 6. Containers and paths

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

### 6.1 Scope resolution

The first segment of a path resolves **in the current scope only**. m2 does not
search enclosing scopes implicitly — a name means what it says where it is
written.

Two anchors navigate explicitly:

| Anchor | Meaning |
| --- | --- |
| `~` | the root scope |
| `^` | the parent scope; repeatable as `^.^` |

```m2
vpc {
  public { lb }
  private {
    app
    app -> ^.public.lb          // sibling container
    app -> ~.dns: "resolve"     // an element at the root
  }
}

dns: "Route 53"
```

### 6.2 Declaring into a container

A dotted declaration creates or updates in place, without opening a block:

```m2
vpc.private.cache: "Redis" [shape: cylinder]
```

Missing intermediate segments are created as empty containers, unless `strict`
is set.

### 6.3 Container edges

An edge may name a container as an endpoint. The edge attaches to the container
boundary rather than to any child:

```m2
internet -> vpc: "443"
```

---

## 7. Selectors

A selector statement applies attributes to every element or edge it matches in
the current scope. Selectors do not declare anything — they only style what
already exists at the point they appear.

| Selector | Matches |
| --- | --- |
| `*` | direct children of the current scope |
| `**` | all descendants of the current scope |
| `.name` | elements carrying class `name` |
| `->` | all edges in the current scope |
| `**->` | all edges at or below the current scope |
| `&id` | the edge with that id |
| `path.*` | direct children of `path` |
| `path.**` | all descendants of `path` |
| `(s1, s2)` | the union of the selectors in the group |

```m2
** [font: mono, font-size: 12]        // whole diagram
-> [stroke: #ADB5BD]                  // every edge at this level

vpc.** [fill: #F8F9FA]                // everything inside the vpc

.external [stroke-dash: 4, opacity: 0.7]
```

Selectors run in source order, so a later one overrides an earlier one. An
attribute written directly on an element always beats a selector — see
[§13.3](#133-precedence).

---

## 8. Classes

A class is a named attribute list.

```m2
class stateful [shape: cylinder, fill: #E7F5FF, stroke: #1971C2]
class external [stroke-dash: 4, opacity: 0.7]

db:    "Postgres" [class: stateful]
cache: "Redis"    [class: stateful]
stripe: "Stripe"  [class: (stateful, external)]     // multiple
```

Classes may be applied to edges too, and may be defined in terms of each other
with spread:

```m2
class base  [font: mono, stroke-width: 2]
class alarm [...base, stroke: #E03131, bold]
```

Class definitions are scoped to the block they appear in and are visible to
nested blocks. A nested definition of the same name shadows the outer one.

---

## 9. Variables

```m2
var accent: #4C6EF5
var region: "us-east-1"
var boxy:   [shape: rect, radius: 0]

api: "API ($region)" [fill: $accent, ...$boxy]
```

Variables hold a value or an attribute list. They interpolate inside
double-quoted and triple-quoted strings — `$name` or `${name}` when the
following character would otherwise be part of the name. Write `$$` for a
literal `$`.

Variables are lexically scoped like classes, and may not be reassigned within a
scope. Redefining in a nested scope shadows.

---

## 10. Imports

```m2
import "./palette.m2"                 // merge into the current scope
import "./aws.m2" as aws              // bind under a namespace
import "./aws.m2" as aws [prefix]     // ...and prefix its labels too
```

A bare import evaluates the file's statements in the current scope: its
classes, vars, and elements become yours, and redeclaration rules apply
normally, so you can import a shared diagram and then adjust it.

A named import places the file's root under a single container of that name:

```m2
import "./aws.m2" as aws

myapp -> aws.rds
```

Paths are resolved relative to the importing file. Import cycles are an error.

---

## 11. Layers and steps

A diagram may carry alternate views. Both forms name a board that a renderer can
present as a separate page, tab, or slide.

```m2
layer name { ... }        // an independent board; inherits nothing
step  name { ... }        // a delta applied on top of the previous board
```

`layer` is a fresh start — useful for drilling into a component:

```m2
api -> db

layer internals: "Inside the API" {
  router -> handler -> repo
}
```

`step` chains: each step begins with everything the previous board had, and
applies its statements as changes. Steps are how you build an animation or a
walkthrough.

```m2
client; lb; app; db
client -> lb -> app
app -> db: "SELECT" [id: lookup]

step "1. cache miss" {
  &lookup [stroke: #E03131]
  cache: "Redis" [fill: #FFF5F5]
}

step "2. warm" {
  cache [fill: #EBFBEE]     // carried over from the previous step, restyled
}
```

Because a step is a delta, it can hide things as well as add them:

```m2
step "simplified" {
  db [hidden]
}
```

---

## 12. Structured kinds

`shape` controls an element's outline. `kind` controls how its **children are
interpreted**. Defaults to `node`, or `container` when the element has children.

| `kind` | Children are |
| --- | --- |
| `node` | (none) |
| `container` | nested elements |
| `sequence` | participants and ordered messages |
| `table` | columns |
| `class` | fields and methods |
| `fragment` | a sequence-diagram fragment (`loop`, `alt`, `opt`, `par`) |
| `note` | free text attached to another element |

This replaces d2's overloading of `shape` with `sequence_diagram`, `sql_table`,
and `class` — a shape is a shape.

### 12.1 Sequence

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

  app  -> user: "302 /home"

  n: "tokens expire in 15m" [kind: note, near: auth]
}
```

`[span]` activates the destination until its reply. `op` selects the fragment
type; an `alt` fragment's branches are its child fragments.

### 12.2 Table

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

Columns are addressable as paths, which is what makes the foreign-key edge
above land on the right rows.

### 12.3 Class

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

Member labels are free text to end of line, so type syntax passes through
untouched. Quote the label if it ends with something that would parse as an
attribute list.

### 12.4 Notes and text

```m2
n: """md
  **Caution:** this path is not idempotent.
""" [kind: note, near: workers.enrich]

banner: "Draft" [shape: text, font-size: 32, opacity: 0.3]
```

---

## 13. Evaluation semantics

### 13.1 Order

1. Imports are resolved, depth first, and their statements evaluated in place.
2. Statements are evaluated top to bottom in each scope.
3. Selectors match against the state of the diagram *at the point they appear*.
   A selector never sees elements declared after it.
4. Layers and steps are evaluated after their parent board is complete.

### 13.2 Merging

Declaring an existing element merges attributes into it (§3.2). Declaring an
existing *edge* creates a new parallel edge, unless it carries an `id` that
already exists, in which case it merges.

### 13.3 Precedence

When several sources set the same attribute on one element, the winner is the
highest of:

1. **Inline** — written in the element's own `[...]`
2. **Selector** — applied by `*`, `**`, `.class`, `->`, `&id`
3. **Class** — via `[class: ...]`
4. **Scope** — a bare `[...]` list on the enclosing container, for inheritable
   attributes only (`font`, `font-size`, `font-color`, `direction`)
5. **Default**

Within one level, later beats earlier.

### 13.4 Errors

These are errors, not warnings: a reference that cannot resolve under `strict`;
an import cycle; a `^` that walks above the root; an unknown attribute key; a
value of the wrong type; reassigning a `var` in the same scope; an `id`
collision between two edges declared with different endpoints.

An unknown *value* for a known key is an error too — `[shape: octagon]` fails
rather than silently drawing a rectangle.

---

## 14. Attribute reference

### 14.1 Any element

| Key | Values | Notes |
| --- | --- | --- |
| `label` | string, `none` | same as the `:` form |
| `class` | name or list | |
| `kind` | see §12 | how children are read |
| `shape` | see below | outline |
| `icon` | string (path or URL) | |
| `link` | string | clickable target |
| `tooltip` | string | |
| `near` | path, or `top-left` … `bottom-right` | pin position |
| `width`, `height` | number | in points |
| `pad` | number, or `(v, h)`, or `(t, r, b, l)` | |
| `hidden` | flag | keep in the model, omit from the render |
| `opacity` | 0–1 or % | |
| `z` | number | draw order |

Shapes: `rect`, `round`, `circle`, `oval`, `diamond`, `hexagon`, `cylinder`,
`queue`, `package`, `document`, `note`, `parallelogram`, `step`, `cloud`,
`person`, `text`, `image`.

### 14.2 Fill and stroke

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

### 14.3 Text

| Key | Values |
| --- | --- |
| `font` | `sans`, `serif`, `mono`, or a family name |
| `font-size` | number |
| `font-color` | color |
| `bold`, `italic`, `underline` | flags |
| `align` | `left`, `center`, `right` |

`font`, `font-size`, and `font-color` are inherited by children.

### 14.4 Containers

| Key | Values |
| --- | --- |
| `direction` | `right`, `left`, `down`, `up` (inherited) |
| `layout` | `auto`, `grid`, `free` |
| `columns`, `rows` | number (with `layout: grid`) |
| `gap` | number, or `(row, col)` |
| `collapsed` | flag |

### 14.5 Edges

| Key | Values |
| --- | --- |
| `id` | name |
| `label` | string |
| `src-label`, `dst-label` | string |
| `src-head`, `dst-head` | see §5.4 |
| `stroke`, `stroke-width`, `stroke-dash` | as §14.2 |
| `curve` | `straight`, `orthogonal`, `curved` |
| `weight` | number — layout pull, default `1` |
| `animate` | flag |
| `span` | flag (`kind: sequence` only) |

### 14.6 Root

| Key | Values |
| --- | --- |
| `title` | string |
| `direction` | as §14.4 |
| `theme`, `dark-theme` | theme name |
| `background` | color |
| `sketch` | flag — hand-drawn rendering |
| `strict` | flag — see §3.3 |
| `layout`, `pad`, `gap`, `font` | as above |

---

## 15. Grammar

EBNF. Whitespace between tokens is insignificant except that a newline
terminates a statement (§2.2).

```ebnf
diagram     = { statement } ;

statement   = declaration
            | edge
            | attrs
            | selector
            | keyword
            | ";" ;

declaration = path [ ":" label ] [ attrs ] [ block ] ;

edge        = endpoints { arrow endpoints } [ ":" label ] [ attrs ] ;
endpoints   = path | "(" path { "," path } [ "," ] ")" ;
arrow       = "->" | "<-" | "<->" | "--" ;

attrs       = "[" [ attr { "," attr } [ "," ] ] "]" ;
attr        = name ":" value
            | name                          (* flag, true  *)
            | "!" name                      (* flag, false *)
            | "..." reference ;             (* spread      *)

selector    = sel attrs ;
sel         = [ path "." ] ( "*" | "**" )
            | "." name
            | [ "**" ] "->"
            | "&" name
            | "(" sel { "," sel } [ "," ] ")" ;

keyword     = "class"  name attrs
            | "var"    name ":" value
            | "import" string [ "as" name ] [ attrs ]
            | "layer"  name [ ":" label ] block
            | "step"   name [ ":" label ] block ;

block       = "{" { statement } "}" ;

path        = [ anchor "." ] segment { "." segment } ;
anchor      = "~" | "^" { "." "^" } ;
segment     = name | string ;

value       = word | number | percent | color | string | list | reference ;
list        = "(" [ value { "," value } [ "," ] ] ")" ;
reference   = "$" name | "$" "{" name "}" | "&" name ;

label       = string | free-text ;
name        = bare-name | string ;
```

`free-text` is everything to end of line, minus a trailing `attrs` group if one
parses (§3.1). `bare-name` and the string forms are defined in §2.4 and §2.5.

### 15.1 Keyword disambiguation

`class`, `var`, `import`, `layer`, and `step` are keywords **only** at the start
of a statement and only when what follows matches the keyword form above.
Everywhere else they are ordinary names:

```m2
class critical [stroke: #E03131]     // keyword: defines a class
class: "Java class"                  // declaration: an element named "class"
class -> instance                    // edge: the same element
```

There are no other reserved words. Every attribute key — `shape`, `label`,
`near`, `style` — is available as an element name, because attribute keys are
only ever read inside `[...]`.

---

## 16. Differences from d2

Each of these is a deliberate departure, listed with the problem it solves.

| # | d2 | m2 | Why |
| --- | --- | --- | --- |
| 1 | Attributes are children: `x.shape: circle` | Attributes are bracketed: `x [shape: circle]` | Node names and attribute keys stop competing for one namespace. No reserved keys, no accidental node called `width`. |
| 2 | `x.style.fill: red` | `x [fill: red]` | One flat attribute namespace; nothing to remember about which keys live under `style`. |
| 3 | `#` starts a comment | `//` and `/* */` start comments | Frees `#` for hex colors, the single most common literal in a diagram. |
| 4 | Undeclared names silently become nodes | Same by default, `[strict]` to forbid it | Keeps the fast path fast, makes typo-proof diagrams possible. |
| 5 | Names resolve by searching outward | Resolve in the current scope; `^` and `~` navigate | A name means the same thing wherever it appears. |
| 6 | `a -> b` only | `(a, b) -> (c, d)` cross products | Fan-in and fan-out without repeating yourself. |
| 7 | `(a -> b)[0]` positional edge refs | `[id: name]` and `&name` | Stable under edits; positional indexes shift when you insert an edge. |
| 8 | Globs: `*.style.fill: red` | Selectors: `**`, `.class`, `->`, `&id` | CSS-shaped and readable, and edges get first-class selection. |
| 9 | `shape: sequence_diagram`, `shape: sql_table`, `shape: class` | `kind: sequence`, `kind: table`, `kind: class` | Separates "what outline" from "how are children interpreted". |
| 10 | `\|md ... \|` text blocks | `"""md ... """` | Familiar from other languages; no delimiter collision with tables or code containing `\|`. |
| 11 | layers, scenarios, steps | `layer` and `step` | Two concepts — independent board, cumulative delta — cover the same ground. |
| 12 | Classes via a `classes` block | `class name [attrs]` | One statement form, lexically scoped, composable with spread. |

What m2 keeps from d2, on purpose: dotted paths, `{}` containers, the four
arrow tokens, `key: label` reading as "box with this text", edges that cross
container boundaries, and the general property that a diagram's source is
diffable and reviewable line by line.

---

## 17. Worked example

```m2
// checkout.m2 — order checkout, end to end
[title: "Checkout", direction: right, font: sans, strict]

import "./palette.m2"          // provides $accent, $muted, $danger

class svc      [shape: rect, radius: 4, fill: #FFF, stroke: $accent]
class store    [shape: cylinder, fill: #E7F5FF, stroke: #1971C2]
class external [stroke-dash: 4, stroke: $muted, opacity: 0.8]

client: "Browser" [shape: person]

edge: "CDN + WAF" [shape: hexagon, fill: #F1F3F5]

platform: "Platform" {
  [direction: down, fill: #F8F9FA, pad: 16]

  gw:      "API Gateway"     [class: svc]
  cart:    "Cart Service"    [class: svc]
  orders:  "Order Service"   [class: svc]
  billing: "Billing Service" [class: svc]

  bus: "Event Bus" [shape: queue, fill: #FFF3BF]

  gw -> cart
  gw -> orders
  orders -> billing: "authorize" [id: authz]
  (cart, orders, billing) -> bus: "emit" [stroke-dash: 2, weight: 0.5]
}

data: "Data" {
  [direction: down]
  pg:    "Postgres" [class: store]
  redis: "Redis"    [class: store, shape: cylinder]
}

stripe: "Stripe"   [class: (svc, external)]
sendgrid: "Email"  [class: (svc, external)]

client -> edge:        "HTTPS"
edge   -> platform.gw: "mTLS" [stroke-width: 2]

platform.cart    -> data.redis: "session"
platform.orders  -> data.pg:    "orders"
platform.billing -> stripe:     "charge" [id: charge]
platform.bus     -> sendgrid:   "receipt"

// the money path gets emphasis
(&authz, &charge) [stroke: $danger, stroke-width: 3]

n: """md
  **SLO:** p99 checkout < 800ms
  Stripe timeouts fall back to the async queue.
""" [kind: note, near: platform.billing]

layer billing-detail: "Inside Billing" {
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
    id:         uuid        [key: primary]
    order_id:   uuid        [key: foreign]
    amount_c:   bigint
    status:     text
    created:    timestamptz
  }
}
```

The `(&authz, &charge) [...]` line is worth a second look: parenthesized
groups work in selector position too, so a single statement can restyle several
named edges at once.
