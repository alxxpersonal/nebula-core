package components

import (
	"regexp"
	"strings"
	"unicode"
)

var ansiCsiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
var ansiOscPattern = regexp.MustCompile(`(?s)\x1b\].*?(?:\x07|\x1b\\)`)
var ansiDcsPattern = regexp.MustCompile(`(?s)\x1bP.*?\x1b\\`)
var ansiApcPattern = regexp.MustCompile(`(?s)\x1b_.*?\x1b\\`)
var ansiPmPattern = regexp.MustCompile(`(?s)\x1b\^.*?\x1b\\`)
var ansiSosPattern = regexp.MustCompile(`(?s)\x1bX.*?\x1b\\`)

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
	cleaned := ansiCsiPattern.ReplaceAllString(input, "")
	cleaned = ansiOscPattern.ReplaceAllString(cleaned, "")
	cleaned = ansiDcsPattern.ReplaceAllString(cleaned, "")
	cleaned = ansiApcPattern.ReplaceAllString(cleaned, "")
	cleaned = ansiPmPattern.ReplaceAllString(cleaned, "")
	cleaned = ansiSosPattern.ReplaceAllString(cleaned, "")
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

// SanitizeOneLine removes control characters and collapses whitespace to keep a single line.
func SanitizeOneLine(input string) string {
	cleaned := SanitizeText(input)
	if cleaned == "" {
		return cleaned
	}
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	return strings.TrimSpace(cleaned)
}
