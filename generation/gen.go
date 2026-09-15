package generation

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	config "main/config"
	utils "main/utils"
	wgsl "main/wgsl"

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
	"generation/templates/search-result.hbs",
	"assets/info-dark.png",
	"assets/info-light.png",
}

func Generate(config config.Config) error {
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

	inputs, err := getShaderInputs(config)
	if err != nil {
		return err
	}
	cargoMetadata := loadOptionalCargoMetadata(config)
	totalFiles := int64(len(inputs))

	utils.LoadWgslTypes()
	SetupHandlebars()

	searchInfo := make([]ShaderSearchableInfo, 0, 4096)
	declaredImportPaths := make(map[string]string)
	parsingBar := progressbar.Default(totalFiles, "📄 Reading WGSL Files")
	wgslFiles, err := parseShaderInputs(inputs, config.ProjectVersion, config.OutputDir, parsingBar)
	if err != nil {
		return err
	}
	resolveWgslPathCollisions(wgslFiles)
	for i := range wgslFiles {
		wgslFiles[i].Link = joinDocURL("project", wgslFiles[i].WgslPath)
		appendSearchInfo(&searchInfo, &declaredImportPaths, wgslFiles[i])
	}

	sections, totalProject, totalDependency := buildHomeSections(wgslFiles)

	// Keep a durable registry of every package/version generated into this
	// output directory. A build may contain only one package, but the homepage
	// should still represent packages produced by earlier builds.
	registry, err := updatePackageRegistry(config.OutputDir, sections)
	if err != nil {
		return err
	}
	registrySections := packageRegistrySections(registry)
	registryShaderCount := packageRegistryShaderCount(registry)
	site := buildDocumentationSite(config, sections, registrySections, registryShaderCount, totalProject, totalDependency, wgslFiles, searchInfo, cargoMetadata, declaredImportPaths)
	return renderDocumentation(config, site, registry)
}

// renderDocumentation selects the output renderer after all discovery,
// parsing, linking, and page-model construction has completed.
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
