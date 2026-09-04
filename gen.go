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
	if sourcePath, err := filepath.Abs(config.SourcePath); err == nil {
		config.SourcePath = sourcePath
	}
	if config.SourceGithubRoot != "" {
		if repositoryRoot, err := filepath.Abs(config.SourceGithubRoot); err == nil {
			config.SourceGithubRoot = repositoryRoot
		}
	}
	fmt.Println("🚀 Starting WGSL Documentation Generator")
	fmt.Println("========================================")
	fmt.Printf("📂 Project Directory     : %s\n", config.SourcePath)
	fmt.Printf("🔍 File Filter Pattern   : %s\n", config.FileFilter)
	fmt.Printf("📁 Output Directory      : %s\n", config.OutputDir)
	fmt.Printf("🏷️ Documentation Version: %s\n", config.Version)
	fmt.Println("========================================")

	inputs := getShaderInputs(config)
	cargoMetadata := loadOptionalCargoMetadata(config)
	totalFiles := int64(len(inputs))

	utils.LoadWgslTypes()
	SetupHandlebars()

	searchInfo := make([]ShaderSearchableInfo, 0, 4096)
	declaredImportPaths := make(map[string]string)
	parsingBar := progressbar.Default(totalFiles, "📄 Reading WGSL Files")
	wgslFiles := parseShaderInputs(inputs, config.ProjectVersion, config.OutputDir, parsingBar)
	resolveWgslPathCollisions(wgslFiles)
	for i := range wgslFiles {
		wgslFiles[i].Link = joinDocURL("project", wgslFiles[i].WgslPath)
		appendSearchInfo(&searchInfo, &declaredImportPaths, wgslFiles[i])
	}

	sections, totalProject, totalDependency := buildHomeSections(wgslFiles)
	// Keep a durable registry of every package/version generated into this
	// output directory. A build may contain only one package, but the homepage
	// should still represent packages produced by earlier builds.
	registry := updatePackageRegistry(config.OutputDir, sections)
	registrySections := packageRegistrySections(registry)
	registryShaderCount := packageRegistryShaderCount(registry)
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

	for _, wgslFile := range wgslFiles {
		wgslFile := wgslFile
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// A combined build may encounter the same package first as a
			// dependency and later as a canonical matrix source. Preserve an
			// already-rendered page during dependency passes so those later
			// canonical pages cannot be overwritten by a less precise source ref.
			if wgslFile.Dependency {
				pagePath := filepath.Join(versionedOutput, wgslFile.WgslPath)
				if _, err := os.Stat(pagePath); err == nil {
					processingBar.Add(1)
					return
				}
			}
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
			dependencies := packageDependencies(wgslFiles, group.PackageName, group.Version)
			metadata := findPackageMetadata(cargoMetadata, group.PackageName, group.Version)
			directDependencies, transitiveDependencies := cargoPackageDependencies(cargoMetadata, group.PackageName, group.Version, wgslFiles)
			renderTemplateToFile(PACKAGE_DOC_TEMPLATE_SOURCE, map[string]interface{}{
				"name":                      group.PackageName,
				"files":                     group.AllFiles,
				"count":                     group.Count,
				"description":               group.Description,
				"dependencies":              dependencies,
				"hasDependencies":           len(dependencies) > 0,
				"dependencyCount":           len(dependencies),
				"directDependencies":        directDependencies,
				"transitiveDependencies":    transitiveDependencies,
				"hasDirectDependencies":     len(directDependencies) > 0,
				"hasTransitiveDependencies": len(transitiveDependencies) > 0,
				"directDependencyCount":     len(directDependencies),
				"transitiveDependencyCount": len(transitiveDependencies),
				"authors":                   metadata.Authors, "license": metadata.License,
				"repository": metadata.Repository, "homepage": metadata.Homepage,
				"packageVersion":   group.Version,
				"version":          config.Version,
				"projectVersion":   group.Version,
				"projectCount":     group.Count,
				"urlPrefix":        joinDocURL("project", ""),
				"packageURLPrefix": joinDocURL("project", filepath.Join(group.PackageName, group.Version)),
				"versionOptions":   packageVersionOptions(config.OutputDir, group.PackageName, group.Version),
			}, filepath.Join(versionedOutput, group.DetailPath))
		}
	}

	renderTemplateToFile(HOME_DOC_TEMPLATE_SOURCE, map[string]interface{}{
		"sections":         registrySections,
		"packageCount":     len(registry),
		"totalShaderCount": registryShaderCount,
		"projectCount":     totalProject,
		"dependencyCount":  totalDependency,
		"name":             config.Name,
		"description":      config.Description,
		"skipHomeButton":   true,
		"version":          config.Version,
		"projectVersion":   config.ProjectVersion,
		"urlPrefix":        joinDocURL("project", ""),
	}, filepath.Join(versionedOutput, "index.html"))

	renderTemplateToFile(NOT_FOUND_TEMPLATE_SOURCE, map[string]interface{}{},
		filepath.Join(config.OutputDir, "404.html"))
	writePackageVersionsManifest(config.OutputDir)

	copyItemsToPublic(&config, searchInfo)
}

