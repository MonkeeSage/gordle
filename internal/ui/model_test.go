package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"gordle/internal/game"
)

func newTestModel(w, h int, target string) Model {
	m := New(Options{Random: true})
	m.newGame(0, target)
	m.width, m.height = w, h
	return m
}

func send(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func typeWord(m Model, w string) Model {
	for _, c := range w {
		m = send(m, tea.KeyPressMsg{Code: c, Text: string(c)})
	}
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	for m.revealing() {
		m = send(m, revealMsg{})
	}
	return m
}

func TestPlayThrough(t *testing.T) {
	m := newTestModel(80, 40, "crane")

	m = typeWord(m, "zzzzz")
	if len(m.game.Guesses) != 0 || m.toast != "Not in word list" || m.shake == 0 {
		t.Fatalf("invalid word: guesses=%d toast=%q shake=%d", len(m.game.Guesses), m.toast, m.shake)
	}
	for range 5 {
		m = send(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	m = typeWord(m, "slate")
	if m.keys['s'-'a'] != game.Absent || m.keys['a'-'a'] != game.Correct {
		t.Errorf("keyboard not updated after reveal: %v", m.keys)
	}
	m = typeWord(m, "crane")
	m = send(m, toastExpireMsg{m.toastID}) // clear "Not in word list"
	if !m.over() || m.statusLine() != "Magnificent!" {
		t.Errorf("after win: over=%v status=%q", m.over(), m.statusLine())
	}
	if got := m.shareText(); !strings.HasPrefix(got, "Gordle (random) 2/6\n\n") {
		t.Errorf("share text = %q", got)
	}
}

// click renders m, finds label on the screen line matching lineSub and sends
// a left click on it through the view's mouse handler.
func click(t *testing.T, m Model, lineSub, label string) tea.Msg {
	t.Helper()
	v := m.View()
	for y, line := range strings.Split(ansi.Strip(v.Content), "\n") {
		if !strings.Contains(line, lineSub) {
			continue
		}
		i := strings.Index(line, label)
		if i < 0 {
			t.Fatalf("label %q not on line %q", label, line)
		}
		x := ansi.StringWidth(line[:i])
		cmd := v.OnMouse(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
		if cmd == nil {
			t.Fatalf("click on %q at %d,%d hit nothing", label, x, y)
		}
		return cmd()
	}
	t.Fatalf("no line containing %q", lineSub)
	return nil
}

func TestMouseHitTesting(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 40}, {50, 24}} {
		m := newTestModel(size.w, size.h, "crane")
		tests := []struct{ line, label, want string }{
			{"ENTER", "ENTER", "key:enter"},
			{"ENTER", "Z", "key:z"},
			{"ENTER", "⌫", "key:back"},
			{"ENTER", "M", "key:m"},
			{"Q   ", "P", "key:p"},
			{"A   ", "L", "key:l"},
		}
		for _, tt := range tests {
			got := click(t, m, tt.line, tt.label)
			if c, ok := got.(clickMsg); !ok || c.id != tt.want {
				t.Errorf("%dx%d: click %q = %#v, want %s", size.w, size.h, tt.label, got, tt.want)
			}
		}

		// Buttons appear once the game is over.
		for _, w := range []string{"slate", "pound", "might", "ducky", "befog", "jumbo"} {
			m = typeWord(m, w)
		}
		if got := click(t, m, "Share", "Share"); got != (clickMsg{"btn:share"}) {
			t.Errorf("%dx%d: share click = %#v", size.w, size.h, got)
		}
	}
}

func TestShareToastExpires(t *testing.T) {
	m := newTestModel(80, 40, "crane")
	m = typeWord(m, "crane")
	next, _ := m.action("btn:share")
	m = next.(Model)
	if m.toast == "" {
		t.Fatal("no toast after share")
	}
	m = send(m, toastExpireMsg{m.toastID})
	if m.toast != "" {
		t.Errorf("toast %q did not expire", m.toast)
	}
}
