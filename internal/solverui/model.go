// Package solverui is the Bubble Tea front end of gordle-solver.
package solverui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"gordle/internal/solver"
	"gordle/internal/theme"
)

// Focusable inputs, in tab order: the correct boxes, the misplaced boxes,
// the excluded field and the Solve button.
const (
	focusMisplaced = solver.Length
	focusExcluded  = 2 * solver.Length
	focusSolve     = focusExcluded + 1
	numFocus       = focusSolve + 1

	maxMisplaced = 5 // letters per misplaced box
)

// Options configure the solver.
type Options struct {
	Words      []string // candidate answers
	Colorblind bool
	NoMouse    bool
}

// Model is the Bubble Tea model.
type Model struct {
	opts Options
	pal  theme.Palette

	c     solver.Constraints
	focus int

	results []string
	solved  bool // results are shown
	stale   bool // inputs changed since the last solve
	scroll  int  // first result row shown
	hover   string

	width, height int
}

type (
	clickMsg struct{ id string }
	hoverMsg struct{ id string }
)

// New creates a solver model.
func New(opts Options) Model {
	m := Model{opts: opts, pal: theme.Standard}
	if opts.Colorblind {
		m.pal = theme.Colorblind
	}
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scroll = m.clampScroll(m.scroll)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case clickMsg:
		return m.click(msg.id)
	case hoverMsg:
		m.hover = msg.id
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.scroll = m.clampScroll(m.scroll - 1)
		case tea.MouseWheelDown:
			m.scroll = m.clampScroll(m.scroll + 1)
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "ctrl+s", "enter":
		return m.solve(), nil
	case "ctrl+r":
		return New(Options{Words: m.opts.Words, Colorblind: m.opts.Colorblind, NoMouse: m.opts.NoMouse}).
			withSize(m.width, m.height), nil
	case "tab":
		m.focus = (m.focus + 1) % numFocus
	case "shift+tab":
		m.focus = (m.focus + numFocus - 1) % numFocus
	case "right":
		if m.focus < focusExcluded && m.focus%solver.Length < solver.Length-1 {
			m.focus++
		}
	case "left":
		if m.focus < focusExcluded && m.focus%solver.Length > 0 {
			m.focus--
		}
	case "down":
		switch {
		case m.focus < focusMisplaced:
			m.focus += solver.Length
		case m.focus < focusExcluded:
			m.focus = focusExcluded
		case m.focus == focusExcluded:
			m.focus = focusSolve
		}
	case "up":
		switch {
		case m.focus == focusSolve:
			m.focus = focusExcluded
		case m.focus == focusExcluded:
			m.focus = focusMisplaced
		case m.focus >= focusMisplaced:
			m.focus -= solver.Length
		}
	case "pgdown":
		m.scroll = m.clampScroll(m.scroll + m.resultRows())
	case "pgup":
		m.scroll = m.clampScroll(m.scroll - m.resultRows())
	case "backspace":
		m.edit(func(s string) string { return s[:max(len(s)-1, 0)] })
	case "delete":
		m.edit(func(string) string { return "" })
	default:
		if t := strings.ToLower(msg.Text); len(t) == 1 && t[0] >= 'a' && t[0] <= 'z' {
			m.typeLetter(t[0])
		}
	}
	return m, nil
}

func (m Model) withSize(w, h int) Model {
	m.width, m.height = w, h
	return m
}

// edit applies f to the text of the focused input.
func (m *Model) edit(f func(string) string) {
	switch {
	case m.focus < focusMisplaced:
		s := ""
		if c := m.c.Correct[m.focus]; c != 0 {
			s = string(c)
		}
		if s = f(s); s == "" {
			m.c.Correct[m.focus] = 0
		} else {
			m.c.Correct[m.focus] = s[0]
		}
	case m.focus < focusExcluded:
		i := m.focus - focusMisplaced
		m.c.Misplaced[i] = f(m.c.Misplaced[i])
	case m.focus == focusExcluded:
		m.c.Excluded = f(m.c.Excluded)
	default:
		return
	}
	m.stale = m.solved
}

