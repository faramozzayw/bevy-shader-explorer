package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	config "main/config"
	utils "main/utils"
	wgsl "main/wgsl"

	"github.com/aymerick/raymond"
	progressbar "github.com/schollz/progressbar/v3"
)

var copyToPublic = []string{
	"assets/styles.css",
	"assets/favicon.ico",
	"assets/search.js",
	"assets/select.js",
	"assets/404.js",
	"assets/404.css",
	"assets/wgsl.png",
	"assets/github-mark.png",
	"assets/github-mark-white.png",
	"templates/search-result.hbs",
	"assets/info-dark.png",
	"assets/info-light.png",
}

func generate(config config.Config) {
	fmt.Println("🚀 Starting WGSL Documentation Generator")
	fmt.Println("========================================")
	fmt.Printf("📂 Project Directory     : %s\n", config.SourcePath)
	fmt.Printf("🔍 File Filter Pattern   : %s\n", config.FileFilter)
	fmt.Printf("📁 Output Directory      : %s\n", config.OutputDir)
	fmt.Printf("🏷️ Documentation Version: %s\n", config.Version)
	fmt.Println("========================================")

	filePaths := getWgslFilesList(config)
	totalFiles := int64(len(filePaths))

	utils.LoadWgslTypes()
	SetupHandlebars()

	searchInfo := make([]ShaderSearchableInfo, 0, 4096)
	declaredImportPaths := make(map[string]string)
	wgslFiles := make([]wgsl.WgslFile, 0, len(filePaths))

	parsingBar := progressbar.Default(totalFiles, "📄 Reading WGSL Files")

	for _, filePath := range filePaths {
		wgslFile := wgsl.ParseWGSLFile(&config, filePath)
		wgslFiles = append(wgslFiles, wgslFile)

		normalizedLink := utils.NormalizeLink(wgslFile.Link)

		exportable := wgslFile.ImportPath != nil

		if exportable {
			declaredImportPaths[*wgslFile.ImportPath] = normalizedLink
		}

		localSearchInfo := make([]ShaderSearchableInfo, 0,
			len(wgslFile.Functions)+len(wgslFile.Structures)+len(wgslFile.Consts)+len(wgslFile.Bindings),
		)

		for _, fn := range wgslFile.Functions {
			localSearchInfo = append(localSearchInfo, ShaderSearchableInfo{
				Link:           normalizedLink,
				Filename:       wgslFile.Filename,
				Exportable:     exportable,
				Name:           fn.Name,
				Type:           "function",
				StageAttribute: fn.StageAttribute,
				Comment:        fn.Comment,
			})
		}

		for _, structure := range wgslFile.Structures {
			localSearchInfo = append(localSearchInfo, ShaderSearchableInfo{
				Link:       normalizedLink,
				Filename:   wgslFile.Filename,
				Exportable: exportable,
				Name:       structure.Name,
				Type:       "struct",
			})
		}

		for _, consts := range wgslFile.Consts {
			localSearchInfo = append(localSearchInfo, ShaderSearchableInfo{
				Link:       normalizedLink,
				Filename:   wgslFile.Filename,
				Exportable: exportable,
				Name:       consts.Name,
				Type:       "const",
			})
		}

		for _, binding := range wgslFile.Bindings {
			localSearchInfo = append(localSearchInfo, ShaderSearchableInfo{
				Link:       normalizedLink,
				Filename:   wgslFile.Filename,
				Exportable: exportable,
				Name:       binding.Name,
				Type:       "binding",
			})
		}

		searchInfo = append(searchInfo, localSearchInfo...)

		parsingBar.Add(1)
	}

	compiledTemplate, err := raymond.Parse(WGSL_DOC_TEMPLATE_SOURCE)
	if err != nil {
		log.Fatal(err)
	}

	processingBar := progressbar.Default(totalFiles, "🛠️ Generating Documentation")

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())

	versionedOutput := filepath.Join(config.OutputDir, config.Version)

	for _, wgslFile := range wgslFiles {
		wgslFile := wgslFile
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			wgslFile.ResolveTypeLinks(declaredImportPaths)
			wgslFile.GenerateWgslPage(compiledTemplate, versionedOutput)
			processingBar.Add(1)
		}()
	}

	wg.Wait()

	files := []map[string]string{}
	for _, filePath := range filePaths {
		docPath, err := filepath.Rel(config.SourcePath, filePath)
		if err != nil {
			log.Fatal(err)
		}

		docPath = strings.Replace(docPath, "src/", "", 1)
		docPath = strings.TrimSuffix(docPath, ".wgsl")
		docPath = utils.DedupPathParts(docPath) + ".html"

		files = append(files, map[string]string{
			"file": docPath,
		})
	}

	renderTemplateToFile(HOME_DOC_TEMPLATE_SOURCE, map[string]interface{}{
		"files":          files,
		"skipHomeButton": true,
		"version":        config.Version,
	}, filepath.Join(versionedOutput, "index.html"))

	renderTemplateToFile(NOT_FOUND_TEMPLATE_SOURCE, map[string]interface{}{},
		filepath.Join(config.OutputDir, "404.html"))

	copyItemsToPublic(&config, searchInfo)
}

func renderTemplateToFile(templateSrc string, context map[string]interface{}, outputPath string) {
	tmpl, err := raymond.Parse(templateSrc)
	if err != nil {
		log.Fatal(err)
	}
	html, err := tmpl.Exec(context)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(outputPath, []byte(html), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func getWgslFilesList(config config.Config) []string {
	var filePaths []string
	err := filepath.WalkDir(config.SourcePath, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filePath != config.SourcePath && shouldExclude(config.SourcePath, filePath, config.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		if matched, _ := path.Match(config.FileFilter, entry.Name()); matched && !shouldExclude(config.SourcePath, filePath, config.Exclude) {
			filePaths = append(filePaths, filePath)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	slices.Sort(filePaths)
	return filePaths
}

func shouldExclude(root, filePath string, excludes []string) bool {
	relative, err := filepath.Rel(root, filePath)
	if err != nil {
		return false
	}
	for _, excluded := range append([]string{".git", "target", "node_modules", "dist", "build"}, excludes...) {
		clean := filepath.Clean(excluded)
		if relative == clean || strings.HasPrefix(relative, clean+string(filepath.Separator)) {
			return true
		}
		if matched, _ := path.Match(clean, filepath.Base(relative)); matched {
			return true
		}
	}
	return false
}

func copyItemsToPublic(config *config.Config, searchInfo []ShaderSearchableInfo) {
	publicDir := filepath.Join(config.OutputDir, "public")
	err := os.MkdirAll(publicDir, os.ModePerm)
	if err != nil {
		log.Fatal("Error creating public directory:", err)
	}

	searchInfoJSON, err := json.MarshalIndent(searchInfo, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling searchInfo:", err)
	}

	err = os.WriteFile(filepath.Join(publicDir, fmt.Sprintf("search-info-%s.json", config.Version)), searchInfoJSON, 0644)
	if err != nil {
		log.Fatal("Error writing search-info.json:", err)
	}

	for _, file := range copyToPublic {
		src := file
		dst := filepath.Join(publicDir, filepath.Base(file))
		err := utils.CopyFile(src, dst)
		if err != nil {
			log.Fatal("Error copying file:", err)
		}
	}
}

type ShaderSearchableInfo struct {
	Link           string `json:"link"`
	Filename       string `json:"filename"`
	Exportable     bool   `json:"exportable"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	StageAttribute string `json:"stageAttribute"`
	Comment        string `json:"comment"`
}
