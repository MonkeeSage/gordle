package ui

import (
	"charm.land/lipgloss/v2"

	"gordle/internal/game"
	"gordle/internal/theme"
)

// sizing holds the cell dimensions for one of the two layouts. Large tiles
// are 5x3 cells, compact ones 3x1 so the game fits a 24-line terminal.
type sizing struct {
	large            bool
	tileW, tileH     int
	boardRowGap      int
	keyW, keyH       int
	wideKeyW         int
	keyGap, kbRowGap int
}

var (
	compact = sizing{tileW: 3, tileH: 1, boardRowGap: 1, keyW: 3, keyH: 1, wideKeyW: 7, keyGap: 1, kbRowGap: 1}
	large   = sizing{large: true, tileW: 5, tileH: 3, boardRowGap: 0, keyW: 5, keyH: 3, wideKeyW: 9, keyGap: 1, kbRowGap: 1}
)

func (s sizing) boardW() int { return 5*s.tileW + 4 }
func (s sizing) boardH() int { return 6*s.tileH + 5*s.boardRowGap }
func (s sizing) kbH() int    { return 3*s.keyH + 2*s.kbRowGap }

type tileState int

const (
	tileEmpty tileState = iota
	tileTyped
	tileRevealed
)

func (m Model) tile(letter byte, state tileState, mark game.Mark) string {
	s, p := m.sizing(), m.pal
	label := " "
	if letter != 0 {
		label = string(letter - 'a' + 'A')
	}
	base := lipgloss.NewStyle().Bold(true).Foreground(p.Text)

	if state == tileRevealed {
		return theme.Block(label, s.tileW, s.tileH, base.Background(p.Mark(mark)))
	}
	if !s.large {
		bg := p.EmptyTile
		if state == tileTyped {
			bg = p.TypedTile
		}
		return theme.Block(label, s.tileW, s.tileH, base.Background(bg))
	}
	border := p.EmptyBorder
	if state == tileTyped {
		border = p.TypedBorder
	}
	return base.Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Render(theme.Block(label, s.tileW-2, s.tileH-2, lipgloss.NewStyle()))
}

func (m Model) key(label string, w int, mark game.Mark, hovered bool) string {
	s := m.sizing()
	bg := m.pal.Mark(mark)
	if hovered {
		bg = lipgloss.Lighten(bg, 0.25)
	}
	st := lipgloss.NewStyle().Bold(true).Foreground(m.pal.Text).Background(bg)
	return theme.Block(label, w, s.keyH, st)
}

// hintButton is the cheat-mode "?" button beside the input row. It is
// dimmed once all hints are used.
func (m Model) hintButton(exhausted, hovered bool) string {
	s := m.sizing()
	bg := m.pal.Present
	if exhausted {
		bg = m.pal.Absent
	}
	if hovered {
		bg = lipgloss.Lighten(bg, 0.25)
	}
	st := lipgloss.NewStyle().Bold(true).Foreground(m.pal.Text).Background(bg)
	return theme.Block("?", s.tileW, s.tileH, st)
}