// parseShaderInputs parses files concurrently while storing results by input
// index. Keeping the original order makes collision resolution and generated
// navigation deterministic regardless of worker scheduling.
func parseShaderInputs(inputs []shaderInput, projectVersion, outputDir string, progress *progressbar.ProgressBar) []wgsl.WgslFile {
	files := make([]wgsl.WgslFile, len(inputs))
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers > len(inputs) {
		workers = len(inputs)
	}
	if workers == 0 {
		return files
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				input := inputs[index]
				file := wgsl.ParseWGSLFile(&input.Config, input.Path)
				if input.Prefix != "" {
					file.WgslPath = filepath.Join(input.Prefix, file.WgslPath)
				}
				file.Dependency = input.Dependency
				file.SourcePath = input.Path
				file.SourceRoot = input.Config.SourcePath
				file.OutputPrefix = input.Prefix
				file.ProjectName = input.PackageName
				file.ProjectDescription = input.PackageDescription
				file.ProjectVersion = input.PackageVersion
				if file.ProjectVersion == "" {
					file.ProjectVersion = projectVersion
				}
				file.ProjectURLPrefix = joinDocURL("project", "")
				file.PackageURLPrefix = joinDocURL("project", filepath.Join(input.PackageName, input.PackageVersion))
				file.VersionOptions = packageVersionOptions(outputDir, input.PackageName, input.PackageVersion)
				file.WgslPath = strings.Replace(file.WgslPath, "src/", "", 1)
				file.WgslPath = utils.DedupPathParts(file.WgslPath)
				files[index] = file
				progress.Add(1)
			}
		}()
	}
	for index := range inputs {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	return files
}

func resolveWgslPathCollisions(files []wgsl.WgslFile) {
	byStem := make(map[string][]int)
	for i, file := range files {
		byStem[strings.TrimSuffix(file.WgslPath, ".html")] = append(byStem[strings.TrimSuffix(file.WgslPath, ".html")], i)
	}
	for _, indexes := range byStem {
		if len(indexes) < 2 {
			continue
		}
		extensions := make(map[string]bool)
		for _, index := range indexes {
			extensions[strings.TrimPrefix(filepath.Ext(files[index].Filename), ".")] = true
		}
		if len(extensions) > 1 {
			for _, index := range indexes {
				extension := strings.TrimPrefix(filepath.Ext(files[index].Filename), ".")
				files[index].WgslPath = strings.TrimSuffix(files[index].WgslPath, ".html") + "." + extension + ".html"
			}
		}
	}

	byPath := make(map[string][]int)
	for i, file := range files {
		byPath[file.WgslPath] = append(byPath[file.WgslPath], i)
	}
	for _, indexes := range byPath {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			dir, err := filepath.Rel(files[index].SourceRoot, filepath.Dir(files[index].SourcePath))
			if err != nil {
				continue
			}
			parts := []string{"root"}
			if dir != "." {
				parts = strings.Split(filepath.ToSlash(dir), "/")
			}
			for width := 1; width <= len(parts); width++ {
				suffix := strings.Join(parts[len(parts)-width:], "/")
				unique := true
				for _, other := range indexes {
					if other == index {
						continue
					}
					otherDir, _ := filepath.Rel(files[other].SourceRoot, filepath.Dir(files[other].SourcePath))
					otherParts := []string{"root"}
					if otherDir != "." {
						otherParts = strings.Split(filepath.ToSlash(otherDir), "/")
					}
					if len(otherParts) >= width && strings.Join(otherParts[len(otherParts)-width:], "/") == suffix {
						unique = false
						break
					}
				}
				if unique {
					files[index].WgslPath = filepath.ToSlash(filepath.Join(files[index].OutputPrefix, suffix, filepath.Base(files[index].WgslPath)))
					break
				}
			}
		}
	}
}

