package wgsl

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	config "main/config"
	utils "main/utils"

	"github.com/aymerick/raymond"
	lo "github.com/samber/lo"
)

func ParseWGSLFile(
	config *config.Config, wgslFilePath string) WgslFile {
	wgslCodeBytes, err := os.ReadFile(wgslFilePath)
	if err != nil {
		log.Fatal(err)
	}
	normalizedCode := strings.ReplaceAll(string(wgslCodeBytes), "\n\r", "\n")

	basename := filepath.Base(wgslFilePath)
	filename := strings.TrimSuffix(basename, ".wgsl")
	originalDir := filepath.Dir(wgslFilePath)
	dir := utils.DedupPathParts(strings.ReplaceAll(originalDir, "src/", ""))

	innerPath, err := filepath.Rel(config.SourcePath, dir)
	if err != nil {
		log.Fatal(err)
	}
	wgslPath := utils.DedupPathParts(filepath.Join(innerPath, filename)) + ".html"

	declaredImports, err := ExtractAllImports(normalizedCode)
	if err != nil {
		log.Fatal(err)
	}

	lineComments := extractComments(strings.Split(normalizedCode, "\n"))
	shaderDefs := extractShaderDefsBlocks(normalizedCode)
	importPath := extractImportPath(normalizedCode)
	consts := extractConsts(normalizedCode, lineComments, shaderDefs)
	structures := extractStructures(normalizedCode, lineComments, shaderDefs)
	functions := extractFunctions(normalizedCode, lineComments, shaderDefs)
	bindings := extractBindings(normalizedCode, lineComments, shaderDefs)
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

func extractShaderDefsBlocks(code string) []ShaderDefBlock {
	lines := strings.Split(code, "\n")
	var blocks []ShaderDefBlock
	var stack []ShaderDefBlock

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		if strings.HasPrefix(trimmed, "#ifdef") {
			stack = append(stack, ShaderDefBlock{
				DefName:   strings.TrimSpace(trimmed[6:]),
				IfdefLine: lineNum,
			})
		} else if strings.HasPrefix(trimmed, "#else") {
			if len(stack) > 0 {
				current := &stack[len(stack)-1]
				if current.ElseLine == nil {
					current.ElseLine = &lineNum
				}
			}
		} else if strings.HasPrefix(trimmed, "#endif") {
			if len(stack) > 0 {
				current := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				current.EndifLine = lineNum
				blocks = append(blocks, current)
			}
		}
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].IfdefLine < blocks[j].IfdefLine
	})

	return blocks
}

