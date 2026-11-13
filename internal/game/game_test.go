package game

import (
	"errors"
	"testing"
)

// marks parses "gy.." style strings: g = Correct, y = Present, . = Absent.
func marks(s string) []Mark {
	ms := make([]Mark, len(s))
	for i, c := range s {
		switch c {
		case 'g':
			ms[i] = Correct
		case 'y':
			ms[i] = Present
		default:
			ms[i] = Absent
		}
	}
	return ms
}

func TestScore(t *testing.T) {
	tests := []struct {
		guess, target, want string
	}{
		{"crane", "crane", "ggggg"},
		{"slate", "crane", "..g.g"},
		{"bring", "crane", ".g.g."},
		{"speed", "abide", "..y.y"}, // only one E in target
		{"eerie", "speed", "yy..."}, // two Es in target, guess has three
		{"geese", "eerie", ".gy.g"}, // green E consumes one copy first
		{"lolly", "hello", ".ygg."}, // both target Ls used by greens
		{"allee", "eagle", "yy.yg"}, // second L has no copy left
		{"mamma", "maxim", "ggy.."}, // only one unmatched M remains
		{"aaaaa", "abcde", "g...."},
	}
	for _, tt := range tests {
		got := Score(tt.guess, tt.target)
		want := marks(tt.want)
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("Score(%q, %q) = %v, want %v", tt.guess, tt.target, got, want)
				break
			}
		}
	}
}

func allValid(string) bool { return true }

func TestGameWinAndLose(t *testing.T) {
	g := New("crane", 2, false, allValid)
	if _, err := g.Submit("slate"); err != nil {
		t.Fatal(err)
	}
	if g.Status() != Playing {
		t.Fatalf("status = %v, want Playing", g.Status())
	}
	if _, err := g.Submit("CRANE"); err != nil {
		t.Fatal(err)
	}
	if g.Status() != Won {
		t.Fatalf("status = %v, want Won", g.Status())
	}
	if _, err := g.Submit("crane"); !errors.Is(err, ErrGameOver) {
		t.Fatalf("submit after win: err = %v, want ErrGameOver", err)
	}

	g = New("crane", 1, false, allValid)
	g.Submit("slate")
	if g.Status() != Lost {
		t.Fatalf("status = %v, want Lost", g.Status())
	}
}

func TestSubmitValidation(t *testing.T) {
	g := New("crane", 6, false, func(w string) bool { return w != "xxxxx" })
	if _, err := g.Submit("cran"); !errors.Is(err, ErrTooShort) {
		t.Errorf("short word: err = %v", err)
	}
	if _, err := g.Submit("xxxxx"); !errors.Is(err, ErrNotInList) {
		t.Errorf("unknown word: err = %v", err)
	}
	if len(g.Guesses) != 0 {
		t.Errorf("rejected guesses were recorded")
	}
}

func TestHardMode(t *testing.T) {
	g := New("crane", 6, true, allValid)
	g.Submit("trace") // T absent; R, A, E green; C yellow
	tests := []struct {
		word string
		ok   bool
	}{
		{"brace", true},
		{"crate", true},
		{"crane", true},
		{"grade", false}, // no C
		{"frame", false}, // no C
		{"cramp", false}, // 5th letter must be E
	}
	for _, tt := range tests {
		err := g.checkHard(tt.word)
		var hm *HardModeError
		if tt.ok && err != nil {
			t.Errorf("%s: unexpected error %v", tt.word, err)
		}
		if !tt.ok && !errors.As(err, &hm) {
			t.Errorf("%s: err = %v, want HardModeError", tt.word, err)
		}
	}
}

func TestLetters(t *testing.T) {
	g := New("crane", 6, false, allValid)
	g.Submit("react") // r y, e y, a g, c y, t .
	g.Submit("cramp") // c g, r g, a g, m ., p .
	l := g.Letters()
	check := map[byte]Mark{'c': Correct, 'r': Correct, 'a': Correct, 'e': Present, 't': Absent, 'z': Unknown}
	for c, want := range check {
		if got := l[c-'a']; got != want {
			t.Errorf("letter %c = %v, want %v", c, got, want)
		}
	}
}
