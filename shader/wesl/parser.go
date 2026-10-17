// Package wesl parses WESL shader source.
package wesl

import (
	"fmt"

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
		ParserSource: func(source string) string { return source },
		ParseDeclarations: func(source, parserSource string, comments map[int]string, definitions []extract.ShaderDefBlock) (extract.Result, error) {
			return extract.ParseWithParser(source, parserSource, comments, definitions, syntax.ParseWESL)
		},
	})
}

func insideFunction(line int, ranges []syntax.LineRange) bool {
	for _, function := range ranges {
		if line >= function.Start && line <= function.End {
			return true
		}
	}
	return false
}
