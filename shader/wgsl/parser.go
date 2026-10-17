// Package wgsl parses WGSL shader source.
package wgsl

import (
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
			definitions := conditions.Blocks(source)
			ranges, err := syntax.FunctionRanges(source, MaskDirectives(source))
			if err != nil {
				return nil, err
			}
			filtered := definitions[:0]
			for _, definition := range definitions {
				if !insideFunction(definition.IfLine, ranges) {
					filtered = append(filtered, definition)
				}
			}
			return filtered, nil
		},
		ParserSource: MaskDirectives,
		ParseDeclarations: func(source, parserSource string, comments map[int]string, definitions []extract.ShaderDefBlock) (extract.Result, error) {
			return extract.ParseWithParser(source, parserSource, comments, definitions, syntax.Parse)
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
