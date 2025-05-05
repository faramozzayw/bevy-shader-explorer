package main

import (
	"fmt"
	"maps"
	"regexp"
	"strings"
)

var lastWordPattern = regexp.MustCompile(`\w+$`)

type PeekableTokenizer struct {
	tokens []Token
	pos    int
}

func NewPeekableTokenizer(tokens []Token) *PeekableTokenizer {
	return &PeekableTokenizer{tokens: tokens}
}

func (p *PeekableTokenizer) Peek() *Token {
	if p.pos < len(p.tokens) {
		return &p.tokens[p.pos]
	}
	return nil
}

func (p *PeekableTokenizer) Next() *Token {
	if p.pos < len(p.tokens) {
		tok := &p.tokens[p.pos]
		p.pos++
		return tok
	}
	return nil
}

type DeclaredImports = map[string][]string

func ExtractAllImports(normalizedCode string) (map[string][]string, error) {
	blocks := extractImportBlocks(normalizedCode)
	declaredImports := make(map[string][]string)

	for _, block := range blocks {
		declared, err := parseImports(block)
		if err != nil {
			return nil, err
		}

		maps.Copy(declaredImports, declared)

	}

	return declaredImports, nil
}

func extractImportBlocks(src string) []string {
	var (
		blocks []string
		lines  = strings.Split(src, "\n")
		buffer strings.Builder
		depth  int
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#import ") {
			buffer.Reset()
			buffer.WriteString(trimmed)
			depth = strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

			if depth == 0 {
				blocks = append(blocks, buffer.String())
			}
			continue
		}

		if depth > 0 {
			depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			buffer.WriteString(trimmed)

			if depth == 0 {
				blocks = append(blocks, buffer.String())
			}
		}
	}

	return blocks
}

func parseImports(importString string) (DeclaredImports, error) {
	declaredImports := make(map[string][]string)
	tokens := NewPeekableTokenizer(NewTokenizer(importString, false).Tokens)

	if tok := tokens.Next(); tok == nil || !(tok.Kind == Other && tok.Text == "#") {
		pos := 0
		if tok != nil {
			pos = tok.Pos
		} else {
			pos = len(importString)
		}
		return nil, fmt.Errorf("expected `#import` at position %d", pos)
	}

	if tok := tokens.Next(); tok == nil || !(tok.Kind == Identifier && tok.Text == "import") {
		pos := len(importString)
		if tok != nil {
			pos = tok.Pos
		}
		return nil, fmt.Errorf("expected `#import` at position %d", pos)
	}

	var (
		stack   []string
		current string
		asName  string
	)

	for {
		switch tok := tokens.Peek(); {
		case tok == nil:
			return declaredImports, nil
		case tok.Kind == Identifier:
			current += tok.Text
			tokens.Next()

			peek := tokens.Peek()

			if peek == nil {
				usedName := lastWordPattern.FindStringSubmatch(tok.Text)[0]
				declaredImports[usedName] = append(declaredImports[usedName], tok.Text)
				return declaredImports, nil
			}

			if peek.Kind == Identifier && peek.Text == "as" {
				pos := peek.Pos
				tokens.Next()
				ident := tokens.Next()
				if ident == nil || ident.Kind != Identifier {
					return nil, fmt.Errorf("expected identifier after `as` at position %d", pos)
				}
				asName = ident.Text
			}

			if peek.Kind == Identifier {
				stack = append(stack, current+"::")
				current = ""
				asName = ""
			}

			continue

		case tok.Kind == Other && tok.Text == "{":
			if !strings.HasSuffix(current, "::") {
				return nil, fmt.Errorf("open brace must follow `::` at position %d", tok.Pos)
			}
			stack = append(stack, current)
			current = ""
			asName = ""

		case tok.Kind == Other && (tok.Text == "," || tok.Text == "}" || tok.Text == "\n"):
			if current != "" {
				usedName := asName
				if usedName == "" {
					if parts := strings.Split(current, "::"); len(parts) > 0 {
						usedName = parts[len(parts)-1]
					} else {
						usedName = current
					}
				}
				full := strings.Join(stack, "") + current
				declaredImports[usedName] = append(declaredImports[usedName], full)
				current = ""
				asName = ""
			}

			if tok.Text == "}" {
				if len(stack) == 0 {
					return nil, fmt.Errorf("close brace without open at position %d", tok.Pos)
				}
				stack = stack[:len(stack)-1]
			}

			if tokens.Peek() == nil {
				break
			}

		case tok.Kind == Other && tok.Text == ";":
			tokens.Next()
			if peek := tokens.Peek(); peek != nil {
				return nil, fmt.Errorf("unexpected token after ';' at position %d", peek.Pos)
			}

		case tok.Kind == Other:
			return nil, fmt.Errorf("unexpected token at position %d", tok.Pos)

		case tok.Kind == Whitespace:
			panic("whitespace tokens should not exist with emitWhitespace = false")
		}

		tokens.Next()
	}
}
