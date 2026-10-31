// Command usage prints a Claude plan usage summary to the terminal.
package main

import (
	"fmt"
	"strings"
)

type meter struct {
	name   string
	resets string
	pct    int
}

func bar(pct, width int) string {
	n := pct * width / 100
	if pct > 0 && n == 0 {
		n = 1
	}
	return "\x1b[34m" + strings.Repeat("█", n) + "\x1b[90m" + strings.Repeat("░", width-n) + "\x1b[0m"
}

func main() {
	plan := "Pro"
	meters := []meter{
		{"Current session", "Resets at 12:30 PM", 9},
		{"This week", "Resets Tuesday 11:00 AM", 2},
	}

	fmt.Printf("\x1b[1mYour usage\x1b[0m  \x1b[90m%s\x1b[0m\n\n", plan)
	fmt.Printf("\x1b[1mFresh week. %d%% of your weekly limit used.\x1b[0m\n", meters[1].pct)
	fmt.Println(strings.Repeat("─", 72))
	for _, m := range meters {
		fmt.Printf("%-24s %s  %3d%% used\n", m.name, bar(m.pct, 36), m.pct)
		fmt.Printf("\x1b[90m%s\x1b[0m\n\n", m.resets)
	}
}
