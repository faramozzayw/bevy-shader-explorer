package main

import (
	"context"
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
	"main/discovery"
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

	inputs := getShaderInputs(config)
	totalFiles := int64(len(inputs))

	utils.LoadWgslTypes()
	SetupHandlebars()

	searchInfo := make([]ShaderSearchableInfo, 0, 4096)
	declaredImportPaths := make(map[string]string)
	wgslFiles := make([]wgsl.WgslFile, 0, len(inputs))

	parsingBar := progressbar.Default(totalFiles, "📄 Reading WGSL Files")

	for _, input := range inputs {
		wgslFile := wgsl.ParseWGSLFile(&input.Config, input.Path)
		if input.Prefix != "" {
			wgslFile.WgslPath = filepath.Join(input.Prefix, wgslFile.WgslPath)
		}
		wgslFile.Dependency = input.Dependency
		wgslFile.ProjectName = config.Name
		wgslFile.ProjectDescription = config.Description
		wgslFile.ProjectVersion = config.ProjectVersion
		wgslFile.ProjectURLPrefix = joinDocURL(config.Version, "")
		wgslFile.WgslPath = strings.Replace(wgslFile.WgslPath, "src/", "", 1)
		wgslFile.WgslPath = utils.DedupPathParts(wgslFile.WgslPath)
		wgslFile.Link = joinDocURL(config.Version, wgslFile.WgslPath)
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

	sections, totalProject, totalDependency := buildHomeSections(wgslFiles)
	for i := range wgslFiles {
		wgslFiles[i].ProjectShaderCount = totalProject
		wgslFiles[i].DependencyShaderCount = totalDependency
		wgslFiles[i].ProjectCount = totalProject
		wgslFiles[i].DependencyCount = totalDependency
	}

	compiledTemplate, err := raymond.Parse(WGSL_DOC_TEMPLATE_SOURCE)
	if err != nil {
		log.Fatal(err)
	}

	processingBar := progressbar.Default(totalFiles, "🛠️ Generating Documentation")

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())

	versionedOutput := config.OutputDir
	if config.Version != "project" {
		versionedOutput = filepath.Join(config.OutputDir, config.Version)
	}

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

	for _, section := range sections {
		for _, group := range section.Groups {
			if err := os.MkdirAll(filepath.Join(versionedOutput, filepath.Dir(group.DetailPath)), os.ModePerm); err != nil {
				log.Fatal(err)
			}
			renderTemplateToFile(PACKAGE_DOC_TEMPLATE_SOURCE, map[string]interface{}{
				"name":           group.Name,
				"files":          group.AllFiles,
				"count":          group.Count,
				"description":    config.Description,
				"version":        config.Version,
				"projectVersion": config.ProjectVersion,
				"urlPrefix":      joinDocURL(config.Version, ""),
			}, filepath.Join(versionedOutput, group.DetailPath))
		}
	}

	renderTemplateToFile(HOME_DOC_TEMPLATE_SOURCE, map[string]interface{}{
		"sections":        sections,
		"projectCount":    totalProject,
		"dependencyCount": totalDependency,
		"name":            config.Name,
		"description":     config.Description,
		"skipHomeButton":  true,
		"version":         config.Version,
		"projectVersion":  config.ProjectVersion,
		"urlPrefix":       joinDocURL(config.Version, ""),
	}, filepath.Join(versionedOutput, "index.html"))

	renderTemplateToFile(NOT_FOUND_TEMPLATE_SOURCE, map[string]interface{}{},
		filepath.Join(config.OutputDir, "404.html"))

	copyItemsToPublic(&config, searchInfo)
}

func joinDocURL(version, filePath string) string {
	prefix := ""
	if version != "project" {
		prefix = version
	}
	return strings.Trim(strings.Join([]string{prefix, filepath.ToSlash(filePath)}, "/"), "/")
}

type homeSection struct {
	Title  string
	Groups []homeGroup
}

type homeGroup struct {
	Name       string
	Count      int
	Files      []map[string]string
	AllFiles   []map[string]string
	DetailPath string
	Preview    bool
	Remaining  int
}

