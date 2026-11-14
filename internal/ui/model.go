// Package ui is gordle's Bubble Tea front end.
package ui

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"gordle/internal/clipboard"
	"gordle/internal/game"
	"gordle/internal/words"
)

const maxGuesses = 6

// Options configure a session.
type Options struct {
	Puzzle     words.Puzzle // first game to play; today's daily puzzle if zero
	Hard       bool
	Colorblind bool
	NoMouse    bool
	Cheat      bool // allow revealing up to game.MaxHints letters; disables sharing
}

// Model is the Bubble Tea model.
type Model struct {
	opts Options
	pal  palette

	game     *game.Game
	gen      int // incremented per game so stale animation ticks are ignored
	puzzle   words.Puzzle
	slots    [words.Length]byte // input row; 0 = empty, hinted letters are locked
	pick     func(n int) int    // chooses hint positions; replaced in tests
	copyText func(string) error // native clipboard copy; replaced in tests
	keys     [26]game.Mark      // keyboard colours, updated once a reveal finishes

	revealed int // tiles revealed in the newest row; words.Length when idle
	shake    int // remaining shake frames for the input row
	toast    string
	toastID  int
	hover    string // ID of the layer under the mouse

	width, height int
}

type (
	revealMsg      struct{ gen int }
	shakeMsg       struct{ gen int }
	toastExpireMsg struct{ id int }
	clickMsg       struct{ id string }
	copiedMsg      struct{ err error }
	hoverMsg       struct{ id string }
)

const (
	revealDelay = 250 * time.Millisecond
	shakeDelay  = 40 * time.Millisecond
	toastTime   = 1500 * time.Millisecond
)

var shakeOffsets = []int{0, 1, -1, 2, -2, 2, -2, 1, -1}

const noHintsLeft = "Cheating got you half way there, but you must go on alone."

// New creates a model and starts the first game.
func New(opts Options) Model {
	m := Model{opts: opts, pal: standardPalette, pick: rand.IntN, copyText: clipboard.Copy}
	if opts.Colorblind {
		m.pal = colorblindPalette
	}
	if opts.Puzzle.Word != "" {
		m.newGame(opts.Puzzle)
	} else {
		m.newGame(words.Daily(time.Now()))
	}
	return m
}

func (m *Model) newGame(puzzle words.Puzzle) {
	m.game = game.New(puzzle.Word, maxGuesses, m.opts.Hard, words.IsValid)
	m.gen++
	m.puzzle = puzzle
	m.slots = [words.Length]byte{}
	m.keys = [26]game.Mark{}
	m.revealed = words.Length
	m.toast = ""
}

// resetInput clears the input row, keeping hinted letters in place.
func (m *Model) resetInput() {
	m.slots = [words.Length]byte{}
	for _, i := range m.game.Hints {
		m.slots[i] = m.game.Target[i]
	}
}

