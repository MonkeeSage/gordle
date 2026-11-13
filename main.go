// Command gordle is a Wordle-style word game for the terminal.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"gordle/internal/ui"
)

func main() {
	var opts ui.Options
	flag.BoolVar(&opts.Random, "random", false, "play a random word instead of today's puzzle")
	flag.BoolVar(&opts.Hard, "hard", false, "hard mode: revealed hints must be used in later guesses")
	flag.BoolVar(&opts.Colorblind, "colorblind", false, "high-contrast orange/blue colours")
	flag.BoolVar(&opts.NoMouse, "no-mouse", false, "disable mouse support")
	flag.Parse()

	if _, err := tea.NewProgram(ui.New(opts)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gordle:", err)
		os.Exit(1)
	}
}
