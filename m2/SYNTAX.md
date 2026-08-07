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

Two rules do most of the work. The first keeps names and configuration apart:

> **Attributes live in `[...]`. Children live in `{...}`. They are separate
> namespaces and never collide.**

In d2, `shape`, `style`, `label`, `near`, `width` and about twenty other words
are reserved *keys* sharing a namespace with your node names. In m2 an element
may be named `style`, `shape`, or `label`, because attributes are never written
as children.

The second rule eliminates name resolution entirely:

> **A name identifies exactly one element in the whole diagram.**

Names are global, so there are no paths, no scopes to search, no shadowing, and
no anchors. `api -> db` means the same thing written anywhere.

This document specifies the syntax and the evaluation semantics. It does not
specify layout or rendering beyond what the syntax must express.

- [1. A first look](#1-a-first-look)
- [2. Lexical structure](#2-lexical-structure)
- [3. Elements](#3-elements)
- [4. Attributes](#4-attributes)
- [5. Reuse](#5-reuse)
- [6. Edges](#6-edges)
- [7. Containers](#7-containers)
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

proxy: "Edge Proxy" [shape: hexagon]
queue: "Kafka"      [shape: queue, fill: #FFF3BF]

workers: "Workers" {
  [direction: down]
  parse:  "Parser"
  enrich: "Enricher"
  parse -> enrich
}

store: "Postgres" [shape: cylinder]

proxy -> queue: "events"
queue -> parse
enrich -> store: "upsert" [stroke-dash: 3]
```

Note the last two edges: `parse` and `enrich` live inside `workers`, but they
are named directly, because a name is a name wherever you write it.

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
`_ - + . # ! ? @ % ' ~ ^ $ &`, plus internal whitespace (runs of whitespace
collapse to a single space; leading and trailing whitespace is trimmed).

A bare name ends at any of `: ; , { } [ ] ( )`, an arrow token (`->`, `<-`,
`<->`, `--`), a comment opener, or end of line. It may not *begin* with `--` or
`->`.

Since m2 has no paths, `.` carries no meaning and is an ordinary name character.
So is every character that was once a sigil:

```m2
Order Service            // internal spaces are fine
api.example.com          // a single name; . is not a separator
us-east-1.rds            // likewise
#cache                   // # is only a color prefix inside a value
-conn                    // a single leading - is not an arrow
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
| List | `(4, 8)`, `(tpl-svc, tpl-external)` |

A bare word is lexed as a name; the attribute key decides whether to read it as
an enum or as a reference to an element. `shape: cylinder` reads `cylinder` as
an enum, `like: base` reads `base` as a reference. Quoting is always allowed and
always means the same thing.

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

### 3.1 Names are global

A name identifies exactly one element in the whole diagram — however deeply it
is nested, and whichever file it came from. Nesting groups elements; it does not
create a namespace.

```m2
vpc: "Production VPC" {
  public:  "Public subnet"  { lb: "Load balancer" }
  private: "Private subnet" { app: "App servers" }
}

lb -> app                 // no qualification needed, from anywhere
```

That is the entirety of name resolution in m2. There is no lookup order, no
scope chain, no shadowing, and nothing to disambiguate.

The cost is that you choose names for a single flat namespace, and diagrams with
repetitive structure need a naming convention to stay unique. Because `.` is an
ordinary character (§2.4), the convention can simply look like qualification:

```m2
users [kind: table] {
  users.id:    uuid       // one name that happens to contain a dot
  users.email: text
}

orgs [kind: table] {
  orgs.id:   uuid
  orgs.name: text
}
```

### 3.2 Labels

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

### 3.3 Redeclaration merges

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

Merging is also what enforces global uniqueness — reusing a name does not raise
a conflict, it simply yields one element. That is the one hazard of a flat
namespace: two containers that both declare `cache` get a single shared `cache`,
not two. `[strict]` catches it (§3.4).

### 3.4 Strict mode

A name that matches no declared element is created on the spot:

```m2
cache -> db               // creates both, if they do not already exist
```

This is convenient, and it is also how typos become extra boxes. Set `[strict]`
on the root for two additional checks:

```m2
[strict]

api
api -> db                 // error: undeclared element "db"
```

Under `[strict]`:

1. Every reference must resolve to an already-declared element.
2. Only the first declaration of a name may place it in a container. A later
   declaration of the same name inside a *different* container is an error
   rather than a silent merge.

The second check is what turns an accidental name collision into a message.

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
stripe: "Stripe" [like: (tpl-svc, tpl-external)]
```

`like` resolves transitively — a source may itself have a `like` — and a cycle
is an error. Only attributes are copied, never children.

It works on edges too, so a shared emphasis style needs no second mechanism:

```m2
tpl-hot [hidden, stroke: #E03131, stroke-width: 3]

orders -> billing: "authorize" [like: tpl-hot]
billing -> stripe: "charge"    [like: tpl-hot]
```

### 5.2 Templates

Because `like` names an ordinary element, a reusable style is just an element
you do not draw. A hidden container hides its whole subtree, which gives
templates somewhere to live:

```m2
tpl [hidden] {
  tpl-svc      [shape: rect, radius: 4, fill: #FFF, stroke: #4C6EF5]
  tpl-store    [shape: cylinder, fill: #E7F5FF, stroke: #1971C2]
  tpl-external [stroke-dash: 4, opacity: 0.8]
}

gw:     "API Gateway" [like: tpl-svc]
pg:     "Postgres"    [like: tpl-store]
stripe: "Stripe"      [like: (tpl-svc, tpl-external)]
```

This is d2's `classes` block, minus the block, minus the keyword, minus the
separate lookup namespace. A template is an element like any other.

### 5.3 `src` — children from another file

```m2
aws: "AWS" [src: "./aws.m2"]

myapp -> rds              // rds was declared in aws.m2
```

The named file is parsed and its root becomes this element's children. Paths
resolve relative to the importing file, and cycles are an error. Attributes
declared alongside `src` apply to the container itself:

```m2
aws: "AWS" [src: "./aws.m2", fill: #F8F9FA, collapsed]
```

Because names are global, an imported file shares one namespace with everything
else — `src` groups the imported elements visually but does not qualify their
names. A file meant to be imported should therefore prefix what it declares.
Under `[strict]`, a collision between two files is an error; otherwise the two
declarations merge.

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

Endpoints are plain names, so an edge may be written anywhere — inside either
endpoint's container, inside a third one, or at the root. Where you put it
affects nothing but readability.

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
org-id -> orgs-pk [src-head: many, dst-head: one]
Dog -> Animal     [dst-head: triangle]           // UML inheritance
Wheel -- Car      [src-head: filled-diamond]     // UML composition
```

Head values: `none`, `arrow`, `triangle`, `diamond`, `filled-diamond`,
`circle`, `filled-circle`, `cross`, `one`, `many`, `zero-or-one`,
`one-or-many`, `zero-or-many`.

### 6.5 Endpoint labels

```m2
client -> server: "TCP" [src-label: "ephemeral", dst-label: ":443"]
```

---

## 7. Containers

A block makes an element a container. Containers group elements visually and
give the layout something to box and label — they do not scope names (§3.1).

```m2
vpc: "Production VPC" {
  public: "Public subnet" {
    lb: "Load balancer"
  }
  private: "Private subnet" {
    app: "App servers"
  }
}

lb -> app
internet -> vpc: "443"    // an edge may name a container as an endpoint
```

An edge to a container attaches to its boundary rather than to any child, and
edges may cross container boundaries freely, in either direction.

### 7.1 Adding to a container later

Restating a container with a block adds to it:

```m2
vpc: "Production VPC" { public { lb } }

vpc {                     // same container, more children
  private { app }
}
```

The same works for attributes, since redeclaration merges (§3.3). An element's
container is fixed by the declaration that first places it in one; a later
declaration elsewhere merges attributes but does not move it, and is an error
under `[strict]`.

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

Children of these kinds are elements, so their names are global like any other.
Members that would collide across two tables or two classes need distinguishing
names; a dotted convention (§3.1) is the usual answer.

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

A participant is a distinct element from whatever it depicts elsewhere in the
diagram, so if a service already exists as a node, its participant needs its own
name.

### 8.2 Table

Inside `kind: table`, each child is a column and its label is the column type.

```m2
users [kind: table] {
  user-id:    uuid        [key: primary]
  email:      text        [unique]
  user-org:   uuid        [key: foreign]
  created-at: timestamptz
}

orgs [kind: table] {
  org-id:   uuid [key: primary]
  org-name: text
}

user-org -> org-id [src-head: many, dst-head: one]
```

Columns are elements, so the foreign-key edge is written between two plain
names and lands on the right rows.

### 8.3 Class

Inside `kind: class`, each child is a member: an optional visibility sigil
(`+` public, `-` private, `#` protected, `~` package), a name, and a label
holding the type or return type.

```m2
Store [kind: class] {
  +Store.name:  string
  -Store.conn:  *sql.DB
  #Store.cache: map[string]Entry

  +Store.Save(e Entry): error
  +Store.Load(id string): (Entry, error)
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
attributes, and its contents share the one global namespace.

### 8.5 Notes and text

```m2
n1: """md
  **Caution:** this path is not idempotent.
""" [kind: note, near: enrich]

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

Because names are global and resolution is a single lookup, an edge may name an
element declared further down the file; the reference binds to whatever that
name ends up meaning.

### 9.2 Merging

Declaring an existing element merges attributes into it (§3.3). Declaring an
edge always creates a new edge, even between the same pair.

### 9.3 Precedence

When several sources set the same attribute on one element, the winner is the
highest of:

1. **Inline** — written in the element's own `[...]`
2. **Copied** — pulled in by `like`, later sources beating earlier
3. **Inherited** — from the enclosing container, for inheritable attributes only
   (`font`, `font-size`, `font-color`, `direction`)
4. **Default**

Within one level, later beats earlier.

### 9.4 Errors

These are errors, not warnings: a `like` or `src` cycle; an unknown attribute
key; a value of the wrong type; and, under `[strict]`, an unresolved reference
or a name declared into two different containers.

An unknown *value* for a known key is an error too — `[shape: octagon]` fails
rather than silently drawing a rectangle.

---

## 10. Attribute reference

### 10.1 Any element

| Key | Values | Notes |
| --- | --- | --- |
| `label` | string, `none` | same as the `:` form |
| `like` | name or list of names | copy attributes, §5.1 |
| `src` | string | children from a file, §5.3 |
| `kind` | see §8 | how children are read |
| `shape` | see below | outline |
| `icon` | string (path or URL) | |
| `link` | string | clickable target |
| `tooltip` | string | |
| `near` | name, or `top-left` … `bottom-right` | pin position |
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
| `like` | name or list of names — §5.1 |
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
| `strict` | flag — see §3.4 |
| `layout`, `pad`, `gap`, `font` | as above |

---

## 11. Grammar

EBNF. Whitespace between tokens is insignificant; a newline is a separator
(§2.2). Comments are stripped by the lexer.

```ebnf
diagram     = [ statement ] { separator [ statement ] } ;
separator   = ";" | newline ;

statement   = declaration | edge | attrs ;

declaration = name [ ":" label ] [ attrs ] [ block ] ;

edge        = endpoints { arrow endpoints } [ ":" label ] [ attrs ] ;
endpoints   = name | "(" name { "," name } [ "," ] ")" ;
arrow       = "->" | "<-" | "<->" | "--" ;

attrs       = "[" [ attr { "," attr } [ "," ] ] "]" ;
attr        = name ":" value
            | name                          (* flag, true  *)
            | "!" name ;                    (* flag, false *)

block       = "{" [ statement ] { separator [ statement ] } "}" ;

value       = number | percent | color | string | list | name ;
list        = "(" [ value { "," value } [ "," ] ] ")" ;

label       = string | free-text ;
name        = bare-name | string ;
```

Twelve productions, one of which is punctuation. `free-text` is everything to
the end of the statement, minus a trailing `attrs` group if one parses (§3.2).
`bare-name` and the string forms are defined in §2.4 and §2.5.

### 11.1 No reserved words, no operators on names

There are no keywords. Every attribute key — `shape`, `label`, `near`, `like`,
`style` — is available as an element name, because attribute keys are only ever
read inside `[...]`:

```m2
shape: "Shape"          // an element named shape
shape -> label          // an edge between two ordinary elements
shape [shape: circle]   // the element named shape, drawn as a circle
```

And with paths gone, a name has no internal structure for the parser to
interpret. `a.b.c` is one name, not three; the only tokens that can interrupt a
name are the delimiters in §2.4.

---

## 12. Differences from d2

Each of these is a deliberate departure, listed with the problem it solves.

| # | d2 | m2 | Why |
| --- | --- | --- | --- |
| 1 | Attributes are children: `x.shape: circle` | Attributes are bracketed: `x [shape: circle]` | Element names and attribute keys stop competing for one namespace. No reserved keys, no accidental node called `width`. |
| 2 | `x.style.fill: red` | `x [fill: red]` | One flat attribute namespace; nothing to remember about which keys live under `style`. |
| 3 | `#` starts a comment | `//` and `/* */` start comments | Frees `#` for hex colors, the most common literal in a diagram. |
| 4 | Hierarchical names, resolved by searching outward, with a failed search creating a node wherever it stopped | One global namespace; a name is a single lookup; `[strict]` turns a miss into an error | Removes paths, scope chains, shadowing, and anchors in one move. A reference means the same thing everywhere it is written. |
| 5 | `a.b.c` is three names | `a.b.c` is one name | With no path separator, `.` is an ordinary character — useful as a *convention* for uniqueness without being a *rule* the parser enforces. |
| 6 | `a -> b` only | `(a, b) -> (c, d)` cross products | Fan-in and fan-out without repeating yourself. |
| 7 | `classes` block, `vars` block, globs (`*.style.fill`), `@`/`...@` imports | `[like: name]` and `[src: "file"]` | Four mechanisms with three syntaxes become two attribute keys. A style bundle is just an element. |
| 8 | `shape: sequence_diagram`, `shape: sql_table`, `shape: class` | `kind: sequence`, `kind: table`, `kind: class` | Separates "what outline" from "how are children interpreted". |
| 9 | `\|md ... \|` text blocks | `"""md ... """` | Familiar from other languages; no delimiter collision with tables or code containing `\|`. |
| 10 | layers, scenarios, steps | `kind: layer` | One concept, expressed with machinery the language already has. |
| 11 | `(a -> b)[0]` positional edge references | none — style edges where you declare them | Positional indexes shift when you insert an edge, and the feature only existed to patch edges after the fact. |

### 12.1 What the flat namespace costs

Global names are the largest trade in the language, and it is not free. Nesting
no longer distinguishes `users.id` from `orgs.id`, so any structure that
naturally repeats member names — table columns, class fields, sequence
participants that mirror nodes elsewhere — needs a naming convention instead of
relying on containment. Two containers that both declare `cache` silently share
one element unless `[strict]` is on.

What it buys: no path grammar, no scope chain, no shadowing rules, no anchors,
no ambiguity about which `db` an edge means, and a reference that can be moved
between scopes without being rewritten. For diagrams — where the element count
is bounded by what a reader can take in — a flat namespace is usually the right
side of that trade.

---

## 13. Worked example

```m2
// checkout.m2 — order checkout, end to end
[title: "Checkout", direction: right, font: sans, strict]

tpl [hidden] {
  tpl-svc      [shape: rect, radius: 4, fill: #FFF, stroke: #4C6EF5]
  tpl-store    [shape: cylinder, fill: #E7F5FF, stroke: #1971C2]
  tpl-external [stroke-dash: 4, stroke: #868E96, opacity: 0.8]
  tpl-money    [stroke: #E03131, stroke-width: 3]
}

client: "Browser"   [shape: person]
cdn:    "CDN + WAF" [shape: hexagon, fill: #F1F3F5]

platform: "Platform" {
  [direction: down, fill: #F8F9FA, pad: 16]

  gw:      "API Gateway"     [like: tpl-svc]
  cart:    "Cart Service"    [like: tpl-svc]
  orders:  "Order Service"   [like: tpl-svc]
  billing: "Billing Service" [like: tpl-svc]

  bus: "Event Bus" [shape: queue, fill: #FFF3BF]

  gw -> cart
  gw -> orders
  orders -> billing: "authorize" [like: tpl-money]
  (cart, orders, billing) -> bus: "emit" [stroke-dash: 2, weight: 0.5]
}

data: "Data" {
  [direction: down]
  pg:    "Postgres" [like: tpl-store]
  redis: "Redis"    [like: tpl-store]
}

stripe:   "Stripe" [like: (tpl-svc, tpl-external)]
sendgrid: "Email"  [like: (tpl-svc, tpl-external)]

client -> cdn: "HTTPS"
cdn -> gw:     "mTLS" [stroke-width: 2]

cart    -> redis:    "session"
orders  -> pg:       "orders"
billing -> stripe:   "charge" [like: tpl-money]
bus     -> sendgrid: "receipt"

n1: """md
  **SLO:** p99 checkout < 800ms
  Stripe timeouts fall back to the async queue.
""" [kind: note, near: billing]

billing-detail: "Inside Billing" [kind: layer] {
  [direction: down]

  flow: "Authorize" [kind: sequence] {
    seq-orders:  "Order Service"
    seq-billing: "Billing Service"
    seq-stripe:  "Stripe"
    seq-pg:      "Postgres"

    seq-orders  -> seq-billing: "POST /authorize" [span]
    seq-billing -> seq-pg:      "insert attempt"
    seq-billing -> seq-stripe:  "PaymentIntent" [span]

    retry: "up to 3, backoff 2^n" [kind: fragment, op: loop] {
      seq-billing -> seq-stripe: "PaymentIntent"
    }

    seq-stripe  -> seq-billing: "requires_action" [stroke-dash: 3]
    seq-billing -> seq-orders:  "202 + redirect"
  }

  ledger [kind: table] {
    ledger-id:     uuid [key: primary]
    ledger-order:  uuid [key: foreign]
    ledger-amount: bigint
    ledger-status: text
  }
}
```

Two things to read off it. The payoff: every edge in the lower half is written
between two bare names — `cart -> redis`, `billing -> stripe` — with no regard
for which container either side sits in.

The price: the `seq-` and `ledger-` prefixes. The sequence chart depicts the
same four services as the main diagram, but its participants are separate
elements, so under one global namespace they need separate names. That is the
flat-namespace trade (§12.1) showing up in practice.
