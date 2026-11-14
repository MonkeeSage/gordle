package solverui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"gordle/internal/solver"
	"gordle/internal/words"
)

func newModel(w, h int) Model {
	return New(Options{Words: words.Answers()}).withSize(w, h)
}

func send(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func key(s string) tea.Msg {
	switch s {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "ctrl+s":
		return tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	case "ctrl+r":
		return tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

func keys(m Model, ks ...string) Model {
	for _, k := range ks {
		m = send(m, key(k))
	}
	return m
}

func TestTabOrder(t *testing.T) {
	m := newModel(80, 30)
	for want := 1; want <= numFocus; want++ {
		m = keys(m, "tab")
		if m.focus != want%numFocus {
			t.Fatalf("after %d tabs focus = %d", want, m.focus)
		}
	}
	m = keys(m, "shift+tab")
	if m.focus != focusSolve {
		t.Errorf("shift+tab from first box: focus = %d, want Solve", m.focus)
	}
	// Tab from the last correct box goes to the first misplaced box.
	m.focus = solver.Length - 1
	if m = keys(m, "tab"); m.focus != focusMisplaced {
		t.Errorf("tab from last correct box: focus = %d", m.focus)
	}
}

func TestArrowKeys(t *testing.T) {
	m := newModel(80, 30)
	m = keys(m, "left") // stays in the first box
	if m.focus != 0 {
		t.Fatalf("left at edge moved to %d", m.focus)
	}
	m = keys(m, "right", "right", "down") // correct[2] -> misplaced[2]
	if m.focus != focusMisplaced+2 {
		t.Fatalf("focus = %d, want misplaced[2]", m.focus)
	}
	m = keys(m, "down", "down")
	if m.focus != focusSolve {
		t.Fatalf("focus = %d, want Solve", m.focus)
	}
	m = keys(m, "up", "up", "up")
	if m.focus != 0 {
		t.Fatalf("up from Solve: focus = %d, want correct[0]", m.focus)
	}
}

func TestTyping(t *testing.T) {
	m := newModel(80, 30)
	m = keys(m, "c", "R") // a correct box holds one letter
	if m.c.Correct[0] != 'r' {
		t.Errorf("correct[0] = %q", m.c.Correct[0])
	}
	m.focus = focusMisplaced
	m = keys(m, "a", "e", "a", "i", "o", "u", "y") // no dupes, max 5
	if m.c.Misplaced[0] != "aeiou" {
		t.Errorf("misplaced[0] = %q", m.c.Misplaced[0])
	}
	m = keys(m, "backspace")
	if m.c.Misplaced[0] != "aeio" {
		t.Errorf("after backspace misplaced[0] = %q", m.c.Misplaced[0])
	}
	m.focus = focusExcluded
	m = keys(m, "x", "y", "x", "1", "z")
	if m.c.Excluded != "xyz" {
		t.Errorf("excluded = %q", m.c.Excluded)
	}
	m = keys(m, "delete")
	if m.c.Excluded != "" {
		t.Errorf("after delete excluded = %q", m.c.Excluded)
	}
	m.focus = focusSolve
	if m = keys(m, "q"); m.c.Excluded != "" || m.c.Misplaced[0] != "aeio" {
		t.Error("typing on the Solve button changed an input")
	}
}

func TestSolve(t *testing.T) {
	m := newModel(80, 30)
	m = keys(m, "tab", "r", "tab", "tab", "tab", "tab", "a") // R second, A misplaced first
	m = keys(m, "ctrl+s")
	if !m.solved || len(m.results) == 0 {
		t.Fatalf("no results after solve")
	}
	for _, w := range m.results {
		if w[1] != 'r' || w[0] == 'a' || !strings.Contains(w, "a") {
			t.Fatalf("result %q doesn't fit", w)
		}
	}
	screen := ansi.Strip(m.View().Content)
	if !strings.Contains(screen, strings.ToUpper(m.results[0])) || !strings.Contains(screen, "candidates") {
		t.Errorf("results not shown:\n%s", screen)
	}

	m = keys(m, "b")
	if !m.stale || !strings.Contains(m.status(), "solve again") {
		t.Errorf("editing after solve should mark results stale: %q", m.status())
	}

	// R can't be both correct and misplaced in the second position.
	m.focus = focusMisplaced + 1
	m = keys(m, "r", "ctrl+s")
	if len(m.results) != 0 || !strings.Contains(m.status(), "No words match") {
		t.Errorf("conflicting letters: results=%v status=%q", m.results, m.status())
	}

	m = keys(m, "ctrl+r")
	if m.c.Correct[1] != 0 || m.solved || m.width != 80 {
		t.Errorf("ctrl+r did not clear: %+v", m.c)
	}
}

func TestScroll(t *testing.T) {
	m := keys(newModel(80, 30), "ctrl+s") // no constraints: every answer
	rows := (len(m.results) + m.gridCols() - 1) / m.gridCols()
	for range 100 {
		m = keys(m, "pgdown")
	}
	if m.scroll != rows-m.resultRows() {
		t.Errorf("scroll = %d, want last page %d", m.scroll, rows-m.resultRows())
	}
	m = send(m, tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m.scroll != rows-m.resultRows()-1 {
		t.Errorf("wheel up: scroll = %d", m.scroll)
	}
	for range 500 {
		m = send(m, tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	}
	if m.scroll != 0 {
		t.Errorf("scroll = %d, want 0", m.scroll)
	}
}

var borderRe = regexp.MustCompile(`[│┃]`)

// clickBox clicks the left border of box n on the screen line containing
// label (border characters are part of a box's layer).
func clickBox(t *testing.T, m Model, label string, n int) tea.Msg {
	t.Helper()
	v := m.View()
	for y, line := range strings.Split(ansi.Strip(v.Content), "\n") {
		if !strings.Contains(line, label) {
			continue
		}
		locs := borderRe.FindAllStringIndex(line, -1)
		x := ansi.StringWidth(line[:locs[2*n][0]])
		cmd := v.OnMouse(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
		if cmd == nil {
			t.Fatalf("click on %s box %d hit nothing", label, n)
		}
		return cmd()
	}
	t.Fatalf("no line with %q", label)
	return nil
}

func TestClickToFocus(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 30}, {minW, minH}} {
		m := newModel(size.w, size.h)
		m = send(m, clickBox(t, m, "Misplaced", 3))
		if m.focus != focusMisplaced+3 {
			t.Errorf("%dx%d: focus = %d, want misplaced[3]", size.w, size.h, m.focus)
		}
		m = send(m, clickBox(t, m, "Correct", 4))
		if m.focus != 4 {
			t.Errorf("%dx%d: focus = %d, want correct[4]", size.w, size.h, m.focus)
		}
		m = send(m, clickBox(t, m, "Excluded", 0))
		if m.focus != focusExcluded {
			t.Errorf("%dx%d: focus = %d, want excluded", size.w, size.h, m.focus)
		}

		// The Solve button.
		v := m.View()
		for y, line := range strings.Split(ansi.Strip(v.Content), "\n") {
			if i := strings.Index(line, "Solve (ctrl+s)"); i >= 0 {
				cmd := v.OnMouse(tea.MouseClickMsg{X: ansi.StringWidth(line[:i]), Y: y, Button: tea.MouseLeft})
				m = send(m, cmd())
			}
		}
		if !m.solved {
			t.Errorf("%dx%d: clicking Solve did not solve", size.w, size.h)
		}
	}
}

func TestTooSmall(t *testing.T) {
	m := newModel(minW-1, minH)
	if !strings.Contains(m.View().Content, "Terminal too small") {
		t.Error("no too-small message")
	}
}
