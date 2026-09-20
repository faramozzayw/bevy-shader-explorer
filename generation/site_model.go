package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"main/config"
	"main/discovery"
	"main/utils"
	"main/wgsl"
)

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
	Dependency     bool
	VersionOptions []map[string]string
}

// documentationSite is the renderer-neutral representation of one generated
// site. HTML is currently the only renderer, but all page data is assembled
// here first so a future JSON renderer can serialize the same structure.
type documentationSite struct {
	Version          string
	HomeSections     []homeSection
	Packages         []documentationPackagePage
	Shaders          []wgsl.WgslFile
	Search           []ShaderSearchableInfo
	TotalShaderCount int
	ProjectCount     int
	DependencyCount  int
}

type documentationPackagePage struct {
	PackageName            string
	Version                string
	BevyVersion            string
	Description            string
	Count                  int
	AllFiles               []map[string]string
	DetailPath             string
	Dependencies           []map[string]string
	DirectDependencies     []map[string]string
	TransitiveDependencies []map[string]string
	Metadata               discovery.CargoPackage
	VersionOptions         []map[string]string
	Dependency             bool
}

type documentationJSON struct {
	Version     string                     `json:"version"`
	Home        documentationHomeJSON      `json:"home"`
	Packages    []documentationPackageJSON `json:"packages"`
	ShaderPages []documentationShaderJSON  `json:"shaderPages"`
	Search      []ShaderSearchableInfo     `json:"search"`
}

type documentationHomeJSON struct {
	Sections        []documentationSectionJSON `json:"sections"`
	PackageCount    int                        `json:"packageCount"`
	ShaderCount     int                        `json:"shaderCount"`
	ProjectCount    int                        `json:"projectCount"`
	DependencyCount int                        `json:"dependencyCount"`
}

type documentationSectionJSON struct {
	Title  string                   `json:"title"`
	Groups []documentationGroupJSON `json:"groups"`
}

type documentationGroupJSON struct {
	Name           string              `json:"name"`
	PackageName    string              `json:"packageName"`
	Version        string              `json:"version"`
	Description    string              `json:"description"`
	Count          int                 `json:"count"`
	Files          []map[string]string `json:"files"`
	DetailPath     string              `json:"detailPath"`
	Preview        bool                `json:"preview"`
	Remaining      int                 `json:"remaining"`
	Dependency     bool                `json:"dependency,omitempty"`
	VersionOptions []map[string]string `json:"versionOptions,omitempty"`
}

type documentationPackageJSON struct {
	Name                   string                    `json:"name"`
	Version                string                    `json:"version"`
	Description            string                    `json:"description"`
	Count                  int                       `json:"count"`
	Files                  []map[string]string       `json:"files"`
	DetailPath             string                    `json:"detailPath"`
	Dependencies           []map[string]string       `json:"dependencies,omitempty"`
	DirectDependencies     []map[string]string       `json:"directDependencies,omitempty"`
	TransitiveDependencies []map[string]string       `json:"transitiveDependencies,omitempty"`
	Metadata               documentationMetadataJSON `json:"metadata"`
	VersionOptions         []map[string]string       `json:"versionOptions"`
}

type documentationMetadataJSON struct {
	Authors    []string `json:"authors,omitempty"`
	License    string   `json:"license,omitempty"`
	Repository string   `json:"repository,omitempty"`
	Homepage   string   `json:"homepage,omitempty"`
}

type documentationShaderJSON struct {
	Path            string                       `json:"path"`
	Filename        string                       `json:"filename"`
	Link            string                       `json:"link"`
	GithubLink      string                       `json:"githubLink,omitempty"`
	ImportPath      *string                      `json:"importPath,omitempty"`
	Dependency      bool                         `json:"dependency"`
	DeclaredImports wgsl.DeclaredImports         `json:"declaredImports,omitempty"`
	Items           documentationShaderItemsJSON `json:"items"`
}

type documentationShaderItemsJSON struct {
	Constants  []wgsl.Const     `json:"constants"`
	Structures []wgsl.Structure `json:"structures"`
	Functions  []wgsl.Function  `json:"functions"`
	Bindings   []wgsl.Binding   `json:"bindings"`
}

