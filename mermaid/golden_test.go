package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files in testdata/")

// renderPlain renders src at a fixed width and flattens the styled runs into
// plain text, so golden files capture the exact box-drawing art.
func renderPlain(src string, width int) string {
	lines := render(src, width)
	var b strings.Builder
	for _, line := range lines {
		for _, sp := range line {
			b.WriteString(sp.text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// goldenCases exercise every diagram type plus the routing, direction-flip, and
// edge-style paths that the substring tests do not lock down.
var goldenCases = []struct {
	name  string
	width int
	src   string
}{
	{"flowchart_td", 100, "flowchart TD\n  A[Start] --> B{OK?}\n  B -->|yes| C[Ship]\n  B -->|no| A\n"},
	{"flowchart_lr", 100, "graph LR\n  A --> B --> C\n  A --> C\n"},
	{"flowchart_bt", 80, "graph BT\n  A --> B --> C\n"},
	{"flowchart_rl", 80, "graph RL\n  A --> B --> C\n"},
	{"flowchart_styles", 100, "flowchart TD\n  A -.-> B\n  A ==> C\n  B --- C\n"},
	{"flowchart_heads", 100, "flowchart LR\n  A --o B\n  A --x C\n  A o--o D\n  A x--x E\n"},
	{"flowchart_selfloop", 100, "flowchart TD\n  A[Node] --> A\n  A -->|retry| A\n"},
	{"flowchart_backedge", 100, "flowchart TD\n  A --> B --> C --> D\n  A --> D\n  D --> A\n"},
	{"flowchart_crossing", 100, "flowchart TD\n  A --> C\n  A --> D\n  B --> C\n  B --> D\n"},
	{"subgraph_nested", 100, "flowchart TB\n  subgraph web[Web Tier]\n    LB --> S1\n    LB --> S2\n  end\n  subgraph db[Data]\n    DB1\n  end\n  S1 --> DB1\n  S2 --> DB1\n"},
	{"state", 100, "stateDiagram-v2\n  [*] --> Idle\n  Idle --> Running : start\n  Running --> Idle : stop\n  Running --> [*]\n"},
	{"state_choice", 100, "stateDiagram-v2\n  state pick <<choice>>\n  [*] --> pick\n  pick --> A : x>0\n  pick --> B : x<=0\n"},
	{"class", 100, "classDiagram\n  Animal <|-- Dog\n  Animal *-- Leg\n  Animal o-- Tail\n  Dog ..> Bone\n  Animal : +int age\n  Animal : +makeSound()\n  class Dog {\n    +String breed\n    +fetch()\n  }\n"},
	{"er", 100, "erDiagram\n  CUSTOMER ||--o{ ORDER : places\n  ORDER ||--|{ LINE_ITEM : contains\n"},
	{"er_attrs", 100, "erDiagram\n  CUSTOMER[\"Customer\"] ||--o{ ORDER : places\n  CUSTOMER {\n    string name\n    int id\n  }\n"},
	{"state_desc", 100, "stateDiagram-v2\n  state \"Running State\" as R\n  [*] --> R\n  R : does work\n  R --> Done\n"},
	{"sequence", 100, "sequenceDiagram\n  participant A as Alice\n  participant B as Bob\n  A->>B: Hello Bob\n  B-->>A: Hi Alice\n  A->>A: think\n"},
	{"sequence_notes", 100, "sequenceDiagram\n  autonumber\n  participant A as Alice\n  participant B as Bob\n  Note over A,B: Handshake\n  A->>B: Syn\n  B-->>A: Ack\n  loop retries\n    A->>B: Data\n  end\n  Note right of B: done\n"},
	{"fallback", 60, "pie title Pets\n  \"Dogs\" : 50\n  \"Cats\" : 30\n"},
	{"toowide", 12, "flowchart LR\n  A --> B --> C --> D --> E --> F --> G --> H\n"},
}

func TestGolden(t *testing.T) {
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderPlain(tc.src, tc.width)
			path := filepath.Join("testdata", tc.name+".golden")
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden (run `go test -run TestGolden -update` to create): %v", err)
			}
			if got != string(want) {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}
		})
	}
}