func buildHomeSections(files []wgsl.WgslFile) ([]homeSection, int, int) {
	projectGroups := map[string][]map[string]string{}
	dependencyGroups := map[string][]map[string]string{}
	for _, file := range files {
		entry := map[string]string{
			"file":  file.WgslPath,
			"label": strings.TrimSuffix(filepath.Base(file.WgslPath), ".html"),
		}
		if file.Dependency {
			parts := strings.Split(file.WgslPath, "/")
			name := "Other dependencies"
			if len(parts) >= 2 {
				name = parts[0] + " " + parts[1]
			}
			dependencyGroups[name] = append(dependencyGroups[name], entry)
			continue
		}
		name := "Project shaders"
		parts := strings.Split(file.WgslPath, "/")
		if len(parts) >= 4 && parts[2] == "crates" {
			name = parts[3]
		}
		projectGroups[name] = append(projectGroups[name], entry)
	}
	toGroups := func(grouped map[string][]map[string]string) []homeGroup {
		groups := make([]homeGroup, 0, len(grouped))
		for name, entries := range grouped {
			slices.SortFunc(entries, func(a, b map[string]string) int { return strings.Compare(a["file"], b["file"]) })
			preview := entries
			if len(preview) > 8 {
				preview = entries[:8]
			}
			groups = append(groups, homeGroup{
				Name:       name,
				Count:      len(entries),
				Files:      preview,
				AllFiles:   entries,
				DetailPath: "packages/" + packageSlug(name) + ".html",
				Preview:    len(entries) > len(preview),
				Remaining:  len(entries) - len(preview),
			})
		}
		slices.SortFunc(groups, func(a, b homeGroup) int { return strings.Compare(a.Name, b.Name) })
		return groups
	}
	project := toGroups(projectGroups)
	dependencies := toGroups(dependencyGroups)
	sections := make([]homeSection, 0, 2)
	if len(project) > 0 {
		sections = append(sections, homeSection{Title: "Project shaders", Groups: project})
	}
	if len(dependencies) > 0 {
		sections = append(sections, homeSection{Title: "Dependencies", Groups: dependencies})
	}
	return sections, countGroups(project), countGroups(dependencies)
}

func packageSlug(name string) string {
	name = strings.ToLower(name)
	var builder strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}

func countGroups(groups []homeGroup) int {
	total := 0
	for _, group := range groups {
		total += group.Count
	}
	return total
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

type shaderInput struct {
	Path       string
	Config     config.Config
	Prefix     string
	Dependency bool
}

func getShaderInputs(config config.Config) []shaderInput {
	projectFiles := getWgslFilesList(config)
	inputs := make([]shaderInput, 0, len(projectFiles))
	seenPaths := make(map[string]bool, len(projectFiles))
	for _, filePath := range projectFiles {
		prefix := ""
		if config.PackageName != "" && config.ProjectVersion != "" {
			prefix = filepath.Join(config.PackageName, config.ProjectVersion)
		}
		inputs = append(inputs, shaderInput{Path: filePath, Config: config, Prefix: prefix})
		seenPaths[filePath] = true
	}
	if config.NoDeps {
		return inputs
	}
	manifestPath := filepath.Join(config.SourcePath, "Cargo.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		return inputs
	}
	metadata, err := discovery.ReadCargoMetadata(context.Background(), config.SourcePath, config.Offline)
	if err != nil {
		modeHint := ""
		if config.Offline {
			modeHint = "; retry without --offline when network access is available"
		} else {
			modeHint = "; retry with --offline if the Cargo cache and lockfile are available"
		}
		log.Printf("warning: dependency discovery skipped: %v%s; continuing with project shaders", err, modeHint)
		return inputs
	}
	packages := discovery.FilterCargoPackages(metadata, config.DependencyInclude, config.DependencyTransitive)
	dependencies, err := discovery.DiscoverDependencyShaders(packages, config.Exclude)
	if err != nil {
		log.Printf("warning: dependency shader discovery skipped: %v", err)
		return inputs
	}
	for _, dependency := range dependencies {
		manifest := findCargoManifest(metadata, dependency.Package, dependency.Version)
		if manifest == "" {
			continue
		}
		dependencyConfig := config
		dependencyConfig.SourcePath = filepath.Dir(manifest)
		dependencyConfig.SourceGithubURL = ""
		if seenPaths[dependency.Path] {
			continue
		}
		seenPaths[dependency.Path] = true
		inputs = append(inputs, shaderInput{Path: dependency.Path, Config: dependencyConfig, Prefix: filepath.Join(dependency.Package, dependency.Version), Dependency: true})
	}
	return inputs
}

func findCargoManifest(metadata discovery.CargoMetadata, name, version string) string {
	for _, pkg := range metadata.Packages {
		if pkg.Name == name && pkg.Version == version {
			return pkg.ManifestPath
		}
	}
	return ""
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
