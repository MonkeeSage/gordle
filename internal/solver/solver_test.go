package solver

import (
	"slices"
	"testing"

	"gordle/internal/game"
	"gordle/internal/words"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name string
		c    Constraints
		yes  []string
		no   []string
	}{
		{
			name: "correct letters",
			c:    Constraints{Correct: [5]byte{'c', 0, 0, 0, 'e'}},
			yes:  []string{"crane", "chase"},
			no:   []string{"brace", "crank"},
		},
		{
			name: "misplaced letters",
			c:    Constraints{Misplaced: [5]string{"a", "", "", "", "rt"}},
			yes:  []string{"trial", "tiara"},
			no:   []string{"after", "stair", "cloud"}, // A at 0, R at 4, no A/R/T
		},
		{
			name: "excluded letters",
			c:    Constraints{Excluded: "xyz"},
			yes:  []string{"crane"},
			no:   []string{"pizza", "proxy"},
		},
		{
			name: "gray copy of a green letter means exactly one",
			c:    Constraints{Correct: [5]byte{0, 0, 0, 0, 'e'}, Excluded: "e"},
			yes:  []string{"crane"},
			no:   []string{"geese", "eerie", "spent"},
		},
		{
			name: "gray copy of a yellow letter means exactly one",
			c:    Constraints{Misplaced: [5]string{"", "", "", "", "e"}, Excluded: "e"},
			yes:  []string{"react"},
			no:   []string{"eerie", "crane"},
		},
		{
			// From one guess that's two Ls, from two guesses it may be one,
			// so both are possible, but not three.
			name: "yellow and green of the same letter are one or two copies",
			c:    Constraints{Correct: [5]byte{0, 'l'}, Misplaced: [5]string{"l"}, Excluded: "l"},
			yes:  []string{"glare", "allay"},
			no:   []string{"ollal"}, // made up: three Ls
		},
		{
			name: "green, yellow and gray in one guess (geese for agree)",
			c:    Constraints{Correct: [5]byte{0, 0, 0, 0, 'e'}, Misplaced: [5]string{"g", "e"}, Excluded: "e"},
			yes:  []string{"agree"},
			no:   []string{"eerie"},
		},
	}
	for _, tt := range tests {
		for _, w := range tt.yes {
			if !tt.c.Match(w) {
				t.Errorf("%s: %q should match", tt.name, w)
			}
		}
		for _, w := range tt.no {
			if tt.c.Match(w) {
				t.Errorf("%s: %q should not match", tt.name, w)
			}
		}
	}
}

// TestNeverRulesOutTheAnswer plays the solver against real Wordle feedback:
// constraints built from game.Score must always keep the target.
func TestNeverRulesOutTheAnswer(t *testing.T) {
	answers := words.Answers()
	guesses := []string{"crane", "spilt", "humor", "geese", "llama", "eerie"}
	for _, target := range answers {
		var c Constraints
		for _, g := range guesses {
			addFeedback(&c, g, game.Score(g, target))
			if !c.Match(target) {
				t.Fatalf("target %q ruled out after %q: %+v", target, g, c)
			}
		}
		if got := Solve(c, answers); !slices.Contains(got, target) {
			t.Fatalf("Solve dropped %q", target)
		}
	}
}

// addFeedback records one guess's marks the way a player fills in the solver.
func addFeedback(c *Constraints, guess string, marks []game.Mark) {
	for i, m := range marks {
		l := guess[i]
		switch m {
		case game.Correct:
			c.Correct[i] = l
		case game.Present:
			if !containsByte(c.Misplaced[i], l) {
				c.Misplaced[i] += string(l)
			}
		case game.Absent:
			if !containsByte(c.Excluded, l) {
				c.Excluded += string(l)
			}
		}
	}
}

func containsByte(s string, b byte) bool { return slices.Contains([]byte(s), b) }

func TestSolveRanking(t *testing.T) {
	got := Solve(Constraints{Correct: [5]byte{0, 'i', 'g', 'h', 't'}}, []string{"might", "fight", "light", "tight", "crane"})
	if len(got) != 4 || slices.Contains(got, "crane") {
		t.Fatalf("Solve = %v", got)
	}
	// "tight" has only 4 distinct letters, so it ranks last.
	if got[3] != "tight" {
		t.Errorf("Solve = %v, want tight last", got)
	}
	if !slices.IsSorted(got[:3]) {
		t.Errorf("ties not alphabetical: %v", got)
	}
}