// moduleLabels returns the compact names shown in package shader lists. Most
// files can use their basename, but repeated basenames need a path suffix so
// users can tell them apart (for example, deferred/top and prepass/top).
func moduleLabels(files []wgsl.WgslFile) []string {
	labels := make([]string, len(files))
	groups := make(map[string][]int)
	for i, file := range files {
		labels[i] = strings.TrimSuffix(filepath.Base(file.WgslPath), ".html")
		key := file.ProjectName + "\x00" + file.ProjectVersion + "\x00" + labels[i]
		groups[key] = append(groups[key], i)
	}

	for _, indexes := range groups {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			parts := sourceDirectoryParts(files[index])
			for width := 1; width <= len(parts); width++ {
				suffix := strings.Join(parts[len(parts)-width:], "/")
				unique := true
				for _, other := range indexes {
					if other == index {
						continue
					}
					otherParts := sourceDirectoryParts(files[other])
					if len(otherParts) >= width && strings.Join(otherParts[len(otherParts)-width:], "/") == suffix {
						unique = false
						break
					}
				}
				if unique {
					labels[index] = suffix + "/" + labels[index]
					break
				}
			}
		}
	}
	return labels
}

func sourceDirectoryParts(file wgsl.WgslFile) []string {
	if file.SourcePath == "" || file.SourceRoot == "" {
		dir := filepath.ToSlash(filepath.Dir(file.WgslPath))
		if dir == "." || dir == "" {
			return []string{"root"}
		}
		return strings.Split(dir, "/")
	}
	dir, err := filepath.Rel(file.SourceRoot, filepath.Dir(file.SourcePath))
	if err != nil || dir == "." || dir == "" {
		return []string{"root"}
	}
	return strings.Split(filepath.ToSlash(dir), "/")
}

func appendSearchInfo(searchInfo *[]ShaderSearchableInfo, imports *map[string]string, file wgsl.WgslFile) {
	link := utils.NormalizeLink(file.Link)
	exportable := file.ImportPath != nil
	if exportable {
		(*imports)[*file.ImportPath] = link
	}
	for _, fn := range file.Functions {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: fn.Name, Type: "function", StageAttribute: fn.StageAttribute, Comment: fn.Comment})
	}
	for _, item := range file.Structures {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: item.Name, Type: "struct"})
	}
	for _, item := range file.Consts {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: item.Name, Type: "const"})
	}
	for _, item := range file.Bindings {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: item.Name, Type: "binding"})
	}
}

