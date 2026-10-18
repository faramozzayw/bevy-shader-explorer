// Package parser coordinates language-specific shader parsing.
package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/faramozzayw/bevy-shader-explorer/shader/conditions"
	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
	"github.com/faramozzayw/bevy-shader-explorer/shader/extract"
	"github.com/faramozzayw/bevy-shader-explorer/shader/imports"
	"github.com/faramozzayw/bevy-shader-explorer/shader/source"
)

type Language struct {
	// Definitions are the blocks that may be attached to declaration models.
	// SourceDefinitions may additionally include statement-level conditions
	// that are intentionally omitted from declaration extraction.
	Definitions       func(string) ([]conditions.DefinitionBlock, error)
	SourceDefinitions func(string) ([]conditions.DefinitionBlock, error)
	ParserSource      func(string) string
	ParseDeclarations func(string, string, map[int]string, []extract.ShaderDefBlock) (extract.Result, error)
}

func Parse(sourceText string, language Language) (document.File, error) {
	normalized := source.New(sourceText).Text
	definitions, err := language.Definitions(normalized)
	if err != nil {
		return document.File{}, err
	}
	sourceDefinitions := definitions
	if language.SourceDefinitions != nil {
		sourceDefinitions, err = language.SourceDefinitions(normalized)
		if err != nil {
			return document.File{}, err
		}
	}
	inputs := make([]extract.ShaderDefBlock, 0, len(definitions))
	shaderDefBlocks := make([]document.ShaderDefBlock, 0, len(sourceDefinitions))
	for _, definition := range sourceDefinitions {
		condition := document.NewShaderCondition(definition.Name)
		shaderDefBlocks = append(shaderDefBlocks, document.ShaderDefBlock{
			Condition: condition,
			Negated:   definition.Negated, IfLine: definition.IfLine,
			ElseLine: definition.ElseLine, EndifLine: definition.EndifLine,
		})
	}
	for _, definition := range definitions {
		condition := document.NewShaderCondition(definition.Name)
		inputs = append(inputs, extract.ShaderDefBlock{
			Condition: condition,
			Negated:   definition.Negated, IfdefLine: definition.IfLine,
			ElseLine: definition.ElseLine, EndifLine: definition.EndifLine,
		})
	}
	declarations, err := language.ParseDeclarations(normalized, language.ParserSource(normalized), ExtractComments(strings.Split(normalized, "\n")), inputs)
	if err != nil {
		return document.File{}, fmt.Errorf("parse declarations: %w", err)
	}
	declaredImports, err := imports.ExtractAll(normalized)
	if err != nil {
		return document.File{}, fmt.Errorf("extract imports: %w", err)
	}
	return document.File{
		ImportPath: ExtractImportPath(normalized), Consts: declarations.Consts,
		Bindings: declarations.Bindings, Functions: declarations.Functions,
		Structures: declarations.Structures, DeclaredImports: declaredImports,
		ShaderDefs: definitionNames(definitions), SourceShaderDefs: definitionNames(sourceDefinitions), ShaderDefBlocks: shaderDefBlocks,
	}, nil
}

func ExtractComments(lines []string) map[int]string {
	comments := make(map[int]string)
	commentBuffer := []string{}
	isCollectingComment := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "// TODO:") {
			continue
		}
		if strings.Contains(trimmed, "/*") {
			isCollectingComment = true
			commentBuffer = append(commentBuffer, strings.TrimSpace(strings.Replace(trimmed, "/*", "", 1)))
			if strings.Contains(trimmed, "*/") {
				isCollectingComment = false
				last := len(commentBuffer) - 1
				commentBuffer[last] = strings.TrimSpace(strings.Split(commentBuffer[last], "*/")[0])
				comments[i+1] = strings.Join(commentBuffer, "\n")
				commentBuffer = []string{}
			}
		} else if isCollectingComment {
			if strings.Contains(trimmed, "*/") {
				commentBuffer = append(commentBuffer, strings.TrimSpace(strings.Split(trimmed, "*/")[0]))
				isCollectingComment = false
				comments[i+1] = strings.Join(commentBuffer, "\n")
				commentBuffer = []string{}
			} else {
				commentBuffer = append(commentBuffer, strings.TrimSpace(strings.TrimPrefix(trimmed, "*")))
			}
		} else if strings.HasPrefix(trimmed, "//") {
			comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "///"))
			comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))
			comments[i+1] = comment
		} else if len(commentBuffer) > 0 {
			comments[i+1] = strings.Join(commentBuffer, "\n")
			commentBuffer = []string{}
		}
	}
	return comments
}

func ExtractImportPath(code string) *string {
	match := regexp.MustCompile(`#define_import_path\s+(.*)`).FindStringSubmatch(code)
	if len(match) > 1 {
		value := strings.TrimSpace(match[1])
		return &value
	}
	return nil
}

func definitionNames(definitions []conditions.DefinitionBlock) []string {
	result := make([]string, 0, len(definitions))
	seen := make(map[string]bool)
	for _, definition := range definitions {
		for _, name := range document.NewShaderCondition(definition.Name).Atoms {
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}
