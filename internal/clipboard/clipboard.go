// Package clipboard copies text using the desktop's native clipboard tool.
//
// Terminal-based copying (OSC 52) is unreliable: tmux drops it unless
// set-clipboard is on, and some terminals ignore it. A native tool works
// regardless of the terminal whenever the program runs on the desktop
// session itself.
package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrUnavailable means no clipboard tool was found for this session.
var ErrUnavailable = errors.New("no clipboard tool available")

type tool struct {
	name string
	args []string
}

// tools lists the clipboard commands to try, in order, for this session.
func tools() []tool {
	switch runtime.GOOS {
	case "darwin":
		return []tool{{"pbcopy", nil}}
	case "windows":
		return []tool{{"clip.exe", nil}}
	}
	var ts []tool
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		ts = append(ts, tool{"wl-copy", nil})
	}
	if os.Getenv("DISPLAY") != "" {
		ts = append(ts,
			tool{"xclip", []string{"-selection", "clipboard"}},
			tool{"xsel", []string{"--clipboard", "--input"}})
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		ts = append(ts, tool{"clip.exe", nil})
	}
	return ts
}

// Copy puts s on the system clipboard using the first tool that works.
func Copy(s string) error {
	err := ErrUnavailable
	for _, t := range tools() {
		path, lookErr := exec.LookPath(t.name)
		if lookErr != nil {
			continue
		}
		cmd := exec.Command(path, t.args...)
		cmd.Stdin = strings.NewReader(s)
		// Stdout and Stderr stay nil (/dev/null): wl-copy and xclip leave a
		// background process serving the clipboard, and if it inherited our
		// pipes Run would wait for it to exit.
		if err = cmd.Run(); err == nil {
			return nil
		}
	}
	return err
}
