// Command gordle is a Wordle-style word game for the terminal.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"gordle/internal/ui"
	"gordle/internal/words"
)

func main() {
	var (
		opts   ui.Options
		random bool
		gameID string
	)
	flag.BoolVar(&random, "random", false, "play a random word instead of today's puzzle")
	flag.StringVar(&gameID, "game", "", "replay a shared game by its ID, e.g. 317 or R-K7Q")
	flag.BoolVar(&opts.Hard, "hard", false, "hard mode: revealed hints must be used in later guesses")
	flag.BoolVar(&opts.Colorblind, "colorblind", false, "high-contrast orange/blue colours")
	flag.BoolVar(&opts.Cheat, "cheat", false, "casual mode: press ? to reveal up to 3 letters (disables sharing)")
	flag.BoolVar(&opts.NoMouse, "no-mouse", false, "disable mouse support")
	flag.Parse()

	switch {
	case random && gameID != "":
		fail(2, "use either -random or -game, not both")
	case gameID != "":
		p, err := words.Parse(gameID, time.Now())
		if err != nil {
			fail(2, err)
		}
		opts.Puzzle = p
	case random:
		opts.Puzzle = words.Random()
	}

	if _, err := tea.NewProgram(ui.New(opts)).Run(); err != nil {
		fail(1, err)
	}
}

func fail(code int, msg any) {
	fmt.Fprintln(os.Stderr, "gordle:", msg)
	os.Exit(code)
}
