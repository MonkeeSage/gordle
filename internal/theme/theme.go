// Package theme holds the colours and drawing helpers shared by gordle and
// gordle-solver.
package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"gordle/internal/game"
)

// Palette is a colour scheme.
type Palette struct {
	Correct, Present, Absent color.Color
	Key                      color.Color // unguessed keyboard key
	EmptyTile, TypedTile     color.Color // compact tile backgrounds
	EmptyBorder, TypedBorder color.Color // bordered tile outlines
	Focus                    color.Color // outline of the focused input
	Text, Dim                color.Color
}

var (
	Standard = Palette{
		Correct:     lipgloss.Color("#538d4e"),
		Present:     lipgloss.Color("#b59f3b"),
		Absent:      lipgloss.Color("#3a3a3c"),
		Key:         lipgloss.Color("#818384"),
		EmptyTile:   lipgloss.Color("#262628"),
		TypedTile:   lipgloss.Color("#565758"),
		EmptyBorder: lipgloss.Color("#3a3a3c"),
		TypedBorder: lipgloss.Color("#878a8c"),
		Focus:       lipgloss.Color("#d7dadc"),
		Text:        lipgloss.Color("#ffffff"),
		Dim:         lipgloss.Color("#818384"),
	}
	Colorblind = func() Palette {
		p := Standard
		p.Correct = lipgloss.Color("#f5793a")
		p.Present = lipgloss.Color("#85c0f9")
		return p
	}()
)

// Mark returns the background colour for a letter's mark.
func (p Palette) Mark(m game.Mark) color.Color {
	switch m {
	case game.Correct:
		return p.Correct
	case game.Present:
		return p.Present
	case game.Absent:
		return p.Absent
	}
	return p.Key
}

// Button renders a one-line button, lighter while hovered.
func (p Palette) Button(label string, hovered bool) string {
	bg := p.Absent
	if hovered {
		bg = lipgloss.Lighten(bg, 0.25)
	}
	st := lipgloss.NewStyle().Foreground(p.Text).Background(bg)
	return Block(label, lipgloss.Width(label)+2, 1, st)
}

// Block renders label centred in a w x h rectangle filled with st's
// background.
func Block(label string, w, h int, st lipgloss.Style) string {
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
