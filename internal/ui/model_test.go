package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"gordle/internal/game"
	"gordle/internal/words"
)

func newTestModel(w, h int, target string) Model {
	return newTestModelOpts(w, h, target, Options{Random: true})
}

func newTestModelOpts(w, h int, target string, opts Options) Model {
	m := New(opts)
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

func typeLetters(m Model, w string) Model {
	for _, c := range w {
		m = send(m, tea.KeyPressMsg{Code: c, Text: string(c)})
	}
	return m
}

func typeWord(m Model, w string) Model {
	m = typeLetters(m, w)
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	for m.revealing() {
		m = send(m, revealMsg{m.gen})
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

func TestShare(t *testing.T) {
	m := newTestModel(80, 40, "crane")
	var copied string
	m.copyText = func(s string) error { copied = s; return nil }
	m = typeWord(m, "slate")
	m = typeWord(m, "crane")

	_, cmd := m.action("btn:share")
	if cmd == nil {
		t.Fatal("share returned no command")
	}
	// Run the batch: one OSC 52 command and one native copy.
	var copiedResult tea.Msg
	for _, c := range cmd().(tea.BatchMsg) {
		if msg, ok := c().(copiedMsg); ok {
			copiedResult = msg
		}
	}
	want := "Gordle (random) 2/6\n\n⬛⬛🟩⬛🟩\n🟩🟩🟩🟩🟩"
	if copied != want {
		t.Errorf("native clipboard got %q, want %q", copied, want)
	}
	if copiedResult == nil {
		t.Fatal("no copiedMsg from share")
	}

	m = send(m, copiedResult)
	if m.toast != "Copied result to clipboard" {
		t.Errorf("toast = %q", m.toast)
	}
	m = send(m, toastExpireMsg{m.toastID})
	if m.toast != "" {
		t.Errorf("toast %q did not expire", m.toast)
	}
}

func TestCopiedToast(t *testing.T) {
	fail := errors.New("no tool")
	if got := copiedToast(fail, true); !strings.Contains(got, "set-clipboard on") {
		t.Errorf("tmux failure toast = %q", got)
	}
	if got := copiedToast(fail, false); !strings.Contains(got, "OSC 52") {
		t.Errorf("non-tmux failure toast = %q", got)
	}
}

// screenHas reports whether the rendered screen contains s.
func screenHas(m Model, s string) bool {
	return strings.Contains(ansi.Strip(m.View().Content), s)
}

func TestCheatMode(t *testing.T) {
	m := newTestModelOpts(80, 40, "crane", Options{Random: true, Cheat: true})
	m.pick = func(int) int { return 0 } // always the first candidate position
	backspace := tea.KeyPressMsg{Code: tea.KeyBackspace}

	// A hint replaces whatever was typed in its slot and turns the key green.
	m = typeLetters(m, "xy")
	m = send(m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.slots != [5]byte{'c', 'y'} || m.keys['c'-'a'] != game.Correct {
		t.Fatalf("after hint: slots=%q keys[c]=%v", m.slots, m.keys['c'-'a'])
	}
	if !screenHas(m, "? hint") || !screenHas(m, "cheat") {
		t.Error("cheat mode help or title missing")
	}

	// Backspace never removes a hinted letter; typing fills around it.
	m = send(m, backspace, backspace, backspace)
	if got := m.inputWord(); got != "c" {
		t.Fatalf("after backspacing input = %q, want c", got)
	}
	m = typeWord(m, "ramp")
	if got := m.game.Guesses[0].Word; got != "cramp" {
		t.Fatalf("submitted %q, want cramp", got)
	}
	if m.slots != [5]byte{'c'} {
		t.Fatalf("hint not carried to the next row: %q", m.slots)
	}

	// C, R and A are now known, so hints go to N then E; then the limit.
	m = send(m, clickMsg{"btn:hint"}, clickMsg{"btn:hint"})
	if m.slots != [5]byte{'c', 0, 0, 'n', 'e'} {
		t.Fatalf("after 3 hints: slots=%q", m.slots)
	}
	m = send(m, clickMsg{"btn:hint"})
	if len(m.game.Hints) != game.MaxHints || m.toast != noHintsLeft {
		t.Errorf("4th hint: hints=%d toast=%q", len(m.game.Hints), m.toast)
	}

	// Win, then check sharing is unavailable.
	m = typeWord(m, "ra")
	if m.game.Status() != game.Won {
		t.Fatalf("status = %v after crane", m.game.Status())
	}
	if screenHas(m, "Share") || !screenHas(m, "New game") {
		t.Error("cheat mode end screen should offer New game but not Share")
	}
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"}); cmd != nil {
		t.Error("s still shares in cheat mode")
	}
}

func TestHintButtonClick(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 40}, {50, 24}} {
		m := newTestModelOpts(size.w, size.h, "crane", Options{Random: true, Cheat: true})
		v := m.View()
		lines := strings.Split(ansi.Strip(v.Content), "\n")
		found := false
		for y, line := range lines {
			if x := strings.Index(line, "?"); x >= 0 && !strings.Contains(line, "hint") {
				cmd := v.OnMouse(tea.MouseClickMsg{X: ansi.StringWidth(line[:x]), Y: y, Button: tea.MouseLeft})
				if cmd == nil || cmd() != (clickMsg{"btn:hint"}) {
					t.Errorf("%dx%d: clicking ? did not hit the hint button", size.w, size.h)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%dx%d: no ? button on screen", size.w, size.h)
		}
	}
	if screenHas(newTestModel(80, 40, "crane"), "?") {
		t.Error("? button shown without cheat mode")
	}
}

func TestRestart(t *testing.T) {
	m := newTestModel(80, 40, "crane")
	m = typeLetters(m, "slate")
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEnter}) // reveal in progress
	gen := m.gen
	m = send(m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if m.gen == gen || len(m.game.Guesses) != 0 || m.revealing() {
		t.Fatalf("ctrl+r did not start a fresh game")
	}
	// A tick from the old game's reveal must not affect the new one.
	m = typeLetters(m, "cra")
	m = send(m, revealMsg{gen})
	if m.revealed != words.Length || m.inputWord() != "cra" {
		t.Errorf("stale reveal tick changed state: revealed=%d input=%q", m.revealed, m.inputWord())
	}
	if !screenHas(m, "ctrl+r restart") {
		t.Error("help line missing ctrl+r")
	}

	// Plain r types a letter during play, but restarts after the game.
	m = typeLetters(m, "r")
	if m.inputWord() != "crar" {
		t.Errorf("r during play: input = %q", m.inputWord())
	}
	m = newTestModel(80, 40, "crane")
	m = typeWord(m, "crane")
	old := m.game
	m = send(m, tea.KeyPressMsg{Code: 'r', Text: "r"})
	if m.game == old {
		t.Error("r after game over did not start a new game")
	}
}

func TestHintKeyColourWithRepeatedLetter(t *testing.T) {
	m := newTestModelOpts(80, 40, "truth", Options{Random: true, Cheat: true})
	m.pick = func(int) int { return 0 }
	m = send(m, clickMsg{"btn:hint"}, clickMsg{"btn:hint"}, clickMsg{"btn:hint"}) // T, R, U
	if m.slots != [5]byte{'t', 'r', 'u'} {
		t.Fatalf("slots = %q", m.slots)
	}
	if got := m.keys['t'-'a']; got != game.Present {
		t.Errorf("T key = %v, want Present: the second T is still hidden", got)
	}
	if got := m.keys['r'-'a']; got != game.Correct {
		t.Errorf("R key = %v, want Correct", got)
	}
}