func writeDocumentationJSON(outputDir string, site documentationSite) error {
	document := documentationJSON{
		Version: site.Version,
		Home: documentationHomeJSON{
			Sections:        documentationSectionsJSON(site.HomeSections),
			PackageCount:    len(site.Packages),
			ShaderCount:     site.TotalShaderCount,
			ProjectCount:    site.ProjectCount,
			DependencyCount: site.DependencyCount,
		},
		Search: site.Search,
	}
	for _, page := range site.Packages {
		document.Packages = append(document.Packages, documentationPackageJSON{
			Name: page.PackageName, Version: page.Version, Description: page.Description,
			Count: page.Count, Files: page.AllFiles, DetailPath: page.DetailPath,
			Dependencies: page.Dependencies, DirectDependencies: page.DirectDependencies,
			TransitiveDependencies: page.TransitiveDependencies,
			Metadata:               documentationMetadataJSON{Authors: page.Metadata.Authors, License: page.Metadata.License, Repository: page.Metadata.Repository, Homepage: page.Metadata.Homepage},
			VersionOptions:         page.VersionOptions,
		})
	}
	for _, shader := range site.Shaders {
		document.ShaderPages = append(document.ShaderPages, documentationShaderJSON{
			Path: shader.WgslPath, Filename: shader.Filename, Link: shader.Link,
			GithubLink: shader.GithubLink, ImportPath: shader.ImportPath, Dependency: shader.Dependency,
			DeclaredImports: shader.DeclaredImports,
			Items:           documentationShaderItemsJSON{Constants: shader.Consts, Structures: shader.Structures, Functions: shader.Functions, Bindings: shader.Bindings},
		})
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode documentation JSON: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create JSON output directory: %w", err)
	}
	if err := utils.WriteFileIfChanged(filepath.Join(outputDir, "documentation.json"), data, 0o644); err != nil {
		return fmt.Errorf("write documentation JSON: %w", err)
	}
	return nil
}

func documentationSectionsJSON(sections []homeSection) []documentationSectionJSON {
	result := make([]documentationSectionJSON, 0, len(sections))
	for _, section := range sections {
		converted := documentationSectionJSON{Title: section.Title}
		for _, group := range section.Groups {
			converted.Groups = append(converted.Groups, documentationGroupJSON{
				Name: group.Name, PackageName: group.PackageName, Version: group.Version,
				Description: group.Description, Count: group.Count, Files: group.AllFiles,
				DetailPath: group.DetailPath, Preview: group.Preview, Remaining: group.Remaining, Dependency: group.Dependency,
				VersionOptions: group.VersionOptions,
			})
		}
		result = append(result, converted)
	}
	return result
}

func buildDocumentationSite(config config.Config, sections, homeSections []homeSection, totalShaderCount, projectCount, dependencyCount int, files []wgsl.WgslFile, search []ShaderSearchableInfo, metadata discovery.CargoMetadata, declaredImportPaths map[string]string) documentationSite {
	shaders := append([]wgsl.WgslFile(nil), files...)
	for i := range shaders {
		shaders[i].ResolveTypeLinks(declaredImportPaths)
	}

	packagesByKey := make(map[string]documentationPackagePage)
	for _, section := range sections {
		for _, group := range section.Groups {
			page := documentationPackagePage{
				PackageName:            group.PackageName,
				Version:                group.Version,
				BevyVersion:            bevyDependencyVersion(metadata, group.PackageName, group.Version),
				Description:            group.Description,
				Count:                  group.Count,
				AllFiles:               group.AllFiles,
				DetailPath:             group.DetailPath,
				Dependencies:           packageDependencies(shaders, group.PackageName, group.Version),
				DirectDependencies:     cargoDependenciesForPage(metadata, group.PackageName, group.Version, shaders, false),
				TransitiveDependencies: cargoDependenciesForPage(metadata, group.PackageName, group.Version, shaders, true),
				Metadata:               findPackageMetadata(metadata, group.PackageName, group.Version),
				VersionOptions:         packageVersionOptions(config.OutputDir, group.PackageName, group.Version),
				Dependency:             group.Dependency,
			}
			key := group.PackageName + "\x00" + group.Version
			if existing, ok := packagesByKey[key]; !ok || (existing.Dependency && !page.Dependency) {
				packagesByKey[key] = page
			}
		}
	}
	packages := make([]documentationPackagePage, 0, len(packagesByKey))
	for _, page := range packagesByKey {
		packages = append(packages, page)
	}
	slices.SortFunc(packages, func(a, b documentationPackagePage) int {
		if c := strings.Compare(a.PackageName, b.PackageName); c != 0 {
			return c
		}
		return strings.Compare(a.Version, b.Version)
	})

	return documentationSite{
		Version:          config.Version,
		HomeSections:     homeSections,
		Packages:         packages,
		Shaders:          shaders,
		Search:           search,
		TotalShaderCount: totalShaderCount,
		ProjectCount:     projectCount,
		DependencyCount:  dependencyCount,
	}
}

func cargoDependenciesForPage(metadata discovery.CargoMetadata, packageName, version string, files []wgsl.WgslFile, transitive bool) []map[string]string {
	direct, transitiveDependencies := cargoPackageDependencies(metadata, packageName, version, files)
	if transitive {
		return transitiveDependencies
	}
	return direct
}