type packageRegistryEntry struct {
	PackageName string `json:"packageName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Count       int    `json:"count"`
	DetailPath  string `json:"detailPath"`
}

func updatePackageRegistry(outputDir string, sections []homeSection) []packageRegistryEntry {
	registryPath := filepath.Join(outputDir, "public", "packages.json")
	registry := make([]packageRegistryEntry, 0)
	if data, err := os.ReadFile(registryPath); err == nil {
		_ = json.Unmarshal(data, &registry)
	}
	byKey := make(map[string]packageRegistryEntry, len(registry))
	for _, entry := range registry {
		byKey[entry.PackageName+"\x00"+entry.Version] = entry
	}
	for _, section := range sections {
		for _, group := range section.Groups {
			key := group.PackageName + "\x00" + group.Version
			byKey[key] = packageRegistryEntry{
				PackageName: group.PackageName,
				Description: group.Description,
				Version:     group.Version,
				Count:       group.Count,
				DetailPath:  group.DetailPath,
			}
		}
	}
	registry = registry[:0]
	for _, entry := range byKey {
		registry = append(registry, entry)
	}
	slices.SortFunc(registry, func(a, b packageRegistryEntry) int {
		if c := strings.Compare(a.PackageName, b.PackageName); c != 0 {
			return c
		}
		return strings.Compare(b.Version, a.Version)
	})
	data, err := json.MarshalIndent(registry, "", "  ")
	if err == nil {
		_ = os.MkdirAll(filepath.Dir(registryPath), os.ModePerm)
		_ = os.WriteFile(registryPath, data, 0644)
	}
	return registry
}

func packageRegistrySections(registry []packageRegistryEntry) []homeSection {
	byPackage := make(map[string][]packageRegistryEntry)
	for _, entry := range registry {
		byPackage[entry.PackageName] = append(byPackage[entry.PackageName], entry)
	}
	groups := make([]homeGroup, 0, len(byPackage))
	for packageName, entries := range byPackage {
		slices.SortFunc(entries, func(a, b packageRegistryEntry) int { return strings.Compare(b.Version, a.Version) })
		latest := entries[0]
		options := make([]map[string]string, 0, len(entries))
		for _, entry := range entries {
			options = append(options, map[string]string{"label": entry.Version, "url": entry.DetailPath})
		}
		groups = append(groups, homeGroup{Name: packageName, PackageName: packageName, Description: latest.Description, Version: latest.Version, Count: latest.Count, DetailPath: latest.DetailPath, VersionOptions: options})
	}
	slices.SortFunc(groups, func(a, b homeGroup) int { return strings.Compare(a.PackageName, b.PackageName) })
	return []homeSection{{Title: "Packages", Groups: groups}}
}

func packageRegistryShaderCount(registry []packageRegistryEntry) int {
	total := 0
	for _, entry := range registry {
		total += entry.Count
	}
	return total
}

func packageDependencies(files []wgsl.WgslFile, packageName, version string) []map[string]string {
	seen := make(map[string]bool)
	dependencies := make([]map[string]string, 0)
	for _, file := range files {
		if !file.Dependency || (file.ProjectName == packageName && file.ProjectVersion == version) {
			continue
		}
		key := file.ProjectName + "\x00" + file.ProjectVersion
		if seen[key] {
			continue
		}
		seen[key] = true
		dependencies = append(dependencies, map[string]string{"name": file.ProjectName, "version": file.ProjectVersion, "url": filepath.ToSlash(filepath.Join(file.ProjectName, file.ProjectVersion, "index.html"))})
	}
	slices.SortFunc(dependencies, func(a, b map[string]string) int { return strings.Compare(a["name"], b["name"]) })
	return dependencies
}

func loadOptionalCargoMetadata(config config.Config) discovery.CargoMetadata {
	if config.NoDeps {
		return discovery.CargoMetadata{}
	}
	metadata, err := discovery.ReadCargoMetadata(context.Background(), config.SourcePath, config.Offline)
	if err != nil {
		return discovery.CargoMetadata{}
	}
	return metadata
}

func findPackageMetadata(metadata discovery.CargoMetadata, name, version string) discovery.CargoPackage {
	for _, pkg := range metadata.Packages {
		if pkg.Name == name && pkg.Version == version {
			return pkg
		}
	}
	return discovery.CargoPackage{}
}

func cargoPackageDependencies(metadata discovery.CargoMetadata, name, version string, files []wgsl.WgslFile) ([]map[string]string, []map[string]string) {
	if metadata.Resolve == nil {
		return nil, nil
	}
	packages := make(map[string]discovery.CargoPackage)
	ids := make(map[string]string)
	for _, pkg := range metadata.Packages {
		packages[pkg.ID] = pkg
		ids[pkg.Name+"\x00"+pkg.Version] = pkg.ID
	}
	rootID := ids[name+"\x00"+version]
	if rootID == "" {
		return nil, nil
	}
	nodes := make(map[string]discovery.CargoNode)
	for _, node := range metadata.Resolve.Nodes {
		nodes[node.ID] = node
	}
	available := make(map[string]bool)
	for _, file := range files {
		available[file.ProjectName+"\x00"+file.ProjectVersion] = true
	}
	seen := map[string]bool{rootID: true}
	queue := []string{rootID}
	depth := map[string]int{rootID: 0}
	var direct, transitive []map[string]string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, dep := range nodes[id].Deps {
			if seen[dep.PackageID] {
				continue
			}
			seen[dep.PackageID] = true
			queue = append(queue, dep.PackageID)
			depth[dep.PackageID] = depth[id] + 1
			pkg, ok := packages[dep.PackageID]
			if !ok || !available[pkg.Name+"\x00"+pkg.Version] {
				continue
			}
			item := map[string]string{"name": pkg.Name, "version": pkg.Version, "url": filepath.ToSlash(filepath.Join(pkg.Name, pkg.Version, "index.html"))}
			if depth[dep.PackageID] == 1 {
				direct = append(direct, item)
			} else {
				transitive = append(transitive, item)
			}
		}
	}
	slices.SortFunc(direct, func(a, b map[string]string) int { return strings.Compare(a["name"], b["name"]) })
	slices.SortFunc(transitive, func(a, b map[string]string) int { return strings.Compare(a["name"], b["name"]) })
	return direct, transitive
}

func writePackageVersionsManifest(outputDir string) {
	packages := make(map[string][]map[string]string)
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return
	}
	for _, pkg := range entries {
		if !pkg.IsDir() || pkg.Name() == "public" {
			continue
		}
		versions, err := os.ReadDir(filepath.Join(outputDir, pkg.Name()))
		if err != nil {
			continue
		}
		for _, version := range versions {
			if version.IsDir() {
				packages[pkg.Name()] = append(packages[pkg.Name()], map[string]string{
					"label": version.Name(),
					"url":   filepath.ToSlash(filepath.Join(pkg.Name(), version.Name(), "index.html")),
				})
			}
		}
	}
	data, err := json.Marshal(packages)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Join(outputDir, "public"), os.ModePerm)
	_ = os.WriteFile(filepath.Join(outputDir, "public", "package-versions.json"), data, 0644)
}

func joinDocURL(version, filePath string) string {
	prefix := ""
	if version != "project" {
		prefix = version
	}
	return strings.Trim(strings.Join([]string{prefix, filepath.ToSlash(filePath)}, "/"), "/")
}

// packageVersionOptions finds sibling version builds in a combined output
// directory and returns links to the same package in each version.
func packageVersionOptions(outputDir, packageName, packageVersion string) []map[string]string {
	options := []map[string]string{{"label": packageVersion, "url": filepath.ToSlash(filepath.Join(packageName, packageVersion, "index.html"))}}
	versions, err := os.ReadDir(filepath.Join(outputDir, packageName))
	if err != nil {
		return options
	}
	for _, version := range versions {
		if !version.IsDir() || version.Name() == packageVersion {
			continue
		}
		options = append(options, map[string]string{
			"label": version.Name(),
			"url":   filepath.ToSlash(filepath.Join(packageName, version.Name(), "index.html")),
		})
	}
	return options
}

type homeSection struct {
	Title  string
	Groups []homeGroup
}

type homeGroup struct {
	Name           string
	PackageName    string
	Description    string
	Version        string
	Count          int
	Files          []map[string]string
	AllFiles       []map[string]string
	DetailPath     string
	Preview        bool
	Remaining      int
	VersionOptions []map[string]string
}

func buildHomeSections(files []wgsl.WgslFile) ([]homeSection, int, int) {
	projectGroups := map[string][]map[string]string{}
	dependencyGroups := map[string][]map[string]string{}
	groupDescriptions := map[string]string{}
	groupVersions := map[string]string{}
	groupPackageNames := map[string]string{}
	labels := moduleLabels(files)
	for i, file := range files {
		entry := map[string]string{
			"file":  file.WgslPath,
			"label": labels[i],
		}
		parts := strings.Split(file.WgslPath, "/")
		if len(parts) >= 3 {
			entry["relative"] = strings.Join(parts[2:], "/")
		} else {
			entry["relative"] = file.WgslPath
		}
		name := "Project shaders"
		if len(parts) >= 2 {
			name = parts[0] + " " + parts[1]
		}
		if _, exists := groupDescriptions[name]; !exists {
			groupDescriptions[name] = file.ProjectDescription
			groupVersions[name] = file.ProjectVersion
			groupPackageNames[name] = file.ProjectName
		}
		if file.Dependency {
			dependencyGroups[name] = append(dependencyGroups[name], entry)
			continue
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
				Name:        name,
				PackageName: groupPackageNames[name],
				Description: groupDescriptions[name],
				Version:     groupVersions[name],
				Count:       len(entries),
				Files:       preview,
				AllFiles:    entries,
				DetailPath:  packageDetailPath(entries),
				Preview:     len(entries) > len(preview),
				Remaining:   len(entries) - len(preview),
			})
		}
		slices.SortFunc(groups, func(a, b homeGroup) int { return strings.Compare(a.Name, b.Name) })
		return groups
	}
	project := toGroups(projectGroups)
	dependencies := toGroups(dependencyGroups)
	all := append(project, dependencies...)
	slices.SortFunc(all, func(a, b homeGroup) int { return strings.Compare(a.Name, b.Name) })
	sections := []homeSection{{Title: "Packages", Groups: all}}
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

func packageDetailPath(entries []map[string]string) string {
	if len(entries) == 0 {
		return "packages/unknown/index.html"
	}
	parts := strings.Split(entries[0]["file"], "/")
	if len(parts) >= 2 {
		return filepath.Join(parts[0], parts[1], "index.html")
	}
	return "packages/unknown/index.html"
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
		if matchesShaderFile(config.FileFilter, entry.Name()) && !shouldExclude(config.SourcePath, filePath, config.Exclude) {
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

func matchesShaderFile(filter, name string) bool {
	if filter == "*.wgsl" && strings.EqualFold(filepath.Ext(name), ".wesl") {
		return true
	}
	matched, _ := path.Match(filter, name)
	return matched
}

type shaderInput struct {
	Path               string
	Config             config.Config
	Prefix             string
	Dependency         bool
	PackageName        string
	PackageDescription string
	PackageVersion     string
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
		inputs = append(inputs, shaderInput{Path: filePath, Config: config, Prefix: prefix, PackageName: config.Name, PackageDescription: config.Description, PackageVersion: config.ProjectVersion})
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
	filteredInputs := inputs[:0]
	for i := range inputs {
		if pkg, ok := discovery.PackageForPath(metadata.Packages, inputs[i].Path); ok {
			if discovery.IsWorkspaceRootPackage(metadata, pkg.ManifestPath) {
				continue
			}
			inputs[i].Config.SourcePath = filepath.Dir(pkg.ManifestPath)
			if inputs[i].Config.SourceGithubRoot == "" {
				inputs[i].Config.SourceGithubRoot = config.SourcePath
			}
			inputs[i].PackageName = pkg.Name
			inputs[i].PackageDescription = pkg.Description
			if pkg.ManifestPath == manifestPath && config.Description != "" {
				inputs[i].PackageDescription = config.Description
			}
			inputs[i].PackageVersion = pkg.Version
			inputs[i].Prefix = filepath.Join(pkg.Name, pkg.Version)
		}
		filteredInputs = append(filteredInputs, inputs[i])
	}
	inputs = filteredInputs
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
		dependencyConfig.SourceGithubRoot = discovery.RepositoryRootForPackage(manifest)
		// Cargo metadata records the upstream repository for each resolved
		// package. Preserve it so dependency shader pages can link back to the
		// exact source file just like project shaders do.
		dependencyConfig.SourceGithubURL = dependencyRepository(metadata, dependency.Package, dependency.Version)
		dependencyConfig.SourceGithubSubpath = discovery.RepositorySubpathForPackage(manifest, dependencyConfig.SourceGithubURL, dependency.Package)
		if dependencyConfig.SourceGithubURL != config.SourceGithubURL {
			dependencyConfig.SourceGithubRef = inferDependencyGithubRef(dependency.Version)
		}
		if seenPaths[dependency.Path] {
			continue
		}
		seenPaths[dependency.Path] = true
		inputs = append(inputs, shaderInput{Path: dependency.Path, Config: dependencyConfig, Prefix: filepath.Join(dependency.Package, dependency.Version), Dependency: true, PackageName: dependency.Package, PackageDescription: dependencyDescription(metadata, dependency.Package, dependency.Version), PackageVersion: dependency.Version})
	}
	return inputs
}

// inferDependencyGithubRef covers the two common tag conventions used by
// shader-bearing Rust repositories: major-version tags (v29 for wgpu) and
// full semantic-version tags (v0.12.1 for most 0.x crates). Canonical matrix
// sources still provide their exact refs explicitly.
func inferDependencyGithubRef(version string) string {
	major := version
	if dot := strings.IndexByte(version, '.'); dot >= 0 {
		major = version[:dot]
	}
	if major != "0" && major != "" {
		return "v" + major
	}
	if version != "" {
		return "v" + version
	}
	return "main"
}

func dependencyDescription(metadata discovery.CargoMetadata, name, version string) string {
	for _, pkg := range metadata.Packages {
		if pkg.Name == name && pkg.Version == version {
			return pkg.Description
		}
	}
	return ""
}

func dependencyRepository(metadata discovery.CargoMetadata, name, version string) string {
	for _, pkg := range metadata.Packages {
		if pkg.Name == name && pkg.Version == version {
			return pkg.Repository
		}
	}
	return ""
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
	PackageName    string `json:"packageName"`
	PackageVersion string `json:"packageVersion"`
	Filename       string `json:"filename"`
	Exportable     bool   `json:"exportable"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	StageAttribute string `json:"stageAttribute"`
	Comment        string `json:"comment"`
}
