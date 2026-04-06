// Package tokenizer provides token counting utilities used internally
// by Dory's chunking strategies. It is not part of the public API.
package tokenizer

import (
	"strings"
	"unicode"
)

// Tokenize splits text into lowercase tokens by whitespace and punctuation
// boundaries. Punctuation characters become their own tokens.
func Tokenize(text string) []string {
	if text == "" {
		return nil
	}

	var tokens []string
	for _, word := range strings.Fields(text) {
		tokens = append(tokens, splitPunctuation(word)...)
	}
	return tokens
}

// splitPunctuation splits a word into tokens, separating leading and trailing
// punctuation as individual tokens.
func splitPunctuation(word string) []string {
	var tokens []string
	var current []rune

	for _, r := range word {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if len(current) > 0 {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = current[:0]
			}
			tokens = append(tokens, string(r))
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}
	return tokens
}

// Count returns the exact number of tokens in text.
func Count(text string) int {
	return len(Tokenize(text))
}

// CountApprox returns a fast approximation of the token count based on the
// assumption that the average English token is about 4 characters.
func CountApprox(text string) int {
	if len(text) == 0 {
		return 0
	}
	return len(text) / 4
}
