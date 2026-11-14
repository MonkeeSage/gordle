package words

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLists(t *testing.T) {
	if Count() < 1000 {
		t.Fatalf("only %d answers embedded", Count())
	}
	for _, w := range answers {
		if len(w) != Length {
			t.Errorf("answer %q has wrong length", w)
		}
		if !IsValid(w) {
			t.Errorf("answer %q is not a valid guess", w)
		}
	}
	if IsValid("zzzzz") {
		t.Error("zzzzz should not be valid")
	}
}

func TestDaily(t *testing.T) {
	loc := time.FixedZone("X", -7*3600)
	p1 := Daily(time.Date(2026, 1, 1, 0, 30, 0, 0, loc))
	p2 := Daily(time.Date(2026, 1, 1, 23, 30, 0, 0, loc))
	if p1.Number != 1 || p1 != p2 {
		t.Errorf("same local day gave %+v and %+v", p1, p2)
	}
	p3 := Daily(time.Date(2026, 1, 2, 0, 0, 0, 0, loc))
	if p3.Number != 2 || p3.Word == p1.Word || p3.ID() != "#2" {
		t.Errorf("next day gave %+v (%s)", p3, p3.ID())
	}
	// Dates before the epoch must not panic.
	Daily(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
}

func TestRandomCodesRoundTrip(t *testing.T) {
	now := time.Now()
	seen := map[string]bool{}
	for i := range answers {
		p := randomIndex(i)
		if seen[p.Code] {
			t.Fatalf("duplicate code %s", p.Code)
		}
		seen[p.Code] = true
		for _, id := range []string{p.Code, strings.ToLower(p.Code), strings.Replace(p.Code, "-", "", 1)} {
			got, err := Parse(id, now)
			if err != nil || got != p {
				t.Fatalf("Parse(%q) = %+v, %v; want %+v", id, got, err, p)
			}
		}
	}
	if p := Random(); !strings.HasPrefix(p.Code, "R-") || p.Number != 0 || p.ID() != p.Code {
		t.Errorf("Random() = %+v", p)
	}
}

func TestCodeLookAlikes(t *testing.T) {
	// Find a code containing 0 or 1 and type it with O or I/L instead.
	for i := range answers {
		c := randomIndex(i).Code
		if !strings.ContainsAny(c[2:], "01") {
			continue
		}
		typed := strings.NewReplacer("0", "o", "1", "l").Replace(c)
		if got, err := Parse(typed, time.Now()); err != nil || got.Code != c {
			t.Errorf("Parse(%q) = %+v, %v; want %s", typed, got, err, c)
		}
		return
	}
	t.Skip("no code contains 0 or 1")
}

func TestCodesRejectMostTypos(t *testing.T) {
	valid := 0
	for v := range 32 * 32 * 32 {
		s := string([]byte{codeDigits[v/1024], codeDigits[v/32%32], codeDigits[v%32]})
		if _, ok := decodeCode(s); ok {
			valid++
		}
	}
	if valid != len(answers) {
		t.Errorf("%d codes decode, want exactly %d (one per answer)", valid, len(answers))
	}
}

func TestParse(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.Local)
	today := Daily(now)
	for _, id := range []string{"317", "#317", " #317 "} {
		if p, err := Parse(id, now); err != nil || p != today {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", id, p, err, today)
		}
	}
	if p, err := Parse("1", now); err != nil || p.Number != 1 {
		t.Errorf("Parse(1) = %+v, %v", p, err)
	}
	if _, err := Parse("318", now); !errors.Is(err, ErrFutureGame) {
		t.Errorf("tomorrow's puzzle: err = %v", err)
	}
	for _, id := range []string{"", "0", "-3", "abc", "R-", "R-UUU", "R-ZZZZ", "R-!!!"} {
		if _, err := Parse(id, now); !errors.Is(err, ErrUnknownGame) {
			t.Errorf("Parse(%q): err = %v, want ErrUnknownGame", id, err)
		}
	}
}
