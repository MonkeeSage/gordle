// Package game implements the rules of Wordle, independent of any UI.
package game

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Mark is the feedback for one letter of a guess.
type Mark uint8

const (
	Unknown Mark = iota // not guessed yet
	Absent              // letter is not in the word
	Present             // letter is in the word, but elsewhere
	Correct             // letter is in this position
)

// Score returns the marks for guess against target. Both must be lowercase
// ASCII of equal length.
//
// Repeated letters are handled like Wordle: greens are assigned first, then
// yellows only while unmatched copies of the letter remain in the target.
// Guessing "speed" for "abide" marks one E yellow, not two.
func Score(guess, target string) []Mark {
	marks := make([]Mark, len(guess))
	var remaining [26]int
	for i := range len(target) {
		if guess[i] == target[i] {
			marks[i] = Correct
		} else {
			remaining[target[i]-'a']++
		}
	}
	for i := range len(guess) {
		if marks[i] == Correct {
			continue
		}
		if c := guess[i] - 'a'; remaining[c] > 0 {
			marks[i] = Present
			remaining[c]--
		} else {
			marks[i] = Absent
		}
	}
	return marks
}

// Guess is a submitted word and its marks.
type Guess struct {
	Word  string
	Marks []Mark
}

// Status is the state of a game.
type Status int

const (
	Playing Status = iota
	Won
	Lost
)

var (
	ErrGameOver  = errors.New("game is over")
	ErrTooShort  = errors.New("not enough letters")
	ErrNotInList = errors.New("not in word list")
	ErrNoHints   = errors.New("no hints left")
)

// MaxHints is how many letters Hint may reveal in one game.
const MaxHints = 3

// HardModeError explains which hard-mode rule a guess broke.
type HardModeError struct{ msg string }

func (e *HardModeError) Error() string { return e.msg }

// Game is a single round.
type Game struct {
	Target     string
	MaxGuesses int
	Hard       bool
	Guesses    []Guess
	Hints      []int // positions revealed by Hint, in order

	isValid func(string) bool
}

// New starts a game. isValid decides which words are accepted as guesses.
func New(target string, maxGuesses int, hard bool, isValid func(string) bool) *Game {
	return &Game{Target: target, MaxGuesses: maxGuesses, Hard: hard, isValid: isValid}
}

// Status reports whether the game is still in progress.
func (g *Game) Status() Status {
	if n := len(g.Guesses); n > 0 && g.Guesses[n-1].Word == g.Target {
		return Won
	}
	if len(g.Guesses) >= g.MaxGuesses {
		return Lost
	}
	return Playing
}

// Submit scores word and records it as the next guess.
func (g *Game) Submit(word string) (Guess, error) {
	word = strings.ToLower(word)
	switch {
	case g.Status() != Playing:
		return Guess{}, ErrGameOver
	case len(word) != len(g.Target):
		return Guess{}, ErrTooShort
	case !g.isValid(word):
		return Guess{}, ErrNotInList
	}
	if g.Hard {
		if err := g.checkHard(word); err != nil {
			return Guess{}, err
		}
	}
	gs := Guess{Word: word, Marks: Score(word, g.Target)}
	g.Guesses = append(g.Guesses, gs)
	return gs, nil
}

// checkHard enforces hard mode: green letters must stay in place and yellow
// letters must be reused (as many times as they were revealed).
func (g *Game) checkHard(word string) error {
	for _, prev := range g.Guesses {
		var need [26]int
		for i, m := range prev.Marks {
			c := prev.Word[i]
			switch m {
			case Correct:
				if word[i] != c {
					return &HardModeError{fmt.Sprintf("%s letter must be %c", ordinal(i+1), c-'a'+'A')}
				}
				need[c-'a']++
			case Present:
				need[c-'a']++
			}
		}
		for c, n := range need {
			if strings.Count(word, string(rune('a'+c))) < n {
				return &HardModeError{fmt.Sprintf("guess must contain %c", 'A'+c)}
			}
		}
	}
	return nil
}

// Hint reveals the target letter at a random position, preferring positions
// no guess has turned green yet. pick(n) must return a number in [0, n).
// It returns the revealed position, or ErrNoHints after MaxHints reveals.
func (g *Game) Hint(pick func(n int) int) (int, error) {
	if g.Status() != Playing {
		return 0, ErrGameOver
	}
	if len(g.Hints) >= MaxHints {
		return 0, ErrNoHints
	}
	var unsolved, unhinted []int
	for i := range len(g.Target) {
		if g.Hinted(i) {
			continue
		}
		unhinted = append(unhinted, i)
		if !g.solved(i) {
			unsolved = append(unsolved, i)
		}
	}
	candidates := unsolved
	if len(candidates) == 0 {
		candidates = unhinted
	}
	i := candidates[pick(len(candidates))]
	g.Hints = append(g.Hints, i)
	return i, nil
}

// Hinted reports whether position i was revealed by Hint.
func (g *Game) Hinted(i int) bool { return slices.Contains(g.Hints, i) }

// solved reports whether any guess has the correct letter at position i.
func (g *Game) solved(i int) bool {
	for _, gs := range g.Guesses {
		if gs.Marks[i] == Correct {
			return true
		}
	}
	return false
}

// Letters returns the best mark known for each letter a-z, from guesses and
// hints, for colouring a keyboard.
//
// A hinted letter is only Correct once every occurrence of it in the target
// is hinted or solved; until then it is Present, so hinting the first T of
// "truth" doesn't hide that there's another T to find.
func (g *Game) Letters() [26]Mark {
	var best [26]Mark
	for _, gs := range g.Guesses {
		for i, m := range gs.Marks {
			c := gs.Word[i] - 'a'
			best[c] = max(best[c], m)
		}
	}
	for _, i := range g.Hints {
		c := g.Target[i]
		m := Correct
		for j := range len(g.Target) {
			if g.Target[j] == c && !g.Hinted(j) && !g.solved(j) {
				m = Present
				break
			}
		}
		best[c-'a'] = max(best[c-'a'], m)
	}
	return best
}

// Share renders the guesses as an emoji grid.
func (g *Game) Share() string {
	var b strings.Builder
	for _, gs := range g.Guesses {
		for _, m := range gs.Marks {
			switch m {
			case Correct:
				b.WriteString("🟩")
			case Present:
				b.WriteString("🟨")
			default:
				b.WriteString("⬛")
			}
		}
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	}
	return fmt.Sprintf("%dth", n)
}