func (m *Model) typeLetter(l byte) {
	m.edit(func(s string) string {
		switch {
		case m.focus < focusMisplaced:
			return string(l) // one letter: replace
		case strings.IndexByte(s, l) >= 0:
			return s // no duplicates
		case m.focus < focusExcluded && len(s) >= maxMisplaced:
			return s
		}
		return s + string(l)
	})
}

func (m Model) click(id string) (tea.Model, tea.Cmd) {
	if id == "btn:solve" {
		m.focus = focusSolve
		return m.solve(), nil
	}
	var n int
	if _, err := fmt.Sscanf(id, "focus:%d", &n); err == nil && n >= 0 && n < numFocus {
		m.focus = n
	}
	return m, nil
}

func (m Model) solve() Model {
	m.results = solver.Solve(m.c, m.opts.Words)
	m.solved, m.stale, m.scroll = true, false, 0
	return m
}

// Layout constants, in cells.
const (
	labelW = 11
	boxW   = 7 // including border
	boxH   = 3
	rowW   = solver.Length*boxW + solver.Length - 1
	formW  = labelW + rowW
	cellW  = solver.Length + 2 // a result word plus its gap
	minW   = formW + 2
	minH   = 22
	formH  = 15 // title through the status line
)

// gridCols is how many result words fit on a line.
func (m Model) gridCols() int { return max((m.width-4+2)/cellW, 1) }

// resultRows is how many result lines fit on screen.
// Below the form: a blank line, the results, a blank line and the help.
func (m Model) resultRows() int { return max(m.height-1-formH-1-2, 1) }

func (m Model) clampScroll(s int) int {
	total := (len(m.results) + m.gridCols() - 1) / m.gridCols()
	return max(min(s, total-m.resultRows()), 0)
}

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.WindowTitle = "gordle-solver"

	if m.width < minW || m.height < minH {
		msg := fmt.Sprintf("Terminal too small\nneed %dx%d, have %dx%d", minW, minH, m.width, m.height)
		v.SetContent(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
		return v
	}

	p := m.pal
	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Foreground(p.Dim)
	var form []*lipgloss.Layer
	add := func(l *lipgloss.Layer) { form = append(form, l) }

	title := bold.Render("GORDLE SOLVER")
	add(lipgloss.NewLayer(title).X((formW - lipgloss.Width(title)) / 2))

	y := 2
	for row, label := range []string{"Correct", "Misplaced"} {
		add(lipgloss.NewLayer(label).Y(y + 1))
		for i := range solver.Length {
			f := row*solver.Length + i
			var text string
			bg := p.Correct
			if row == 0 {
				if c := m.c.Correct[i]; c != 0 {
					text = string(c)
				}
			} else {
				text, bg = m.c.Misplaced[i], p.Present
			}
			add(lipgloss.NewLayer(m.box(f, text, bg)).ID(fmt.Sprintf("focus:%d", f)).
				X(labelW + i*(boxW+1)).Y(y))
		}
		y += boxH
	}

	add(lipgloss.NewLayer("Excluded").Y(y + 1))
	add(lipgloss.NewLayer(m.excludedField()).ID(fmt.Sprintf("focus:%d", focusExcluded)).X(labelW).Y(y))
	y += boxH + 1

	add(lipgloss.NewLayer(m.solveButton()).ID("btn:solve").X(labelW).Y(y))
	y += 2

	add(lipgloss.NewLayer(dim.Render(m.status())).Y(y))

	layers := []*lipgloss.Layer{lipgloss.NewLayer("", form...).X((m.width - formW) / 2).Y(1)}

	if m.solved && len(m.results) > 0 {
		grid := m.grid()
		layers = append(layers, lipgloss.NewLayer(grid).
			X((m.width-lipgloss.Width(grid))/2).Y(1+formH+1))
	}

	help := "tab move · ctrl+s solve · ctrl+r clear · esc quit"
	if m.width >= 72 {
		help = "tab move · ctrl+s solve · pgup/pgdn scroll · ctrl+r clear · esc quit"
	}
	help = dim.Render(help)
	layers = append(layers, lipgloss.NewLayer(help).X((m.width-lipgloss.Width(help))/2).Y(m.height-1))

	comp := lipgloss.NewCompositor(layers...)
	v.SetContent(lipgloss.NewCanvas(m.width, m.height).Compose(comp).Render())

	if !m.opts.NoMouse {
		v.MouseMode = tea.MouseModeAllMotion
		hover := m.hover
		v.OnMouse = func(msg tea.MouseMsg) tea.Cmd {
			mouse := msg.Mouse()
			id := comp.Hit(mouse.X, mouse.Y).ID()
			switch msg.(type) {
			case tea.MouseClickMsg:
				if mouse.Button == tea.MouseLeft && id != "" {
					return func() tea.Msg { return clickMsg{id} }
				}
			case tea.MouseMotionMsg:
				if id != hover {
					return func() tea.Msg { return hoverMsg{id} }
				}
			}
			return nil
		}
	}
	return v
}

