// Package words provides gordle's embedded word lists.
//
// The lists are generated from SCOWL (see SCOWL-COPYRIGHT) by tools/genwords.
// To regenerate, point SCOWL at an extracted SCOWL release and run:
//
//	SCOWL=/path/to/scowl-2020.12.07 go generate ./internal/words
package words

//go:generate go run ../../tools/genwords -scowl $SCOWL -out . -exclude ../../tools/genwords/exclude.txt

import (
	_ "embed"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
)

var (
	//go:embed answers.txt
	answersTxt string
	//go:embed valid.txt
	validTxt string
)

// Length is the number of letters in every word.
const Length = 5

var (
	answers = strings.Fields(answersTxt)
	valid   = toSet(strings.Fields(validTxt))

	// dailyOrder is a fixed shuffle of answers so consecutive days don't
	// walk through the list alphabetically.
	dailyOrder = shuffled(len(answers), 0x60_4D_1E)
)

// epoch is day 1 of the daily puzzle.
var epoch = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// IsValid reports whether w is an acceptable guess.
func IsValid(w string) bool {
	_, ok := valid[w]
	return ok
}

// Count returns the number of possible answers.
func Count() int { return len(answers) }

// Answers returns a copy of the possible answers, in alphabetical order.
func Answers() []string { return slices.Clone(answers) }

func toSet(ws []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ws))
	for _, w := range ws {
		m[w] = struct{}{}
	}
	return m
}

// shuffled returns a permutation of [0, n) that depends only on seed. It uses
// its own Fisher-Yates over the PCG generator, whose output is fully
// specified, so the daily order can't change between Go releases.
func shuffled(n int, seed uint64) []int {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	r := rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)
	for i := n - 1; i > 0; i-- {
		j := int(r.Uint64() % uint64(i+1))
		p[i], p[j] = p[j], p[i]
	}
	return p
}
