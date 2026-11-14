# gordle

A Wordle-style word game for the terminal, written in Go with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

### Download

Binaries built by a [github workflow](.github/workflows/release.yml) can be downloaded from the [releases page](https://github.com/MonkeeSage/gordle/releases).

Note: See [Releases](#Releases) for the command to allow the binary to run on macOS.

### Build

```
go build && ./gordle
```

### Flags

| Flag | |
|---|---|
| `-random` | play a random word instead of today's daily puzzle |
| `-game ID` | replay a shared game, e.g. `-game 317` or `-game R-K7Q` |
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

Every game has an ID, shown in the title and at the start of the shared
result: daily puzzles are numbered (`#317`) and random games get a code
(`R-K7Q`). Anyone can play the same word with `gordle -game <ID>`. Codes are
case-insensitive and the dash is optional. Future daily puzzles can't be
played early. IDs depend on the embedded word list, so regenerating the
lists changes which word an ID refers to.

Share copies the result with the desktop's clipboard tool (`wl-copy`,
`xclip` or `xsel` on Linux, `pbcopy` on macOS, `clip.exe` on Windows/WSL),
and also sends it as an OSC 52 terminal sequence, which covers SSH sessions
without a local clipboard. tmux drops OSC 52 from programs unless
`set -g set-clipboard on` is set.

## Word list

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

## Releases

Pushing a version tag (e.g. `git tag v1.0.0 && git push origin v1.0.0`) runs
the [release workflow](.github/workflows/release.yml): it tests, builds
`gordle` for Linux, macOS and Windows on amd64 and arm64, and publishes a
GitHub release with the archives and a `SHA256SUMS` file. `gordle-solver` is
not included. `gordle -version` prints the version a binary was built from.

The macOS binaries aren't signed, so Gatekeeper blocks them at first; allow
them with `xattr -d com.apple.quarantine gordle`.

## gordle-solver

A companion tool that lists the answers still possible given what you know.

```
go build ./cmd/gordle-solver && ./gordle-solver
```

Fill in the **Correct** row (green letters, one per box), the **Misplaced**
row (yellow letters; a box can hold several, e.g. "not E or A here") and the
**Excluded** field (gray letters), then press **Solve** (`ctrl+s`, `Enter` or
click). Candidates are listed best guess first, ranked by how many of the
remaining candidates share their letters, with known letters highlighted.

Click a box or use `Tab`/`Shift+Tab` and the arrow keys to move between
inputs; `Backspace` deletes a letter, `Delete` clears a box, `ctrl+r` clears
everything, and `PgUp`/`PgDn` or the mouse wheel scroll the results. Solving
with nothing filled in ranks every answer, which makes a good list of
starting words. `-colorblind` and `-no-mouse` work as in gordle.

A gray letter that is also green or yellow limits how many copies the
answer can have, matching how Wordle reports repeated letters.