// border picks the outline for input f: thick and bright when focused,
// lighter when hovered or filled.
func (m Model) border(f int, filled bool) lipgloss.Style {
	p := m.pal
	switch {
	case m.focus == f:
		return lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(p.Focus)
	case m.hover == fmt.Sprintf("focus:%d", f) || filled:
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.TypedBorder)
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.EmptyBorder)
}

// box renders a correct or misplaced box. Its letters are shown on bg.
func (m Model) box(f int, text string, bg color.Color) string {
	inner := strings.Repeat(" ", boxW-2)
	if text != "" {
		st := lipgloss.NewStyle().Bold(true).Foreground(m.pal.Text).Background(bg)
		inner = theme.Block(strings.ToUpper(text), boxW-2, 1, st)
	}
	return m.border(f, text != "").Render(inner)
}

func (m Model) excludedField() string {
	text := strings.ToUpper(m.c.Excluded)
	if len(text)*2-1 <= rowW-4 {
		text = strings.Join(strings.Split(text, ""), " ")
	}
	if m.focus == focusExcluded {
		text += "▏"
	}
	inner := lipgloss.NewStyle().Foreground(m.pal.Text).Width(rowW - 2).Render(" " + text)
	return m.border(focusExcluded, m.c.Excluded != "").Render(inner)
}

func (m Model) solveButton() string {
	p := m.pal
	bg := p.Absent
	if m.focus == focusSolve || m.hover == "btn:solve" {
		bg = lipgloss.Lighten(bg, 0.25)
	}
	st := lipgloss.NewStyle().Bold(m.focus == focusSolve).Foreground(p.Text).Background(bg)
	return theme.Block("Solve (ctrl+s)", len("Solve (ctrl+s)")+4, 1, st)
}

func (m Model) status() string {
	switch {
	case !m.solved:
		return "Fill in what you know, then Solve."
	case len(m.results) == 0:
		return "No words match. Check for conflicting letters."
	}
	s := fmt.Sprintf("%d candidates, best first", len(m.results))
	if len(m.results) == 1 {
		s = "1 candidate"
	}
	cols := m.gridCols()
	if total := (len(m.results) + cols - 1) / cols; total > m.resultRows() {
		s += fmt.Sprintf(" · rows %d-%d of %d",
			m.scroll+1, min(m.scroll+m.resultRows(), total), total)
	}
	if m.stale {
		s += " · edited, solve again"
	}
	return s
}

// grid renders the visible result rows, colouring known letters.
func (m Model) grid() string {
	p := m.pal
	green := lipgloss.NewStyle().Bold(true).Foreground(p.Text).Background(p.Correct)
	yellow := lipgloss.NewStyle().Bold(true).Foreground(p.Text).Background(p.Present)
	plain := lipgloss.NewStyle().Foreground(p.Text)

	cols := m.gridCols()
	start := m.scroll * cols
	end := min(start+m.resultRows()*cols, len(m.results))
	var lines []string
	var line strings.Builder
	for i, w := range m.results[start:end] {
		if i > 0 && i%cols == 0 {
			lines = append(lines, line.String())
			line.Reset()
		}
		if i%cols != 0 {
			line.WriteString("  ")
		}
		for j := range solver.Length {
			st := plain
			switch {
			case m.c.Correct[j] == w[j]:
				st = green
			case m.c.IsMisplaced(w[j]):
				st = yellow
			}
			line.WriteString(st.Render(strings.ToUpper(w[j : j+1])))
		}
	}
	lines = append(lines, line.String())
	return strings.Join(lines, "\n")
}
