package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/aymerick/raymond"
	. "github.com/samber/lo"
	progressbar "github.com/schollz/progressbar/v3"
)

const OUTPUT_DIR_ROOT = "./dist"

var CURRENT_VERSION = "release-0.15.3"
var bevyUrl = "https://github.com/bevyengine/bevy/tree/" + CURRENT_VERSION + "/"

var copyToPublic = []string{
	"styles.css",
	"favicon.ico",
	"search.js",
	"wgsl.png",
	"github-mark.png",
	"github-mark-white.png",
	"templates/search-result.hbs",
}

var PUBLIC_FOLDER = filepath.Join(OUTPUT_DIR_ROOT, "public")

//go:embed templates/wgsl-doc.hbs
var WGSL_DOC_TEMPLATE_SOURCE string

//go:embed templates/home.hbs
var HOME_DOC_TEMPLATE_SOURCE string

//go:embed wgpu-types.json
var wgpuTypesData []byte
var wgpuTypes map[string]string

var structurePattern = regexp.MustCompile(`struct\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{([^}]*)\}`)
var constPattern = regexp.MustCompile(`const\s+(\w+)\s{0,}(?::\s{0,}(.*))?=\s+(.*);`)
var namedTypeStringPattern = regexp.MustCompile(`^(?:@([^\s]+)\s+)?([a-zA-Z_]\w*):(.+)$`)
var typeStringPattern = regexp.MustCompile(`^(?:@([^\s]+)\s+)?(.+)$`)
var functionPattern = regexp.MustCompile(`(?m)(@[^;]*\s+)?(vertex|fragment|compute\s+)?\bfn\b\s+([a-zA-Z0-9_]+)[\s\S]*?\{`)
var functionSigWithReturnTypePattern = regexp.MustCompile(`\bfn\b\s+(\w+)\(([\s\S]+)?\)\s+->`)
var functionSigWithoutReturnTypePattern = regexp.MustCompile(`\bfn\b\s+(\w+)\(([\s\S]+)?\).*`)
var bindingPattern = regexp.MustCompile(`@group\((\d+)\)\s{0,}@binding\((\d+)\)\s{0,}var\s{0,}(?:<(.*?)>)?\s{0,}(\w+):\s{0,}(.*);`)
var shaderStagePattern = regexp.MustCompile(`@(vertex|fragment|compute)`)
var vecPattern = regexp.MustCompile(`(vec\d(?:<.*>))`)
var annotationPattern = regexp.MustCompile(`(?:@([^\s]+)\((.*?)\)){1,}`)

var sourcePath string