func extractConsts(normalizedCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []Const {
	matches := constPattern.FindAllStringSubmatch(normalizedCode, -1)
	var results []Const
	for _, match := range matches {
		name, typ, value := match[1], match[2], match[3]

		// FIXME:
		// lineNumber := getLineNumber(normalizedCode, match[0])
		lineNumber := 0
		thisShaderDefs := getShaderDefsByLine(shaderDefs, lineNumber)

		// If type is not provided, infer it based on value
		if typ == "" {
			if matched, _ := regexp.MatchString(`^\d+\.\d+`, value); matched {
				typ = "AbstractFloat"
			} else if vecPattern.MatchString(value) {
				typ = vecPattern.FindStringSubmatch(value)[1]
			} else if matched, _ := regexp.MatchString(`\d+u$`, value); matched {
				typ = "u32"
			} else if matched, _ := regexp.MatchString(`\d+$`, value); matched {
				typ = "AbstractInt"
			}
		}
		typ = utils.RemovePath(typ)

		results = append(results, Const{
			LineNumber:    lineNumber,
			Name:          name,
			Value:         value,
			HasShaderDefs: len(thisShaderDefs) > 0,
			ShaderDefs:    thisShaderDefs,
			TypeInfo: TypeInfo{
				Type: typ,
			},
		})
	}

	return results
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

func getShaderDefsByLine(shaderDefs []ShaderDefBlock, lineNumber int) []DefResult {
	var results []DefResult

	for _, shaderDef := range shaderDefs {
		ifDefEndline := utils.ValueOrDefault(shaderDef.ElseLine, shaderDef.EndifLine)

		if lineNumber > shaderDef.IfdefLine && lineNumber < ifDefEndline {
			results = append(results, DefResult{
				DefName:    shaderDef.DefName,
				Branch:     "if",
				LineNumber: shaderDef.IfdefLine,
			})
		}

		// Check if the line number is between elseLine and endifLine
		if shaderDef.ElseLine != nil && lineNumber > *shaderDef.ElseLine && lineNumber < shaderDef.EndifLine {
			results = append(results, DefResult{
				DefName:    shaderDef.DefName,
				Branch:     "else",
				LineNumber: *shaderDef.ElseLine,
			})
		}
	}

	return results
}

func extractStructures(normalizedCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []Structure {
	matches := structurePattern.FindAllStringSubmatchIndex(normalizedCode, -1)
	var structures []Structure

	for _, match := range matches {
		name := normalizedCode[match[2]:match[3]]
		fieldsRaw := normalizedCode[match[4]:match[5]]

		commentStrip := regexp.MustCompile(`\/{1,3}.*`)
		cleanFields := commentStrip.ReplaceAllString(fieldsRaw, "")
		fields := parseNamedTypeString(cleanFields, normalizedCode, shaderDefs)

		lineNumber := getLineNumber(normalizedCode, match[0])
		comments := getItemComments(lineNumber, lineComments)
		shaderDefsThis := getShaderDefsByLine(shaderDefs, lineNumber)

		fieldsShaderDefs := lo.SomeBy(fields, func(field NamedType) bool {
			return field.HasShaderDefs
		})

		structures = append(structures, Structure{
			Name:             name,
			Fields:           fields,
			LineNumber:       lineNumber,
			Comment:          strings.Join(comments, "\n"),
			HasShaderDefs:    len(shaderDefsThis) > 0,
			HasFields:        len(fields) != 0,
			ShaderDefs:       shaderDefsThis,
			FieldsShaderDefs: fieldsShaderDefs,
		})
	}

	return structures
}

func extractFunctions(normalizedCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []Function {
	var functions []Function
	fullCode := normalizedCode

	matches := functionPattern.FindAllStringSubmatchIndex(fullCode, -1)

	for _, matchIdx := range matches {
		startIdx := matchIdx[0]
		endIdx := matchIdx[1]
		signature := strings.TrimSpace(fullCode[startIdx:endIdx])
		signature = strings.TrimSuffix(signature, "{")

		var stageAttr string
		var workgroupSize []string
		if stageMatch := shaderStagePattern.FindStringSubmatch(signature); len(stageMatch) > 1 {
			stageAttr = stageMatch[1]

			if stageAttr == "compute" {
				workgroupSizeMatch := strings.TrimSpace(workgroupSizePattern.FindStringSubmatch(signature)[1])
				workgroupSize = lo.Map(strings.Split(workgroupSizeMatch, ","), func(v string, _ int) string {
					return strings.TrimSpace(v)
				})
			}
		}

		var name, rawParams string
		var sigMatch []string

		if strings.Contains(signature, "->") {
			sigMatch = functionSigWithReturnTypePattern.FindStringSubmatch(signature)
		} else {
			sigMatch = functionSigWithoutReturnTypePattern.FindStringSubmatch(signature)
		}

		if len(sigMatch) > 0 {
			name = sigMatch[1]
			rawParams = sigMatch[2]
		}

		params := parseNamedTypeString(rawParams, fullCode, shaderDefs)
		returnType := "void"
		returnTypeAnnotations := make([]Annotation, 0)

		if rt := regexp.MustCompile(`->(.*)`).FindStringSubmatch(signature); len(rt) > 1 {
			trimmedRt := strings.TrimSpace(rt[1])

			matches := annotationPattern.FindAllStringSubmatchIndex(trimmedRt, -1)

			for _, match := range matches {
				returnTypeAnnotations = append(returnTypeAnnotations, Annotation{
					Name:  trimmedRt[match[2]:match[3]],
					Value: trimmedRt[match[4]:match[5]],
				})
			}

			if len(matches) > 0 {
				last := matches[len(matches)-1]
				returnType = strings.TrimSpace(trimmedRt[last[1]:])
			} else {
				returnType = trimmedRt
			}
		}

		lineNumber := getLineNumber(fullCode, startIdx)
		comments := getItemComments(lineNumber, lineComments)
		thisShaderDefs := getShaderDefsByLine(shaderDefs, lineNumber)

		functions = append(functions, Function{
			StageAttribute:   stageAttr,
			WorkgroupSize:    workgroupSize,
			HasWorkgroupSize: len(workgroupSize) > 0,
			Name:             name,
			LineNumber:       lineNumber,
			Params:           params,
			HasParams:        len(params) != 0,
			HasShaderDefs:    len(thisShaderDefs) > 0,
			ShaderDefs:       thisShaderDefs,
			Comment:          strings.Join(comments, "\n"),
			ReturnTypeInfo: TypeInfo{
				Type:        returnType,
				Annotations: returnTypeAnnotations,
			},
		})
	}

	return functions
}

func extractBindings(code string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []Binding {
	var bindings []Binding

	matches := bindingPattern.FindAllStringSubmatchIndex(code, -1)
	for _, matchIdx := range matches {
		match := code[matchIdx[0]:matchIdx[1]]
		submatches := bindingPattern.FindStringSubmatch(match)

		groupIndex := submatches[1]
		bindingIndex := submatches[2]
		bindingType := submatches[3]
		name := submatches[4]
		typeStr := utils.RemovePath(submatches[5])
		lineNumber := getLineNumber(code, matchIdx[0])
		thisShaderDefs := getShaderDefsByLine(shaderDefs, lineNumber)
		annotations := []Annotation{
			{
				Name:  "group",
				Value: groupIndex,
			},
			{
				Name:  "binding",
				Value: bindingIndex,
			},
		}

		bindings = append(bindings, Binding{
			LineNumber:    lineNumber,
			Name:          name,
			Annotations:   annotations,
			BindingType:   bindingType,
			HasShaderDefs: len(thisShaderDefs) > 0,
			ShaderDefs:    thisShaderDefs,
			TypeInfo: TypeInfo{
				Type:         typeStr,
				FullTypePath: submatches[5],
			},
		})
	}

	return bindings
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

func parseNamedTypeString(str, fullCode string, shaderDefs []ShaderDefBlock) []NamedType {
	str = strings.ReplaceAll(str, "\n", "")
	str = regexp.MustCompile(`#ifdef\s+\w+|#else|#endif`).ReplaceAllString(str, "")
	str = strings.TrimSpace(str)

	entries := utils.SplitParams(str)
	var result []NamedType

	for _, entry := range entries {
		annotations := make([]Annotation, 0)
		matches := annotationPattern.FindAllStringSubmatchIndex(entry, -1)

		for _, match := range matches {
			annotations = append(annotations, Annotation{
				Name:  entry[match[2]:match[3]],
				Value: entry[match[4]:match[5]],
			})
		}

		param := strings.ReplaceAll(entry, " ", "")
		if len(matches) > 0 {
			last := matches[len(matches)-1]
			param = strings.ReplaceAll(entry[last[1]:], " ", "")
		}

		splittedParam := strings.Split(param, ":")

		name := splittedParam[0]
		typ := utils.RemovePath(splittedParam[1])

		// Approximate the line number
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*:`)
		loc := re.FindStringIndex(fullCode)
		lineNumber := 0
		if loc != nil {
			lineNumber = getLineNumber(fullCode, loc[0])
		}

		shaderDefMatches := getShaderDefsByLine(shaderDefs, lineNumber)

		result = append(result, NamedType{
			Annotations:   annotations,
			Name:          name,
			HasShaderDefs: len(shaderDefMatches) > 0,
			ShaderDefs:    shaderDefMatches,
			TypeInfo: TypeInfo{
				Type:         typ,
				FullTypePath: typ,
			},
		})
	}

	return result
}

func getItemComments(lineNumber int, lineComments map[int]string) []string {
	var comments []string
	currentLine := lineNumber

	for currentLine > 0 && lineComments[currentLine-1] != "" {
		if comment, ok := lineComments[currentLine]; ok {
			comments = append(comments, comment)
		}
		currentLine--
	}

	if comment, ok := lineComments[currentLine]; ok {
		comments = append(comments, comment)
	}

	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}

	return comments
}

func getLineNumber(code string, matchIndex int) int {
	if matchIndex > len(code) {
		matchIndex = len(code)
	}
	codeBeforeMatch := code[:matchIndex]
	return strings.Count(codeBeforeMatch, "\n") + 1
}

func (typeInfo *TypeInfo) ResolveTypeLink(imports map[string]string, definedStructuresList []string) {
	if len(typeInfo.TypeLink) == 0 {
		typeInfo.TypeLink = utils.GetTypeLink(typeInfo.Type)
	}

	if len(typeInfo.TypeLink) != 0 {
		return
	}

	if len(typeInfo.FullTypePath) == 0 {
		typeInfo.FullTypePath = typeInfo.Type
	}

	importTarget := strings.Split(typeInfo.FullTypePath, "::")[0]

	if typeLink, ok := imports[importTarget]; ok {
		typeInfo.TypeLink = typeLink + "#" + typeInfo.Type
		typeInfo.TypeLinkBlank = true
		return
	}

	if slices.Contains(definedStructuresList, typeInfo.Type) {
		typeInfo.TypeLink = "#" + typeInfo.Type
		typeInfo.TypeLinkBlank = false
	}
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
	innerPath, err := filepath.Rel(config.SourcePath, dir)
	if err != nil {
		log.Fatal(err)
	}

	joinedPath := filepath.Join(innerPath, basename)

	baseURL, err := url.Parse(config.SourceGithubURL)
	if err != nil {
		log.Fatal(err)
	}

	return baseURL.ResolveReference(&url.URL{Path: joinedPath}).String()
}
