package wgsl

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	config "main/config"
	utils "main/utils"
	"main/wgsl/bevy"
	"main/wgsl/extract"
	"main/wgsl/source"

	"github.com/aymerick/raymond"
	lo "github.com/samber/lo"
)

func ParseWGSLFile(
	config *config.Config, wgslFilePath string) WgslFile {
	wgslCodeBytes, err := os.ReadFile(wgslFilePath)
	if err != nil {
		log.Fatal(err)
	}
	normalizedCode := source.New(string(wgslCodeBytes)).Text

	basename := filepath.Base(wgslFilePath)
	extension := filepath.Ext(basename)
	filename := strings.TrimSuffix(basename, extension)
	originalDir := filepath.Dir(wgslFilePath)
	innerPath, err := filepath.Rel(config.SourcePath, originalDir)
	if err != nil {
		log.Fatal(err)
	}
	innerPath = strings.ReplaceAll(innerPath, "src"+string(filepath.Separator), "")
	wgslPath := utils.DedupPathParts(filepath.Join(innerPath, filename)) + ".html"

	declaredImports, err := bevy.ExtractAllImports(normalizedCode)
	if err != nil {
		log.Fatal(err)
	}

	lineComments := extractComments(strings.Split(normalizedCode, "\n"))
	definitions := bevy.DefinitionBlocks(normalizedCode)
	extractDefs := make([]extract.ShaderDefBlock, 0, len(definitions))
	for _, definition := range definitions {
		extractDefs = append(extractDefs, extract.ShaderDefBlock{DefName: definition.Name, IfdefLine: definition.IfLine, ElseLine: definition.ElseLine, EndifLine: definition.EndifLine})
	}
	parserCode := bevy.MaskDirectives(normalizedCode)
	parseDeclarations := extract.Parse
	if extension == ".wesl" {
		parserCode = normalizedCode
		parseDeclarations = extract.ParseWESL
	}
	declarations, err := parseDeclarations(normalizedCode, parserCode, lineComments, extractDefs)
	if err != nil {
		log.Fatal(err)
	}
	importPath := extractImportPath(normalizedCode)
	consts := declarations.Consts
	structures := declarations.Structures
	functions := declarations.Functions
	bindings := declarations.Bindings
	githubLink := GetGithubLink(config, originalDir, basename)

	wgslFile := WgslFile{
		Version:    config.Version,
		ImportPath: importPath,

		Consts:           consts,
		ConstsShaderDefs: anyShaderDefs(consts),
		NotEmptyConsts:   len(consts) != 0,

		Bindings:           bindings,
		BindingsShaderDefs: anyShaderDefs(bindings),
		NotEmptyBindings:   len(bindings) != 0,

		Functions:         functions,
		NotEmptyFunctions: len(functions) != 0,

		Structures:           structures,
		StructuresShaderDefs: anyShaderDefs(structures),
		NotEmptyStructures:   len(structures) != 0,
		DeclaredImports:      declaredImports,

		Filename:   basename,
		WgslPath:   wgslPath,
		GithubLink: githubLink,
		Link:       fmt.Sprintf("%s/%s", config.Version, wgslPath),
	}

	return wgslFile
}

