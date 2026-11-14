// Command gordle-solver lists the words that fit what you know about a
// gordle (or Wordle) answer.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"gordle/internal/solverui"
	"gordle/internal/words"
)

func main() {
	opts := solverui.Options{Words: words.Answers()}
	flag.BoolVar(&opts.Colorblind, "colorblind", false, "high-contrast orange/blue colours")
	flag.BoolVar(&opts.NoMouse, "no-mouse", false, "disable mouse support")
	flag.Parse()

	if _, err := tea.NewProgram(solverui.New(opts)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gordle-solver:", err)
		os.Exit(1)
	}
}
