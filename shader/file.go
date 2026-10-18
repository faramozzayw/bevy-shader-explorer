package shader

import (
	"fmt"
	"os"
	"sort"

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
	return ParseSource(config, document.NewSource(shaderFilePath, string(wgslCodeBytes)))
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
	shaderDefs := shaderFile.collectShaderDefs(true)
	return shaderDefs
}

// CollectedDeclarationShaderDefs returns definitions attached to declarations and their
// members. It intentionally excludes definitions that only occur in source
// control-flow, so declaration pages do not show unusable toggles.
func (shaderFile ShaderFile) CollectedDeclarationShaderDefs() []string {
	return shaderFile.collectShaderDefs(false)
}

func (shaderFile ShaderFile) collectShaderDefs(includeExisting bool) []string {
	capacity := len(shaderFile.ShaderDefs)
	if !includeExisting {
		capacity = 0
	}
	shaderDefs := make([]string, 0, capacity)
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
	if includeExisting {
		for _, def := range shaderFile.ShaderDefs {
			for _, name := range document.NewShaderCondition(def).Atoms {
				if !seen[name] {
					seen[name] = true
					shaderDefs = append(shaderDefs, name)
				}
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

	names := make([]string, 0, len(shaderFile.DeclaredImports))
	for key := range shaderFile.DeclaredImports {
		names = append(names, key)
	}
	sort.Strings(names)

	for _, key := range names {
		paths := shaderFile.DeclaredImports[key]
		for index, fullPath := range paths {
			link, ok := document.ResolveModuleLink(fullPath, declaredImportPaths)
			ref := document.ImportRef{Name: key, Path: fullPath}
			if ok {
				ref.Link = link + "#" + key
				if index == 0 {
					importsMap[key] = link
				}
			}
			shaderFile.ResolvedImports = append(shaderFile.ResolvedImports, ref)
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

func (shaderFile *ShaderFile) AttachUsage(usedBy map[string][]document.UsageRef) {
	if shaderFile.ImportPath == nil {
		return
	}
	modulePath := *shaderFile.ImportPath
	prefix := modulePath + "::"

	for i := range shaderFile.Consts {
		shaderFile.Consts[i].UsedBy, shaderFile.Consts[i].UsedBySummary = document.ResolveUsage(usedBy, prefix+shaderFile.Consts[i].Name, modulePath)
	}
	for i := range shaderFile.Bindings {
		shaderFile.Bindings[i].UsedBy, shaderFile.Bindings[i].UsedBySummary = document.ResolveUsage(usedBy, prefix+shaderFile.Bindings[i].Name, modulePath)
	}
	for i := range shaderFile.Structures {
		shaderFile.Structures[i].UsedBy, shaderFile.Structures[i].UsedBySummary = document.ResolveUsage(usedBy, prefix+shaderFile.Structures[i].Name, modulePath)
	}
	for i := range shaderFile.Functions {
		shaderFile.Functions[i].UsedBy, shaderFile.Functions[i].UsedBySummary = document.ResolveUsage(usedBy, prefix+shaderFile.Functions[i].Name, modulePath)
	}
}
