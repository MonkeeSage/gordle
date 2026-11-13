package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"gordle/internal/game"
)

type palette struct {
	correct, present, absent color.Color
	key                      color.Color // unguessed keyboard key
	emptyTile, typedTile     color.Color // compact mode tile backgrounds
	emptyBorder, typedBorder color.Color // large mode tile borders
	text, dim                color.Color
}

var (
	standardPalette = palette{
		correct:     lipgloss.Color("#538d4e"),
		present:     lipgloss.Color("#b59f3b"),
		absent:      lipgloss.Color("#3a3a3c"),
		key:         lipgloss.Color("#818384"),
		emptyTile:   lipgloss.Color("#262628"),
		typedTile:   lipgloss.Color("#565758"),
		emptyBorder: lipgloss.Color("#3a3a3c"),
		typedBorder: lipgloss.Color("#878a8c"),
		text:        lipgloss.Color("#ffffff"),
		dim:         lipgloss.Color("#818384"),
	}
	colorblindPalette = func() palette {
		p := standardPalette
		p.correct = lipgloss.Color("#f5793a")
		p.present = lipgloss.Color("#85c0f9")
		return p
	}()
)

func (p palette) mark(m game.Mark) color.Color {
	switch m {
	case game.Correct:
		return p.correct
	case game.Present:
		return p.present
	case game.Absent:
		return p.absent
	}
	return p.key
}

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

// block renders label centred in a w x h rectangle filled with st's background.
func block(label string, w, h int, st lipgloss.Style) string {
	lines := make([]string, h)
	for i := range lines {
		if i == h/2 {
			lw := lipgloss.Width(label)
			left := (w - lw) / 2
			lines[i] = strings.Repeat(" ", left) + label + strings.Repeat(" ", w-lw-left)
		} else {
			lines[i] = strings.Repeat(" ", w)
		}
	}
	return st.Render(strings.Join(lines, "\n"))
}

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
	base := lipgloss.NewStyle().Bold(true).Foreground(p.text)

	if state == tileRevealed {
		return block(label, s.tileW, s.tileH, base.Background(p.mark(mark)))
	}
	if !s.large {
		bg := p.emptyTile
		if state == tileTyped {
			bg = p.typedTile
		}
		return block(label, s.tileW, s.tileH, base.Background(bg))
	}
	border := p.emptyBorder
	if state == tileTyped {
		border = p.typedBorder
	}
	return base.Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Render(block(label, s.tileW-2, s.tileH-2, lipgloss.NewStyle()))
}

func (m Model) key(label string, w int, mark game.Mark, hovered bool) string {
	s := m.sizing()
	bg := m.pal.mark(mark)
	if hovered {
		bg = lipgloss.Lighten(bg, 0.25)
	}
	st := lipgloss.NewStyle().Bold(true).Foreground(m.pal.text).Background(bg)
	return block(label, w, s.keyH, st)
}

func (m Model) button(label string, hovered bool) string {
	bg := m.pal.absent
	if hovered {
		bg = lipgloss.Lighten(bg, 0.25)
	}
	st := lipgloss.NewStyle().Foreground(m.pal.text).Background(bg)
	return block(label, lipgloss.Width(label)+2, 1, st)
}

// hintButton is the cheat-mode "?" button beside the input row. It is
// dimmed once all hints are used.
func (m Model) hintButton(exhausted, hovered bool) string {
	s := m.sizing()
	bg := m.pal.present
	if exhausted {
		bg = m.pal.absent
	}
	if hovered {
		bg = lipgloss.Lighten(bg, 0.25)
	}
	st := lipgloss.NewStyle().Bold(true).Foreground(m.pal.text).Background(bg)
	return block("?", s.tileW, s.tileH, st)
}
