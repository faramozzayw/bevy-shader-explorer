package wesl

import (
	"fmt"

	"github.com/faramozzayw/bevy-shader-explorer/shader/bevy"
	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
	"github.com/faramozzayw/bevy-shader-explorer/shader/extract"
	"github.com/faramozzayw/bevy-shader-explorer/shader/parser"
	"github.com/faramozzayw/bevy-shader-explorer/shader/syntax"
)

type Parser struct{}

func (Parser) Parse(source document.Source) (document.File, error) {
	return parser.Parse(source.Text, language{})
}

type language struct{}

func (language) Definitions(source string) ([]bevy.DefinitionBlock, error) {
	blocks, err := syntax.WESLConditionalBlocks(source)
	if err != nil {
		return nil, fmt.Errorf("extract WESL shader definitions: %w", err)
	}
	definitions := make([]bevy.DefinitionBlock, 0, len(blocks))
	for _, block := range blocks {
		definitions = append(definitions, bevy.DefinitionBlock{Name: block.Name, Negated: block.Negated, IfLine: block.IfLine, ElseLine: block.ElseLine, EndifLine: block.EndifLine})
	}
	return definitions, nil
}
func (language) ParserSource(source string) string { return source }
func (language) ParseDeclarations(source, parserSource string, comments map[int]string, definitions []extract.ShaderDefBlock) (extract.Result, error) {
	return extract.ParseWithParser(source, parserSource, comments, definitions, syntax.ParseWESL)
}
