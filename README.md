# gordle

A Wordle-style word game for the terminal, written in Go with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

```
go build && ./gordle
```

| Flag | |
|---|---|
| `-random` | play a random word instead of today's daily puzzle |
| `-hard` | hard mode: greens must stay put, yellows must be reused |
| `-cheat` | casual mode: reveal up to 3 letters with `?` (disables sharing) |
| `-colorblind` | orange/blue instead of green/yellow |
| `-no-mouse` | don't enable mouse reporting |

Type letters, `Enter` to submit, `Backspace` to delete, `Ctrl+R` to restart
with a new random word, `Esc` to quit. After a game: `r` new random game, `s`
copy the result grid to the clipboard, `q` quit.

### Cheat mode

With `-cheat`, a `?` button sits beside the row you're typing. Click it or
press `?` to reveal a correct letter in a random position you haven't solved
yet. It turns green on the board and keyboard and stays locked in place for
the rest of the game. Up to 3 letters can be revealed per game. Results can't
be shared in cheat mode.

## Mouse

The on-screen keyboard and the end-of-game buttons are clickable, and keys
highlight on hover. Any terminal with SGR mouse reporting works (xterm, kitty,
WezTerm, Alacritty, foot, iTerm2, Windows Terminal, ...).

Under tmux, mouse events are only forwarded with `set -g mouse on`.

## Sharing

Share copies the result with the desktop's clipboard tool (`wl-copy`,
`xclip` or `xsel` on Linux, `pbcopy` on macOS, `clip.exe` on Windows/WSL),
and also sends it as an OSC 52 terminal sequence, which covers SSH sessions
without a local clipboard. tmux drops OSC 52 from programs unless
`set -g set-clipboard on` is set.

## Layout

The board scales to the terminal: large 5x3 tiles from 63x35, compact tiles
down to 45x22.

## Word lists

`internal/words/answers.txt` (about 2,000 common words, no plurals or
inflections) and `valid.txt` (about 11,000 accepted guesses) are generated
from [SCOWL](http://wordlist.aspell.net/) and embedded with `go:embed`.
To regenerate them from an extracted SCOWL release:

```
SCOWL=/path/to/scowl-2020.12.07 go generate ./internal/words
```

Words in `tools/genwords/exclude.txt` are never chosen as answers (they're
still accepted as guesses). SCOWL's license is in
`internal/words/SCOWL-COPYRIGHT`.
