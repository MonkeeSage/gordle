package words

import (
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
	n1, w1 := Daily(time.Date(2026, 1, 1, 0, 30, 0, 0, loc))
	n2, w2 := Daily(time.Date(2026, 1, 1, 23, 30, 0, 0, loc))
	if n1 != 1 || n2 != 1 || w1 != w2 {
		t.Errorf("same local day gave #%d %q and #%d %q", n1, w1, n2, w2)
	}
	n3, w3 := Daily(time.Date(2026, 1, 2, 0, 0, 0, 0, loc))
	if n3 != 2 || w3 == w1 {
		t.Errorf("next day gave #%d %q", n3, w3)
	}
	// Dates before the epoch must not panic.
	Daily(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
}