// inputWord returns the letters in the input row. It is shorter than
// words.Length if any slot is empty.
func (m Model) inputWord() string {
	var b strings.Builder
	for _, c := range m.slots {
		if c != 0 {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func (m Model) revealTick() tea.Cmd {
	gen := m.gen
	return tea.Tick(revealDelay, func(time.Time) tea.Msg { return revealMsg{gen} })
}

func (m Model) shakeTick() tea.Cmd {
	gen := m.gen
	return tea.Tick(shakeDelay, func(time.Time) tea.Msg { return shakeMsg{gen} })
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) revealing() bool { return m.revealed < words.Length }
func (m Model) over() bool      { return m.game.Status() != game.Playing && !m.revealing() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case clickMsg:
		return m.action(msg.id)

	case hoverMsg:
		m.hover = msg.id

	case revealMsg:
		if msg.gen != m.gen {
			break
		}
		m.revealed++
		if m.revealing() {
			return m, m.revealTick()
		}
		m.keys = m.game.Letters()

	case shakeMsg:
		if msg.gen != m.gen {
			break
		}
		m.shake--
		if m.shake > 0 {
			return m, m.shakeTick()
		}

	case copiedMsg:
		return m.showToast(copiedToast(msg.err, os.Getenv("TMUX") != ""))

	case toastExpireMsg:
		if msg.id == m.toastID {
			m.toast = ""
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k := msg.String(); k {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "ctrl+r":
		return m.action("btn:new")
	case "enter":
		return m.action("key:enter")
	case "backspace":
		return m.action("key:back")
	default:
		if msg.Text == "?" {
			return m.action("btn:hint")
		}
		if m.over() {
			switch k {
			case "r", "n":
				return m.action("btn:new")
			case "s":
				return m.action("btn:share")
			case "q":
				return m.action("btn:quit")
			}
			return m, nil
		}
		if t := strings.ToLower(msg.Text); len(t) == 1 && t[0] >= 'a' && t[0] <= 'z' {
			return m.action("key:" + t)
		}
	}
	return m, nil
}

// action performs a keyboard or mouse action, identified by the same IDs
// used for the clickable layers.
func (m Model) action(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "btn:quit":
		return m, tea.Quit
	case "btn:new":
		m.newGame(words.Random())
		return m, nil
	case "btn:share":
		if m.over() && !m.opts.Cheat {
			// OSC 52 covers SSH sessions; the native tool covers terminals and
			// tmux setups that ignore OSC 52. The result picks the message.
			text, copyText := m.shareText(), m.copyText
			return m, tea.Batch(tea.SetClipboard(text), func() tea.Msg {
				return copiedMsg{copyText(text)}
			})
		}
		return m, nil
	}

	if m.game.Status() != game.Playing || m.revealing() {
		return m, nil
	}
	switch id {
	case "btn:hint":
		return m.hint()
	case "key:enter":
		return m.submit()
	case "key:back":
		for i := len(m.slots) - 1; i >= 0; i-- {
			if m.slots[i] != 0 && !m.game.Hinted(i) {
				m.slots[i] = 0
				break
			}
		}
	default:
		if l, ok := strings.CutPrefix(id, "key:"); ok && len(l) == 1 {
			for i := range m.slots {
				if m.slots[i] == 0 {
					m.slots[i] = l[0]
					break
				}
			}
		}
	}
	return m, nil
}

// hint reveals a correct letter in the input row (cheat mode only). It
// replaces whatever was typed in that position.
func (m Model) hint() (tea.Model, tea.Cmd) {
	if !m.opts.Cheat {
		return m, nil
	}
	i, err := m.game.Hint(m.pick)
	if errors.Is(err, game.ErrNoHints) {
		return m.showToast(noHintsLeft)
	}
	if err != nil {
		return m, nil
	}
	m.slots[i] = m.game.Target[i]
	m.keys = m.game.Letters()
	return m, nil
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	if _, err := m.game.Submit(m.inputWord()); err != nil {
		msg := err.Error()
		m, expire := m.showToast(strings.ToUpper(msg[:1]) + msg[1:])
		m.shake = len(shakeOffsets)
		return m, tea.Batch(expire, m.shakeTick())
	}
	m.resetInput()
	m.revealed = 0
	return m, m.revealTick()
}

// showToast displays s and returns the command that will clear it. Newer
// toasts replace older ones, and an older toast's timer won't clear them.
func (m Model) showToast(s string) (Model, tea.Cmd) {
	m.toast = s
	m.toastID++
	id := m.toastID
	return m, tea.Tick(toastTime, func(time.Time) tea.Msg { return toastExpireMsg{id} })
}

func (m Model) shareText() string {
	score := "X"
	if m.game.Status() == game.Won {
		score = fmt.Sprint(len(m.game.Guesses))
	}
	hard := ""
	if m.game.Hard {
		hard = "*"
	}
	return fmt.Sprintf("Gordle %s %s/%d%s\n\n%s", m.puzzle.ID(), score, maxGuesses, hard, m.game.Share())
}

// copiedToast describes the outcome of sharing. When the native copy fails
// only OSC 52 is left, which can't report success, and which tmux drops by
// default.
func copiedToast(err error, inTmux bool) string {
	switch {
	case err == nil:
		return "Copied result to clipboard"
	case inTmux:
		return "No clipboard tool found; in tmux, copying needs set-clipboard on"
	}
	return "Sent result to the terminal clipboard (OSC 52)"
}

var praise = []string{"Genius!", "Magnificent!", "Impressive!", "Splendid!", "Great!", "Phew!"}

// statusLine is the message shown under the board.
func (m Model) statusLine() string {
	if m.toast != "" {
		return m.toast
	}
	if !m.over() {
		return ""
	}
	if m.game.Status() == game.Won {
		return praise[min(len(m.game.Guesses), len(praise))-1]
	}
	return "The word was " + strings.ToUpper(m.game.Target)
}

func (m Model) sizing() sizing {
	if m.width >= 63 && m.height >= 35 {
		return large
	}
	return compact
}

// View lays the screen out as positioned layers. Interactive layers carry IDs
// so mouse events can be hit-tested against exactly what was drawn.
func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.WindowTitle = "gordle"

	const minW, minH = 45, 22
	if m.width < minW || m.height < minH {
		msg := fmt.Sprintf("Terminal too small\nneed %dx%d, have %dx%d", minW, minH, m.width, m.height)
		v.SetContent(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg))
		return v
	}

	s := m.sizing()
	dim := lipgloss.NewStyle().Foreground(m.pal.dim)
	var layers []*lipgloss.Layer

	kbRows := []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"}
	kbW := 2*s.wideKeyW + 7*s.keyW + 8*s.keyGap
	contentW := max(s.boardW(), kbW)
	center := func(w int) int { return (contentW - w) / 2 }
	y := 0

	// Title.
	sub := "random " + m.puzzle.ID()
	if m.puzzle.Number > 0 {
		sub = "daily " + m.puzzle.ID()
	}
	if m.game.Hard {
		sub += " · hard"
	}
	if m.opts.Cheat {
		sub += " · cheat"
	}
	title := lipgloss.NewStyle().Bold(true).Render("GORDLE") + "  " + dim.Render(sub)
	layers = append(layers, lipgloss.NewLayer(title).X(center(lipgloss.Width(title))).Y(y))
	y += 2

	// Board: one layer per row so the input row can shake.
	for r := range maxGuesses {
		tiles := make([]string, words.Length)
		for c := range tiles {
			tiles[c] = m.boardTile(r, c)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, intersperse(tiles, " ")...)
		x := center(s.boardW())
		if r == len(m.game.Guesses) && m.shake > 0 {
			x += shakeOffsets[len(shakeOffsets)-m.shake]
		}
		layers = append(layers, lipgloss.NewLayer(row).X(x).Y(y+r*(s.tileH+s.boardRowGap)))
	}
	if m.opts.Cheat && m.game.Status() == game.Playing {
		btn := m.hintButton(len(m.game.Hints) >= game.MaxHints, m.hover == "btn:hint")
		layers = append(layers, lipgloss.NewLayer(btn).ID("btn:hint").
			X(center(s.boardW())+s.boardW()+2).Y(y+len(m.game.Guesses)*(s.tileH+s.boardRowGap)))
	}
	y += s.boardH() + 1

	// Status line and end-of-game buttons.
	if line := m.statusLine(); line != "" {
		// Long toasts wrap onto the buttons' line, which is free during play.
		line = lipgloss.NewStyle().Bold(true).Width(contentW).Align(lipgloss.Center).Render(line)
		layers = append(layers, lipgloss.NewLayer(line).Y(y))
	}
	if m.over() {
		btns := []struct{ id, label string }{{"btn:new", "New game (r)"}}
		if !m.opts.Cheat {
			btns = append(btns, struct{ id, label string }{"btn:share", "Share (s)"})
		}
		btns = append(btns, struct{ id, label string }{"btn:quit", "Quit (q)"})
		rendered := make([]string, len(btns))
		total := 2 * (len(btns) - 1)
		for i, b := range btns {
			rendered[i] = m.button(b.label, m.hover == b.id)
			total += lipgloss.Width(rendered[i])
		}
		x := center(total)
		for i, b := range btns {
			layers = append(layers, lipgloss.NewLayer(rendered[i]).ID(b.id).X(x).Y(y+1))
			x += lipgloss.Width(rendered[i]) + 2
		}
	}
	y += 3

	// Keyboard: one layer per key.
	for r, letters := range kbRows {
		type k struct {
			id, label string
			w         int
			mark      game.Mark
		}
		var row []k
		if r == 2 {
			row = append(row, k{"key:enter", "ENTER", s.wideKeyW, game.Unknown})
		}
		for _, l := range letters {
			row = append(row, k{"key:" + string(l), strings.ToUpper(string(l)), s.keyW, m.keys[l-'a']})
		}
		if r == 2 {
			row = append(row, k{"key:back", "⌫", s.wideKeyW, game.Unknown})
		}
		rowW := s.keyGap * (len(row) - 1)
		for _, key := range row {
			rowW += key.w
		}
		x := center(rowW)
		for _, key := range row {
			layers = append(layers, lipgloss.NewLayer(m.key(key.label, key.w, key.mark, m.hover == key.id)).
				ID(key.id).X(x).Y(y+r*(s.keyH+s.kbRowGap)))
			x += key.w + s.keyGap
		}
	}
	y += s.kbH()

	if m.height >= y+2 {
		items := []string{"ctrl+r restart", "esc quit"}
		if m.opts.Cheat {
			items = append([]string{"? hint"}, items...)
		}
		help := dim.Render(strings.Join(items, " · "))
		layers = append(layers, lipgloss.NewLayer(help).X(center(lipgloss.Width(help))).Y(y+1))
		y += 2
	}

	root := lipgloss.NewLayer("", layers...).X((m.width - contentW) / 2).Y((m.height - y) / 2)
	comp := lipgloss.NewCompositor(root)
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

func (m Model) boardTile(r, c int) string {
	gs := m.game.Guesses
	switch {
	case r < len(gs):
		shown := words.Length
		if r == len(gs)-1 {
			shown = m.revealed
		}
		if c < shown {
			return m.tile(gs[r].Word[c], tileRevealed, gs[r].Marks[c])
		}
		return m.tile(gs[r].Word[c], tileTyped, game.Unknown)
	case r == len(gs) && m.game.Status() == game.Playing && m.slots[c] != 0:
		if m.game.Hinted(c) {
			return m.tile(m.slots[c], tileRevealed, game.Correct)
		}
		return m.tile(m.slots[c], tileTyped, game.Unknown)
	}
	return m.tile(0, tileEmpty, game.Unknown)
}

func intersperse(items []string, sep string) []string {
	out := make([]string, 0, 2*len(items))
	for i, it := range items {
		if i > 0 {
			out = append(out, sep)
		}
		out = append(out, it)
	}
	return out
}
