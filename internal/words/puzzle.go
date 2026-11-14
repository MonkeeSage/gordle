package words

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

// Puzzle is a game's answer together with the ID used to share and replay it.
type Puzzle struct {
	Number int    // daily puzzle number; 0 for random games
	Code   string // random game code such as "R-K7Q"; empty for daily puzzles
	Word   string
}

// ID is how the puzzle is shown and shared: "#317" or "R-K7Q".
func (p Puzzle) ID() string {
	if p.Number > 0 {
		return fmt.Sprintf("#%d", p.Number)
	}
	return p.Code
}

// Daily returns the puzzle for the calendar day of t (in t's location).
// Everyone playing on the same date gets the same word.
func Daily(t time.Time) Puzzle {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return dailyNumber(int(day.Sub(epoch).Hours()/24) + 1)
}

func dailyNumber(n int) Puzzle {
	i := ((n-1)%len(answers) + len(answers)) % len(answers)
	return Puzzle{Number: n, Word: answers[dailyOrder[i]]}
}

// Random returns a random puzzle with a code that can replay it.
func Random() Puzzle {
	return randomIndex(rand.IntN(len(answers)))
}

func randomIndex(i int) Puzzle {
	return Puzzle{Code: encodeCode(i), Word: answers[i]}
}

var (
	ErrUnknownGame = errors.New("unknown game ID")
	ErrFutureGame  = errors.New("that daily puzzle isn't out yet")
)

// Parse finds the puzzle for a shared ID: a daily number ("317" or "#317")
// or a random game code ("R-K7Q", case-insensitive, dash optional). Daily
// puzzles after now's date are refused.
func Parse(id string, now time.Time) (Puzzle, error) {
	s := strings.ToUpper(strings.TrimSpace(id))
	s = strings.TrimPrefix(s, "#")
	if code, ok := strings.CutPrefix(s, "R"); ok {
		i, ok := decodeCode(strings.TrimPrefix(code, "-"))
		if !ok {
			return Puzzle{}, fmt.Errorf("%w %q", ErrUnknownGame, id)
		}
		return randomIndex(i), nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return Puzzle{}, fmt.Errorf("%w %q", ErrUnknownGame, id)
	}
	if n > Daily(now).Number {
		return Puzzle{}, fmt.Errorf("%w (#%d)", ErrFutureGame, n)
	}
	return dailyNumber(n), nil
}

// Random game codes are an answer's index pushed through an affine cipher
// modulo a prime just under 32^3, written as 3 Crockford base32 digits.
// The scrambling hides the word's alphabetical position, and only about 6%
// of codes map to an answer, so most typos are rejected instead of quietly
// loading a different word.
const (
	codeDigits = "0123456789ABCDEFGHJKMNPQRSTVWXYZ" // Crockford: no I, L, O, U
	codeLen    = 3
	codeMod    = 32749 // prime < 32^3
	codeMul    = 7919
	codeAdd    = 12345
)

var codeMulInv = modInverse(codeMul, codeMod)

func encodeCode(i int) string {
	v := (i*codeMul + codeAdd) % codeMod
	b := make([]byte, codeLen)
	for k := codeLen - 1; k >= 0; k-- {
		b[k] = codeDigits[v%32]
		v /= 32
	}
	return "R-" + string(b)
}

// decodeCode turns the 3 digits after "R-" back into an answer index.
func decodeCode(s string) (int, bool) {
	if len(s) != codeLen {
		return 0, false
	}
	v := 0
	for _, c := range s {
		// Crockford decoding accepts look-alikes for the missing letters.
		switch c {
		case 'O':
			c = '0'
		case 'I', 'L':
			c = '1'
		}
		d := strings.IndexRune(codeDigits, c)
		if d < 0 {
			return 0, false
		}
		v = v*32 + d
	}
	if v >= codeMod {
		return 0, false
	}
	i := (v - codeAdd%codeMod + codeMod) % codeMod * codeMulInv % codeMod
	if i >= len(answers) {
		return 0, false
	}
	return i, true
}

// modInverse returns x such that a*x ≡ 1 (mod m), for a coprime to m.
func modInverse(a, m int) int {
	t, newT, r, newR := 0, 1, m, a
	for newR != 0 {
		q := r / newR
		t, newT = newT, t-q*newT
		r, newR = newR, r-q*newR
	}
	return (t + m) % m
}
