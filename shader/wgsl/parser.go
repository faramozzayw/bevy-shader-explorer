package wgsl

import (
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
	return bevy.DefinitionBlocks(source), nil
}
func (language) ParserSource(source string) string { return bevy.MaskDirectives(source) }
func (language) ParseDeclarations(source, parserSource string, comments map[int]string, definitions []extract.ShaderDefBlock) (extract.Result, error) {
	return extract.ParseWithParser(source, parserSource, comments, definitions, syntax.Parse)
}
