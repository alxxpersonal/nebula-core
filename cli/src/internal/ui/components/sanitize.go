package components

import (
	"regexp"
	"strings"
	"unicode"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

var bidiControls = map[rune]struct{}{
	'\u202a': {},
	'\u202b': {},
	'\u202c': {},
	'\u202d': {},
	'\u202e': {},
	'\u2066': {},
	'\u2067': {},
	'\u2068': {},
	'\u2069': {},
	'\u200e': {},
	'\u200f': {},
}

// SanitizeText strips control characters and ANSI escape sequences from display strings.
func SanitizeText(input string) string {
	if input == "" {
		return input
	}
	cleaned := ansiPattern.ReplaceAllString(input, "")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if _, ok := bidiControls[r]; ok {
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, cleaned)
}