func main() {
	err := json.Unmarshal(wgpuTypesData, &wgpuTypes)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse wgpu-types.json: %v", err))
	}

	source := flag.String("source", "", "Source file path")
	file_filter := flag.String("filter", "*.wgsl", "Source file filter")
	flag.Parse()

	if *source == "" {
		log.Fatal("Error: 'source' is a required argument")
	}

	sourcePath = *source

	cmd := exec.Command("find", sourcePath, "-type", "f", "-name", *file_filter)
	stdout, err := cmd.Output()

	RegisterHelpers()
	RegisterPartials()

	if err != nil {
		log.Fatal(err)
	}

	filePaths := strings.Split(strings.TrimSpace(string(stdout)), "\n")

	var searchInfo []ShaderSearchableInfo

	declaredImportPaths := make(map[string]string)
	wgslFiles := make([]WgslFile, 0)

	parsing_bar := progressbar.Default(
		int64(len(filePaths)),
		"parsing WGSL files",
	)

	for _, filePath := range filePaths {

		wgslFile := parseWGSLFile(filePath)
		wgslFiles = append(wgslFiles, wgslFile)

		normalizedLink := NormalizeLink(wgslFile.Link)
		exportable := wgslFile.ImportPath != nil

		if exportable {
			declaredImportPaths[*wgslFile.ImportPath] = normalizedLink
		}

		functions := make([]ShaderSearchableInfo, 0)
		for _, fn := range wgslFile.Functions {
			functions = append(functions, ShaderSearchableInfo{
				Link:           normalizedLink,
				Filename:       wgslFile.Filename,
				Exportable:     exportable,
				Name:           fn.Name,
				Type:           "function",
				StageAttribute: fn.StageAttribute,
				Comment:        fn.Comment,
			})
		}

		structures := make([]ShaderSearchableInfo, 0)
		for _, structure := range wgslFile.Structures {
			structures = append(structures, ShaderSearchableInfo{
				Link:       normalizedLink,
				Filename:   wgslFile.Filename,
				Exportable: exportable,
				Name:       structure.Name,
				Type:       "struct",
			})
		}

		consts := make([]ShaderSearchableInfo, 0)
		for _, c := range wgslFile.Consts {
			consts = append(consts, ShaderSearchableInfo{
				Link:       normalizedLink,
				Filename:   wgslFile.Filename,
				Exportable: exportable,
				Name:       c.Name,
				Type:       "const",
			})
		}

		bindings := make([]ShaderSearchableInfo, 0)
		for _, binding := range wgslFile.Bindings {
			bindings = append(bindings, ShaderSearchableInfo{
				Link:       normalizedLink,
				Filename:   wgslFile.Filename,
				Exportable: exportable,
				Name:       binding.Name,
				Type:       "binding",
			})
		}

		searchInfo = append(searchInfo, functions...)
		searchInfo = append(searchInfo, structures...)
		searchInfo = append(searchInfo, consts...)
		searchInfo = append(searchInfo, bindings...)

		parsing_bar.Add(1)
	}

	processing_bar := progressbar.Default(
		int64(len(filePaths)),
		"processing WGSL files",
	)

	for _, wgslFile := range wgslFiles {
		wgslFile.ResolveTypeLinks(declaredImportPaths)
		wgslFile.generateWgslPage()
		processing_bar.Add(1)
	}

	compiledTemplate, err := raymond.Parse(HOME_DOC_TEMPLATE_SOURCE)
	if err != nil {
		log.Fatal(err)
	}

	files := []map[string]string{}
	for _, filePath := range filePaths {
		docPath, err := filepath.Rel(sourcePath, filePath)
		if err != nil {
			log.Fatal(err)
		}

		docPath = strings.Replace(docPath, "src/", "", 1)
		docPath = strings.Replace(docPath, ".wgsl", "", 1)
		docPath = DedupPathParts(docPath) + ".html"

		files = append(files, map[string]string{
			"file": docPath,
		})
	}

	html, err := compiledTemplate.Exec(map[string]interface{}{
		"files": files,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(OUTPUT_DIR_ROOT, "index.html"), []byte(html), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// copy stuff to /public

	err = os.MkdirAll(PUBLIC_FOLDER, os.ModePerm)
	if err != nil {
		log.Fatal("Error creating public folder:", err)
		return
	}

	searchInfoJSON, err := json.MarshalIndent(searchInfo, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling searchInfo:", err)
		return
	}

	err = os.WriteFile(filepath.Join(PUBLIC_FOLDER, "search-info.json"), searchInfoJSON, 0644)
	if err != nil {
		log.Fatal("Error writing search-info.json:", err)
	}

	// Copy files to the public folder
	for _, file := range copyToPublic {
		src := file
		dst := filepath.Join(PUBLIC_FOLDER, filepath.Base(file))
		err := CopyFile(src, dst)
		if err != nil {
			log.Fatal("Error copying file:", err)
		}
	}

	// Copy serve.json to output folder
	err = CopyFile("./serve.json", filepath.Join(OUTPUT_DIR_ROOT, "serve.json"))
	if err != nil {
		log.Fatal("Error copying serve.json:", err)
	}
}

func getGithubLink(dir string, basename string) string {
	innerPath, err := filepath.Rel(sourcePath, dir)
	if err != nil {
		log.Fatal(err)
	}

	joinedPath := filepath.Join(innerPath, basename)

	baseURL, err := url.Parse(bevyUrl)
	if err != nil {
		log.Fatal(err)
	}

	return baseURL.ResolveReference(&url.URL{Path: joinedPath}).String()
}

func parseWGSLFile(wgslFilePath string) WgslFile {
	wgslCodeBytes, err := os.ReadFile(wgslFilePath)
	if err != nil {
		log.Fatal(err)
	}
	normalizedCode := strings.ReplaceAll(string(wgslCodeBytes), "\n\r", "\n")

	basename := filepath.Base(wgslFilePath)
	filename := strings.Replace(basename, ".wgsl", "", 1)
	originalDir := filepath.Dir(wgslFilePath)
	dir := DedupPathParts(strings.Replace(originalDir, "src/", "", 1))

	innerPath, err := filepath.Rel(sourcePath, dir)
	if err != nil {
		log.Fatal(err)
	}
	wgslPath := DedupPathParts(filepath.Join(innerPath, filename)) + ".html"

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
	githubLink := getGithubLink(originalDir, basename)

	wgslFile := WgslFile{
		ImportPath: importPath,

		Consts:           consts,
		ConstsShaderDefs: AnyShaderDefs(consts),
		NotEmptyConsts:   len(consts) != 0,

		Bindings:           bindings,
		BindingsShaderDefs: AnyShaderDefs(bindings),
		NotEmptyBindings:   len(bindings) != 0,

		Functions:         functions,
		NotEmptyFunctions: len(functions) != 0,

		Structures:           structures,
		StructuresShaderDefs: AnyShaderDefs(structures),
		NotEmptyStructures:   len(structures) != 0,
		DeclaredImports:      declaredImports,

		Filename:   basename,
		WgslPath:   wgslPath,
		GithubLink: githubLink,
		Link:       wgslPath,
	}

	return wgslFile
}

func (wgslFile *WgslFile) ResolveTypeLinks(declaredImportPaths map[string]string) {
	importsMap := make(map[string]string)
	structuresList := make([]string, 0)

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

	for _, structure := range wgslFile.Structures {
		structuresList = append(structuresList, structure.Name)
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

func (wgslFile *WgslFile) generateWgslPage() {
	fileOutputPath := filepath.Join(OUTPUT_DIR_ROOT, wgslFile.WgslPath)
	compiledTemplate, err := raymond.Parse(WGSL_DOC_TEMPLATE_SOURCE)
	if err != nil {
		log.Fatal(err)
	}

	html, err := compiledTemplate.Exec(wgslFile)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(filepath.Dir(filepath.Join(OUTPUT_DIR_ROOT, wgslFile.WgslPath)), os.ModePerm)
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

func extractConsts(normalizedCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []WgslConst {
	matches := constPattern.FindAllStringSubmatch(normalizedCode, -1)
	var results []WgslConst
	for _, match := range matches {
		name, typ, value := match[1], match[2], match[3]

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
		typ = RemovePath(typ)

		results = append(results, WgslConst{
			LineNumber:    lineNumber,
			Name:          name,
			Value:         value,
			HasShaderDefs: len(thisShaderDefs) > 0,
			ShaderDefs:    thisShaderDefs,
			TypeInfo: WgslTypeInfo{
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

func getShaderDefsByLine(shaderDefs []ShaderDefBlock, lineNumber int) []WgslDefResult {
	var results []WgslDefResult

	for _, shaderDef := range shaderDefs {
		ifDefEndline := ValueOrDefault(shaderDef.ElseLine, shaderDef.EndifLine)

		if lineNumber > shaderDef.IfdefLine && lineNumber < ifDefEndline {
			results = append(results, WgslDefResult{
				DefName:    shaderDef.DefName,
				Branch:     "if",
				LineNumber: shaderDef.IfdefLine,
			})
		}

		// Check if the line number is between elseLine and endifLine
		if shaderDef.ElseLine != nil && lineNumber > *shaderDef.ElseLine && lineNumber < shaderDef.EndifLine {
			results = append(results, WgslDefResult{
				DefName:    shaderDef.DefName,
				Branch:     "else",
				LineNumber: *shaderDef.ElseLine,
			})
		}
	}

	return results
}

func extractStructures(normalizedCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []WgslStructure {
	matches := structurePattern.FindAllStringSubmatchIndex(normalizedCode, -1)
	var structures []WgslStructure

	for _, match := range matches {
		name := normalizedCode[match[2]:match[3]]
		fieldsRaw := normalizedCode[match[4]:match[5]]

		commentStrip := regexp.MustCompile(`\/{1,3}.*`)
		cleanFields := commentStrip.ReplaceAllString(fieldsRaw, "")
		fields := parseNamedTypeString(cleanFields, normalizedCode, shaderDefs)

		lineNumber := getLineNumber(normalizedCode, match[0])
		comments := getItemComments(lineNumber, lineComments)
		shaderDefsThis := getShaderDefsByLine(shaderDefs, lineNumber)

		fieldsShaderDefs := SomeBy(fields, func(field WgslNamedType) bool {
			return field.HasShaderDefs
		})

		structures = append(structures, WgslStructure{
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

func extractFunctions(normalizedCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []WgslFunction {
	var functions []WgslFunction
	fullCode := normalizedCode

	matches := functionPattern.FindAllStringSubmatchIndex(fullCode, -1)

	for _, matchIdx := range matches {
		startIdx := matchIdx[0]
		endIdx := matchIdx[1]
		signature := strings.TrimSpace(fullCode[startIdx:endIdx])
		signature = strings.TrimSuffix(signature, "{")

		var stageAttr string
		if stageMatch := shaderStagePattern.FindStringSubmatch(signature); len(stageMatch) > 1 {
			stageAttr = stageMatch[1]
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
		returnTypeAnnotations := make([]WgslAnnotation, 0)

		if rt := regexp.MustCompile(`->(.*)`).FindStringSubmatch(signature); len(rt) > 1 {
			trimmedRt := strings.TrimSpace(rt[1])

			matches := annotationPattern.FindAllStringSubmatchIndex(trimmedRt, -1)

			for _, match := range matches {
				returnTypeAnnotations = append(returnTypeAnnotations, WgslAnnotation{
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

		functions = append(functions, WgslFunction{
			StageAttribute: stageAttr,
			Name:           name,
			LineNumber:     lineNumber,
			Params:         params,
			HasParams:      len(params) != 0,
			HasShaderDefs:  len(thisShaderDefs) > 0,
			ShaderDefs:     thisShaderDefs,
			Comment:        strings.Join(comments, "\n"),
			ReturnTypeInfo: WgslTypeInfo{
				Type:        returnType,
				Annotations: returnTypeAnnotations,
			},
		})
	}

	return functions
}

func extractBindings(code string, lineComments map[int]string, shaderDefs []ShaderDefBlock) []WgslBinding {
	var bindings []WgslBinding

	matches := bindingPattern.FindAllStringSubmatchIndex(code, -1)
	for _, matchIdx := range matches {
		match := code[matchIdx[0]:matchIdx[1]]
		submatches := bindingPattern.FindStringSubmatch(match)

		groupIndex := submatches[1]
		bindingIndex := submatches[2]
		bindingType := submatches[3]
		name := submatches[4]
		typeStr := RemovePath(submatches[5])
		lineNumber := getLineNumber(code, matchIdx[0])
		thisShaderDefs := getShaderDefsByLine(shaderDefs, lineNumber)
		annotations := []WgslAnnotation{
			{
				Name:  "group",
				Value: groupIndex,
			},
			{
				Name:  "binding",
				Value: bindingIndex,
			},
		}

		bindings = append(bindings, WgslBinding{
			LineNumber:    lineNumber,
			Name:          name,
			Annotations:   annotations,
			BindingType:   bindingType,
			HasShaderDefs: len(thisShaderDefs) > 0,
			ShaderDefs:    thisShaderDefs,
			TypeInfo: WgslTypeInfo{
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

// ---------------------------------------------

func parseNamedTypeString(str, fullCode string, shaderDefs []ShaderDefBlock) []WgslNamedType {
	str = strings.ReplaceAll(str, "\n", "")
	str = regexp.MustCompile(`#ifdef\s+\w+|#else|#endif`).ReplaceAllString(str, "")
	str = strings.TrimSpace(str)

	entries := SplitParams(str)
	var result []WgslNamedType

	for _, entry := range entries {
		annotations := make([]WgslAnnotation, 0)
		matches := annotationPattern.FindAllStringSubmatchIndex(entry, -1)

		for _, match := range matches {
			annotations = append(annotations, WgslAnnotation{
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
		typ := RemovePath(splittedParam[1])

		// Approximate the line number
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*:`)
		loc := re.FindStringIndex(fullCode)
		lineNumber := 0
		if loc != nil {
			lineNumber = getLineNumber(fullCode, loc[0])
		}

		shaderDefMatches := getShaderDefsByLine(shaderDefs, lineNumber)

		result = append(result, WgslNamedType{
			Annotations:   annotations,
			Name:          name,
			HasShaderDefs: len(shaderDefMatches) > 0,
			ShaderDefs:    shaderDefMatches,
			TypeInfo: WgslTypeInfo{
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
