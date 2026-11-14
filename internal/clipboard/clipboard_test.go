//go:build linux

package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// fakeSession makes a PATH containing only a fake clipboard tool called name,
// which saves its input to the returned file and then leaves a background
// process running, like wl-copy and xclip do.
func fakeSession(t *testing.T, name string) string {
	cat, err1 := exec.LookPath("cat")
	sleep, err2 := exec.LookPath("sleep")
	if err1 != nil || err2 != nil {
		t.Skip("cat or sleep not found")
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "clipboard")
	script := "#!/bin/sh\n" + cat + " > " + out + "\n" + sleep + " 3 &\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")
	t.Setenv("WSL_DISTRO_NAME", "")
	return out
}

func TestCopy(t *testing.T) {
	for _, tt := range []struct{ tool, env string }{
		{"wl-copy", "WAYLAND_DISPLAY"},
		{"xclip", "DISPLAY"},
		{"xsel", "DISPLAY"},
	} {
		out := fakeSession(t, tt.tool)
		t.Setenv(tt.env, "test")

		start := time.Now()
		if err := Copy("Gordle 3/6\n\n🟩🟩🟩🟩🟩"); err != nil {
			t.Fatalf("%s: %v", tt.tool, err)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("%s: Copy waited %v for the background process", tt.tool, d)
		}
		got, err := os.ReadFile(out)
		if err != nil || string(got) != "Gordle 3/6\n\n🟩🟩🟩🟩🟩" {
			t.Errorf("%s: clipboard = %q, %v", tt.tool, got, err)
		}
	}
}

func TestCopyUnavailable(t *testing.T) {
	fakeSession(t, "wl-copy")
	// Tool installed, but no display session to use it with.
	if err := Copy("x"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}
