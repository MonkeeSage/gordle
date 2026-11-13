// Command genwords builds gordle's embedded word lists from a SCOWL release.
//
// Usage:
//
//	go run ./tools/genwords -scowl path/to/scowl-2020.12.07 -out internal/words
//
// SCOWL groups words into size levels (10 = most common ... 95 = most obscure).
// Answers come from the common levels with inflected forms (plurals, past
// tenses, -ing forms) removed, so the hidden word is always a "headword".
// Valid guesses come from a much larger set so real words are rarely rejected.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var levels = []int{10, 20, 35, 40, 50, 55, 60, 70, 80, 95}

func main() {
	var (
		scowlDir   = flag.String("scowl", "", "path to an extracted SCOWL release (contains final/ and Copyright)")
		outDir     = flag.String("out", ".", "output directory")
		length     = flag.Int("len", 5, "word length")
		answersMax = flag.Int("answers-max", 35, "highest SCOWL level used for answers")
		validMax   = flag.Int("valid-max", 80, "highest SCOWL level used for valid guesses")
		exclude    = flag.String("exclude", "", "optional file of words never to use as answers")
	)
	flag.Parse()
	if *scowlDir == "" {
		log.Fatal("-scowl is required")
	}

	final := filepath.Join(*scowlDir, "final")
	common := readLevels(final, []string{"english", "american"}, *answersMax, *length)
	valid := readLevels(final, []string{"english", "american", "british"}, *validMax, *length)
	// Stems only come from reasonably common levels: the obscure levels
	// contain fragments like "sw" and "twee" that would make "swing" and
	// "tweed" look like inflections.
	stems := readLevels(final, []string{"english", "american", "british"}, 60, 0)

	excluded := map[string]bool{}
	if *exclude != "" {
		for w := range readWords(*exclude, 0) {
			excluded[w] = true
		}
	}

	var answers []string
	for w := range common {
		if !excluded[w] && !isInflection(w, stems) {
			answers = append(answers, w)
		}
	}
	slices.Sort(answers)

	guesses := make([]string, 0, len(valid))
	for w := range valid {
		guesses = append(guesses, w)
	}
	for _, w := range answers {
		if !valid[w] {
			guesses = append(guesses, w)
		}
	}
	slices.Sort(guesses)

	writeList(filepath.Join(*outDir, "answers.txt"), answers)
	writeList(filepath.Join(*outDir, "valid.txt"), guesses)
	copyFile(filepath.Join(*scowlDir, "Copyright"), filepath.Join(*outDir, "SCOWL-COPYRIGHT"))

	fmt.Printf("answers: %d words, valid guesses: %d words\n", len(answers), len(guesses))
}

// notInflections are words that look like inflections of a shorter word but
// are headwords in their own right.
var notInflections = map[string]bool{
	"thing": true, // not "the" + "ing"
	"bleed": true, "breed": true, "creed": true, "greed": true, "tweed": true,
	"naked": true,
}

// isInflection reports whether w looks like a regular inflection of a
// shorter word in stems, e.g. "dills" (dill), "homed" (home), "tries" (try),
// "acing" (ace).
func isInflection(w string, stems map[string]bool) bool {
	if notInflections[w] {
		return false
	}
	n := len(w)
	has := func(s string) bool { return len(s) >= 2 && stems[s] }

	switch {
	case strings.HasSuffix(w, "ss"):
		return false
	case strings.HasSuffix(w, "ies") || strings.HasSuffix(w, "ied"):
		if has(w[:n-3] + "y") {
			return true
		}
	}
	switch {
	case strings.HasSuffix(w, "s"):
		return has(w[:n-1]) || (strings.HasSuffix(w, "es") && has(w[:n-2]))
	case strings.HasSuffix(w, "ed"):
		return has(w[:n-1]) || has(w[:n-2])
	case strings.HasSuffix(w, "ing"):
		return has(w[:n-3]) || has(w[:n-3]+"e")
	}
	return false
}

func readLevels(dir string, kinds []string, maxLevel, length int) map[string]bool {
	words := map[string]bool{}
	for _, lvl := range levels {
		if lvl > maxLevel {
			break
		}
		for _, kind := range kinds {
			path := filepath.Join(dir, fmt.Sprintf("%s-words.%d", kind, lvl))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				continue
			}
			for w := range readWords(path, length) {
				words[w] = true
			}
		}
	}
	return words
}

// readWords returns the lowercase ASCII words in path. Capitalised words
// (proper nouns), words with apostrophes and accented words (SCOWL files are
// Latin-1) are skipped. A length of 0 accepts any length.
func readWords(path string, length int) map[string]bool {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	words := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		w := strings.TrimSpace(sc.Text())
		if (length == 0 || len(w) == length) && isLowerASCII(w) {
			words[w] = true
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
	return words
}

func isLowerASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}
	return true
}

func writeList(path string, words []string) {
	if err := os.WriteFile(path, []byte(strings.Join(words, "\n")+"\n"), 0o644); err != nil {
		log.Fatal(err)
	}
}

func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		log.Fatal(err)
	}
	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}
