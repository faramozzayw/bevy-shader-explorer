// Package wesl parses WESL shader source.
package wesl

import (
	"fmt"
	"strings"

	"github.com/faramozzayw/bevy-shader-explorer/shader/conditions"
	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
	"github.com/faramozzayw/bevy-shader-explorer/shader/extract"
	"github.com/faramozzayw/bevy-shader-explorer/shader/parser"
	"github.com/faramozzayw/bevy-shader-explorer/shader/syntax"
)

type Parser struct{}

func (Parser) Parse(source document.Source) (document.File, error) {
	return parser.Parse(source.Text, parser.Language{
		Definitions: func(source string) ([]conditions.DefinitionBlock, error) {
			blocks, err := syntax.WESLConditionalBlocks(source)
			if err != nil {
				return nil, fmt.Errorf("extract WESL shader definitions: %w", err)
			}
			ranges, err := syntax.WESLFunctionRanges(source, source)
			if err != nil {
				return nil, err
			}
			definitions := make([]conditions.DefinitionBlock, 0, len(blocks))
			for _, block := range blocks {
				if insideFunction(block.IfLine, ranges) {
					continue
				}
				definitions = append(definitions, conditions.DefinitionBlock{Name: block.Name, Negated: block.Negated, IfLine: block.IfLine, ElseLine: block.ElseLine, EndifLine: block.EndifLine})
			}
			return definitions, nil
		},
		SourceDefinitions: func(source string) ([]conditions.DefinitionBlock, error) {
			blocks, err := syntax.WESLConditionalBlocks(source)
			if err != nil {
				return nil, fmt.Errorf("extract WESL source shader definitions: %w", err)
			}
			definitions := make([]conditions.DefinitionBlock, 0, len(blocks))
			for _, block := range blocks {
				definitions = append(definitions, conditions.DefinitionBlock{Name: block.Name, Negated: block.Negated, IfLine: block.IfLine, ElseLine: block.ElseLine, EndifLine: block.EndifLine})
			}
			return definitions, nil
		},
		ParserSource: maskFunctionAttributes,
		ParseDeclarations: func(source, parserSource string, comments map[int]string, definitions []extract.ShaderDefBlock) (extract.Result, error) {
			return extract.ParseWithParser(source, parserSource, comments, definitions, syntax.ParseWESL)
		},
	})
}

// maskFunctionAttributes keeps declaration-level WESL attributes visible to
// tree-sitter while removing statement-level attributes from its parser input.
// Some checked-in grammar versions recover poorly on those statements and can
// otherwise swallow subsequent top-level function declarations. The original
// source is still used for extraction and rendering.
func maskFunctionAttributes(source string) string {
	ranges, _ := syntax.WESLFunctionRanges(source, source)
	lines := strings.Split(source, "\n")
	for index := range lines {
		lineNumber := index + 1
		inside := false
		for _, function := range ranges {
			if lineNumber > function.Start && lineNumber <= function.End {
				inside = true
				break
			}
		}
		if !inside {
			continue
		}
		code := strings.SplitN(lines[index], "//", 2)[0]
		masked := []byte(lines[index])
		for offset := 0; offset < len(code); {
			at := strings.IndexByte(code[offset:], '@')
			if at < 0 {
				break
			}
			at += offset
			kind, _, end, ok := parseFunctionAttribute(code, at)
			if !ok {
				offset = at + 1
				continue
			}
			if kind == "if" || kind == "else" {
				for position := at; position < end && position < len(masked); position++ {
					if masked[position] != '\t' && masked[position] != '\n' {
						masked[position] = ' '
					}
				}
			}
			offset = end
		}
		lines[index] = string(masked)
	}
	return strings.Join(lines, "\n")
}

func parseFunctionAttribute(line string, at int) (kind, expression string, end int, ok bool) {
	rest := line[at:]
	if strings.HasPrefix(rest, "@else") {
		return "else", "", at + len("@else"), true
	}
	if !strings.HasPrefix(rest, "@if") && !strings.HasPrefix(rest, "@elif") {
		return "", "", at + 1, false
	}
	open := strings.IndexByte(rest, '(')
	if open < 0 {
		return "", "", at + 1, false
	}
	depth := 0
	for position := at + open; position < len(line); position++ {
		switch line[position] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return "if", line[at+open+1 : position], position + 1, true
			}
		}
	}
	return "", "", at + 1, false
}

func insideFunction(line int, ranges []syntax.LineRange) bool {
	for _, function := range ranges {
		if line >= function.Start && line <= function.End {
			return true
		}
	}
	return false
}