func (wgslFile *WgslFile) ResolveTypeLinks(declaredImportPaths map[string]string) {
	importsMap := make(map[string]string)
	structuresList := lo.Map(wgslFile.Structures, func(v Structure, _ int) string {
		return v.Name
	})

	for key, paths := range wgslFile.DeclaredImports {
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

	for i := range wgslFile.Structures {
		for j := range wgslFile.Structures[i].Fields {
			wgslFile.Structures[i].Fields[j].TypeInfo.ResolveTypeLink(importsMap, structuresList)
		}
	}

	for i := range wgslFile.Consts {
		wgslFile.Consts[i].TypeInfo.ResolveTypeLink(importsMap, structuresList)
	}

	for i := range wgslFile.Bindings {
		wgslFile.Bindings[i].TypeInfo.ResolveTypeLink(importsMap, structuresList)

	}

	for i := range wgslFile.Functions {
		for j := range wgslFile.Functions[i].Params {
			wgslFile.Functions[i].Params[j].TypeInfo.ResolveTypeLink(importsMap, structuresList)
		}

		wgslFile.Functions[i].ReturnTypeInfo.ResolveTypeLink(importsMap, structuresList)
	}
}

func (wgslFile *WgslFile) GenerateWgslPage(compiledTemplate *raymond.Template, outputDir string) {
	fileOutputPath := strings.ReplaceAll(filepath.Join(outputDir, wgslFile.WgslPath), "src/", "")

	html, err := compiledTemplate.Exec(wgslFile)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(filepath.Dir(filepath.Join(outputDir, wgslFile.WgslPath)), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(fileOutputPath, []byte(html), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func extractComments(lines []string) map[int]string {
	lineComments := make(map[int]string)
	commentBuffer := []string{}
	isCollectingComment := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "// TODO:") {
			continue
		}

		// Handle multi-line comments
		if strings.Contains(trimmed, "/*") {
			isCollectingComment = true
			cleaned := strings.Replace(trimmed, "/*", "", 1)
			commentBuffer = append(commentBuffer, strings.TrimSpace(cleaned))

			// Multi-line comment ends on the same line
			if strings.Contains(trimmed, "*/") {
				isCollectingComment = false
				last := len(commentBuffer) - 1
				commentBuffer[last] = strings.TrimSpace(
					strings.Split(commentBuffer[last], "*/")[0],
				)
				lineComments[i+1] = strings.Join(commentBuffer, "\n")
				commentBuffer = []string{}
			}
		} else if isCollectingComment {
			if strings.Contains(trimmed, "*/") {
				cleaned := strings.Split(trimmed, "*/")[0]
				commentBuffer = append(commentBuffer, strings.TrimSpace(cleaned))
				isCollectingComment = false
				lineComments[i+1] = strings.Join(commentBuffer, "\n")
				commentBuffer = []string{}
			} else {
				// Remove leading '*' if present
				cleaned := strings.TrimPrefix(trimmed, "*")
				commentBuffer = append(commentBuffer, strings.TrimSpace(cleaned))
			}

		} else if strings.HasPrefix(trimmed, "//") {
			comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "///"))
			comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))
			lineComments[i+1] = comment
		} else {
			if len(commentBuffer) > 0 {
				lineComments[i+1] = strings.Join(commentBuffer, "\n")
				commentBuffer = []string{}
			}
		}
	}

	return lineComments
}

func extractImportPath(normalizedCode string) *string {
	re := regexp.MustCompile(`#define_import_path\s+(.*)`)
	match := re.FindStringSubmatch(normalizedCode)
	if len(match) > 1 {
		result := match[1]
		return &result
	}
	return nil
}

// checks if any item has shader definitions
func anyShaderDefs[T any](input []T) bool {
	for _, v := range input {
		val := reflect.ValueOf(v)
		field := val.FieldByName("HasShaderDefs")
		if field.IsValid() && field.Kind() == reflect.Bool && field.Bool() {
			return true
		}
	}
	return false
}

func GetGithubLink(config *config.Config, dir string, basename string) string {
	if config.SourceGithubURL == "" {
		return ""
	}
	repositoryRoot := config.SourceGithubRoot
	if repositoryRoot == "" {
		repositoryRoot = config.SourcePath
	}
	innerPath, err := filepath.Rel(repositoryRoot, dir)
	if err != nil {
		log.Fatal(err)
	}

	joinedPath := filepath.Join(config.SourceGithubSubpath, innerPath, basename)

	ref := config.SourceGithubRef
	if ref == "" {
		ref = "main"
	}
	baseURL, err := url.Parse(strings.TrimRight(config.SourceGithubURL, "/") + "/blob/" + url.PathEscape(ref) + "/")
	if err != nil {
		log.Fatal(err)
	}

	return baseURL.ResolveReference(&url.URL{Path: joinedPath}).String()
}
