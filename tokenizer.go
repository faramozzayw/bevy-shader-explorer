package main

import (
	"unicode"
	"unicode/utf8"
)

// TokenKind represents the type of token.
type TokenKind int

const (
	Identifier TokenKind = iota
	Other
	Whitespace
)

// Token represents a lexical token with its kind, text, and position.
type Token struct {
	Kind TokenKind
	Text string
	Pos  int
}

// Tokenizer processes an input string and produces a sequence of tokens.
type Tokenizer struct {
	Tokens []Token
}

// NewTokenizer creates a new Tokenizer instance, tokenizing the input string.
// If emitWhitespace is true, whitespace tokens are included in the output.
func NewTokenizer(src string, emitWhitespace bool) *Tokenizer {
	var tokens []Token
	var currentTokenStart int
	var currentTokenKind *TokenKind
	var quotedToken bool

	for i := 0; i < len(src); {
		r, width := utf8.DecodeRuneInString(src[i:])
		nextIndex := i + width

		if r == '"' {
			quotedToken = !quotedToken
			i = nextIndex
			continue
		}

		if currentTokenKind != nil {
			switch *currentTokenKind {
			case Identifier:
				if quotedToken || isIdentContinue(r) {
					i = nextIndex
					continue
				}
				if r == ':' && nextIndex < len(src) {
					nextRune, nextWidth := utf8.DecodeRuneInString(src[nextIndex:])
					if nextRune == ':' {
						i = nextIndex + nextWidth
						continue
					}
				}
				tokens = append(tokens, Token{
					Kind: Identifier,
					Text: src[currentTokenStart:i],
					Pos:  currentTokenStart,
				})
			case Whitespace:
				if unicode.IsSpace(r) {
					i = nextIndex
					continue
				}
				tokens = append(tokens, Token{
					Kind: Whitespace,
					Text: src[currentTokenStart:i],
					Pos:  currentTokenStart,
				})
			}
			currentTokenKind = nil
			currentTokenStart = i
		}

		if quotedToken || isIdentStart(r) {
			kind := Identifier
			currentTokenKind = &kind
			currentTokenStart = i
		} else if !unicode.IsSpace(r) {
			tokens = append(tokens, Token{
				Kind: Other,
				Text: string(r),
				Pos:  i,
			})
		} else if emitWhitespace {
			kind := Whitespace
			currentTokenKind = &kind
			currentTokenStart = i
		}

		i = nextIndex
	}

	if currentTokenKind != nil {
		switch *currentTokenKind {
		case Identifier:
			tokens = append(tokens, Token{
				Kind: Identifier,
				Text: src[currentTokenStart:],
				Pos:  currentTokenStart,
			})
		case Whitespace:
			tokens = append(tokens, Token{
				Kind: Whitespace,
				Text: src[currentTokenStart:],
				Pos:  currentTokenStart,
			})
		}
	}

	return &Tokenizer{Tokens: tokens}
}

// isIdentStart checks if a rune is a valid identifier start character.
func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

// isIdentContinue checks if a rune is a valid identifier continuation character.
func isIdentContinue(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
