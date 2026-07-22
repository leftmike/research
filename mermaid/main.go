package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point: it parses args, reads the given files (or
// stdin), renders every Mermaid block, and writes to stdout. It returns the
// process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mermaid", flag.ContinueOnError)
	fs.SetOutput(stderr)
	width := fs.Int("w", 0, "max render width in columns (0 = auto-detect, fallback 100)")
	colorFl := fs.String("color", "auto", "colorize output: auto, always, or never")
	rawFl := fs.Bool("raw", false, "treat entire input as one diagram, ignoring ``` fences")
	htmlFl := fs.Bool("html", false, "emit a self-contained HTML page instead of terminal text")
	titleFl := fs.String("title", "Mermaid diagrams", "page title for -html output")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: mermaid [-w cols] [-color auto|always|never] [-html] [-title t] [-raw] [file ...]\n\n")
		fmt.Fprintf(stderr, "Renders Mermaid diagrams as Unicode terminal art (or a self-contained HTML\n")
		fmt.Fprintf(stderr, "page with -html). Reads ```mermaid fenced code blocks from each file (or\n")
		fmt.Fprintf(stderr, "stdin); with -raw the whole input is one diagram.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// A terminal reports its width and enables color under -color=auto; a piped
	// or in-memory writer does neither.
	termW, isTTY := 0, false
	if f, ok := stdout.(*os.File); ok {
		termW, isTTY = terminalWidth(f.Fd())
	}

	maxWidth := *width
	if maxWidth <= 0 {
		switch {
		case *htmlFl:
			// An HTML page scrolls horizontally, so don't impose a width limit
			// (which would trigger the "too wide" fallback) unless -w was given.
			maxWidth = 0
		case columnsEnv() > 0:
			maxWidth = columnsEnv()
		case isTTY:
			maxWidth = termW
		default:
			maxWidth = 100
		}
	}

	color := false
	switch *colorFl {
	case "always":
		color = true
	case "never":
		color = false
	default: // auto
		color = isTTY
	}

	var docs []string
	if fs.NArg() == 0 {
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintln(stderr, "mermaid:", err)
			return 1
		}
		docs = append(docs, string(data))
	} else {
		for _, name := range fs.Args() {
			data, err := os.ReadFile(name)
			if err != nil {
				fmt.Fprintln(stderr, "mermaid:", err)
				return 1
			}
			docs = append(docs, string(data))
		}
	}

	out := bufio.NewWriter(stdout)
	defer out.Flush()

	var blocks []string
	for _, doc := range docs {
		if *rawFl {
			blocks = append(blocks, doc)
		} else {
			b := extractMermaidBlocks(doc)
			if len(b) == 0 {
				// No fences found: render the whole document as one diagram.
				blocks = append(blocks, doc)
			} else {
				blocks = append(blocks, b...)
			}
		}
	}

	if *htmlFl {
		var diagrams [][]span2D
		for _, src := range blocks {
			if lines := render(src, maxWidth); lines != nil {
				diagrams = append(diagrams, lines)
			}
		}
		if err := writeHTML(out, *titleFl, diagrams); err != nil {
			fmt.Fprintln(stderr, "mermaid:", err)
			return 1
		}
		return 0
	}

	rendered := 0
	for _, src := range blocks {
		lines := render(src, maxWidth)
		if lines == nil {
			continue
		}
		if rendered > 0 {
			fmt.Fprintln(out)
		}
		writeLines(out, lines, color)
		rendered++
	}
	return 0
}

// extractMermaidBlocks returns the contents of every ```mermaid fenced block.
func extractMermaidBlocks(doc string) []string {
	var blocks []string
	sc := bufio.NewScanner(strings.NewReader(doc))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	inBlock := false
	var cur strings.Builder
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if info, ok := fenceInfo(trimmed); ok && strings.EqualFold(info, "mermaid") {
				inBlock = true
				cur.Reset()
			}
			continue
		}
		if _, ok := fenceInfo(trimmed); ok && fenceClose(trimmed) {
			blocks = append(blocks, cur.String())
			inBlock = false
			continue
		}
		cur.WriteString(line)
		cur.WriteByte('\n')
	}
	if inBlock && cur.Len() > 0 {
		blocks = append(blocks, cur.String())
	}
	return blocks
}

// fenceInfo reports whether a trimmed line opens/closes a ``` or ~~~ fence and
// returns the trailing info string (language).
func fenceInfo(trimmed string) (string, bool) {
	for _, mark := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, mark) {
			return strings.TrimSpace(trimmed[len(mark):]), true
		}
	}
	return "", false
}

// fenceClose reports whether a trimmed fence line is a bare closing fence.
func fenceClose(trimmed string) bool {
	info, _ := fenceInfo(trimmed)
	return info == ""
}

func writeLines(w *bufio.Writer, lines []span2D, color bool) {
	for _, line := range lines {
		for _, sp := range line {
			if color {
				if code := ansiFor(sp.cls); code != "" {
					w.WriteString(code)
					w.WriteString(sp.text)
					w.WriteString("\x1b[0m")
					continue
				}
			}
			w.WriteString(sp.text)
		}
		w.WriteByte('\n')
	}
}

func ansiFor(cls int) string {
	switch cls {
	case clsBorder:
		return "\x1b[90m" // bright black / gray
	case clsText:
		return "\x1b[1m" // bold
	case clsEdge:
		return "\x1b[36m" // cyan
	case clsEdgeLabel:
		return "\x1b[33m" // yellow
	}
	return ""
}

type winsize struct {
	row, col, xpixel, ypixel uint16
}

func columnsEnv() int {
	if env := os.Getenv("COLUMNS"); env != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(env)); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// terminalWidth returns the column count for fd, and whether fd is a terminal.
func terminalWidth(fd uintptr) (int, bool) {
	ws := &winsize{}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, tiocgwinsz, uintptr(unsafe.Pointer(ws)))
	if errno != 0 || ws.col == 0 {
		return 0, false
	}
	return int(ws.col), true
}
