package shader

import (
	"fmt"
	"os"
	"strings"

	config "github.com/faramozzayw/bevy-shader-explorer/config"
	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
	lo "github.com/samber/lo"
)

func ParseWGSLFile(
	config *config.Config, shaderFilePath string,
) (ShaderFile, error) {
	wgslCodeBytes, err := os.ReadFile(shaderFilePath)
	if err != nil {
		return ShaderFile{}, fmt.Errorf("read source: %w", err)
	}
	return ParseSource(config, Source{Path: shaderFilePath, Text: string(wgslCodeBytes)})
}

func ParseSource(config *config.Config, source Source) (ShaderFile, error) {
	parsed, err := parserFor(source.Path).Parse(source)
	if err != nil {
		return ShaderFile{}, err
	}
	shaderFile, err := (ShaderFileEnricher{}).Enrich(parsed, FileMetadata{Config: config, Path: source.Path})
	if err != nil {
		return ShaderFile{}, err
	}
	shaderFile.SourceText = source.Text
	return shaderFile, nil
}

func (shaderFile ShaderFile) CollectedShaderDefs() []string {
	existing := shaderFile.ShaderDefs
	shaderDefs := make([]string, 0, len(existing))
	seen := make(map[string]bool)
	add := func(defs []DefResult) {
		for _, def := range defs {
			for _, name := range def.Condition.Atoms {
				if !seen[name] {
					seen[name] = true
					shaderDefs = append(shaderDefs, name)
				}
			}
		}
	}
	for _, def := range existing {
		for _, name := range document.NewShaderCondition(def).Atoms {
			if !seen[name] {
				seen[name] = true
				shaderDefs = append(shaderDefs, name)
			}
		}
	}
	for _, item := range shaderFile.Consts {
		add(item.ShaderDefs)
	}
	for _, item := range shaderFile.Bindings {
		add(item.ShaderDefs)
	}
	for _, item := range shaderFile.Structures {
		add(item.ShaderDefs)
		for _, field := range item.Fields {
			add(field.ShaderDefs)
		}
	}
	for _, item := range shaderFile.Functions {
		add(item.ShaderDefs)
		for _, param := range item.Params {
			add(param.ShaderDefs)
		}
	}
	return shaderDefs
}

func (shaderFile *ShaderFile) ResolveTypeLinks(declaredImportPaths map[string]string) {
	importsMap := make(map[string]string)
	structuresList := lo.Map(shaderFile.Structures, func(v Structure, _ int) string {
		return v.Name
	})

	for key, paths := range shaderFile.DeclaredImports {
		if len(paths) == 0 {
			continue
		}
		fullPath := paths[0]

		var longestMatch string
		for module := range declaredImportPaths {
			if strings.HasPrefix(fullPath, module) {
				if len(module) > len(longestMatch) {
					longestMatch = module
				}
			}
		}

		if longestMatch != "" {
			importsMap[key] = declaredImportPaths[longestMatch]
		}
	}

	for i := range shaderFile.Structures {
		for j := range shaderFile.Structures[i].Fields {
			shaderFile.Structures[i].Fields[j].TypeInfo.ResolveTypeLink(importsMap, structuresList)
		}
	}

	for i := range shaderFile.Consts {
		shaderFile.Consts[i].TypeInfo.ResolveTypeLink(importsMap, structuresList)
	}

	for i := range shaderFile.Bindings {
		shaderFile.Bindings[i].TypeInfo.ResolveTypeLink(importsMap, structuresList)
	}

	for i := range shaderFile.Functions {
		for j := range shaderFile.Functions[i].Params {
			shaderFile.Functions[i].Params[j].TypeInfo.ResolveTypeLink(importsMap, structuresList)
		}

		shaderFile.Functions[i].ReturnTypeInfo.ResolveTypeLink(importsMap, structuresList)
	}
}
