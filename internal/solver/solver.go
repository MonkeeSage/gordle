// Package solver finds the words that fit what's known about a Wordle answer.
package solver

import (
	"cmp"
	"slices"
	"strings"
)

// Length is the number of letters in a word.
const Length = 5

// Constraints is what the player knows. Letters are lowercase a-z.
type Constraints struct {
	// Correct holds the green letter at each position, or 0 if unknown.
	Correct [Length]byte
	// Misplaced holds the yellow letters seen at each position: each is in
	// the word, but not there.
	Misplaced [Length]string
	// Excluded holds the gray letters. A gray letter that is also green or
	// yellow limits how many copies the word has, which is how Wordle
	// reports a repeated letter: see maxCount.
	Excluded string
}

// Match reports whether w (lowercase, Length letters) fits c.
func (c Constraints) Match(w string) bool {
	if len(w) != Length {
		return false
	}
	for i := range Length {
		if c.Correct[i] != 0 && w[i] != c.Correct[i] {
			return false
		}
		for _, l := range []byte(c.Misplaced[i]) {
			if w[i] == l || strings.IndexByte(w, l) < 0 {
				return false
			}
		}
	}
	for _, l := range []byte(c.Excluded) {
		if strings.Count(w, string(l)) > c.maxCount(l) {
			return false
		}
	}
	return true
}

// maxCount is the most copies of a gray letter l the word can contain.
//
// In the guess where l came back gray, the word had exactly as many copies
// as that guess had green or yellow l's, each at a different position. Those
// positions are all among the green and yellow boxes holding l, so their
// number is an upper bound. It can't be exact without knowing which guess
// each mark came from: a green L and a yellow L are one copy if they came
// from different guesses but two if from the same one.
func (c Constraints) maxCount(l byte) int {
	n := 0
	for i := range Length {
		if c.Correct[i] == l || strings.IndexByte(c.Misplaced[i], l) >= 0 {
			n++
		}
	}
	return n
}

// IsMisplaced reports whether l was marked yellow anywhere.
func (c Constraints) IsMisplaced(l byte) bool {
	for _, m := range c.Misplaced {
		if strings.IndexByte(m, l) >= 0 {
			return true
		}
	}
	return false
}

// Solve returns the words that match c, best guesses first: words whose
// distinct letters are most common among the candidates, so a guess rules
// out as many of them as possible. Ties are alphabetical.
func Solve(c Constraints, words []string) []string {
	var out []string
	for _, w := range words {
		if c.Match(w) {
			out = append(out, w)
		}
	}

	var freq [26]int
	for _, w := range out {
		var seen [26]bool
		for _, l := range []byte(w) {
			if !seen[l-'a'] {
				seen[l-'a'] = true
				freq[l-'a']++
			}
		}
	}
	score := func(w string) int {
		var seen [26]bool
		s := 0
		for _, l := range []byte(w) {
			if !seen[l-'a'] {
				seen[l-'a'] = true
				s += freq[l-'a']
			}
		}
		return s
	}
	slices.SortStableFunc(out, func(a, b string) int {
		return cmp.Or(cmp.Compare(score(b), score(a)), cmp.Compare(a, b))
	})
	return out
}
