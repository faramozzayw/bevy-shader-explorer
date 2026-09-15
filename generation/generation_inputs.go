package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	config "main/config"
	"main/discovery"
	"main/utils"

	"github.com/aymerick/raymond"
)

func renderTemplateToFile(templateSrc string, context map[string]interface{}, outputPath string) error {
	tmpl, err := raymond.Parse(templateSrc)
	if err != nil {
		return err
	}
	html, err := tmpl.Exec(context)
	if err != nil {
		return err
	}
	err = os.WriteFile(outputPath, []byte(html), 0644)
	if err != nil {
		return err
	}
	return nil
}

func getWgslFilesList(cfg config.Config) ([]string, error) {
	var filePaths []string
	err := filepath.WalkDir(cfg.SourcePath, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filePath != cfg.SourcePath && discovery.IsExcludedPath(cfg.SourcePath, filePath, cfg.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		if config.MatchesShaderFile(cfg.FileFilter, entry.Name()) && !discovery.IsExcludedPath(cfg.SourcePath, filePath, cfg.Exclude) {
			filePaths = append(filePaths, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", cfg.SourcePath, err)
	}
	slices.Sort(filePaths)
	return filePaths, nil
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

func getShaderInputs(config config.Config) ([]shaderInput, error) {
	projectFiles, err := getWgslFilesList(config)
	if err != nil {
		return nil, err
	}
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
		return inputs, nil
	}
	manifestPath := filepath.Join(config.SourcePath, "Cargo.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		return inputs, nil
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
		return inputs, nil
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
		return inputs, nil
	}
	for _, dependency := range dependencies {
		pkg := findPackageMetadata(metadata, dependency.Package, dependency.Version)
		manifest := pkg.ManifestPath
		if manifest == "" {
			continue
		}
		dependencyConfig := config
		dependencyConfig.SourcePath = filepath.Dir(manifest)
		dependencyConfig.SourceGithubRoot = discovery.RepositoryRootForPackage(manifest)
		// Cargo metadata records the upstream repository for each resolved
		// package. Preserve it so dependency shader pages can link back to the
		// exact source file just like project shaders do.
		dependencyConfig.SourceGithubURL = pkg.Repository
		dependencyConfig.SourceGithubSubpath = discovery.RepositorySubpathForPackage(manifest, dependencyConfig.SourceGithubURL, dependency.Package)
		if dependencyConfig.SourceGithubURL != config.SourceGithubURL {
			dependencyConfig.SourceGithubRef = packageSourceRef(pkg.Source)
		}
		if seenPaths[dependency.Path] {
			continue
		}
		seenPaths[dependency.Path] = true
		inputs = append(inputs, shaderInput{Path: dependency.Path, Config: dependencyConfig, Prefix: filepath.Join(dependency.Package, dependency.Version), Dependency: true, PackageName: dependency.Package, PackageDescription: pkg.Description, PackageVersion: dependency.Version})
	}
	return inputs, nil
}

func copyItemsToPublic(config *config.Config, searchInfo []ShaderSearchableInfo) error {
	publicDir := filepath.Join(config.OutputDir, "public")
	err := os.MkdirAll(publicDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("create public directory: %w", err)
	}

	searchInfoJSON, err := json.MarshalIndent(searchInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal search index: %w", err)
	}

	err = os.WriteFile(filepath.Join(publicDir, fmt.Sprintf("search-info-%s.json", config.Version)), searchInfoJSON, 0644)
	if err != nil {
		return fmt.Errorf("write search index: %w", err)
	}

	for _, file := range copyToPublic {
		src := file
		dst := filepath.Join(publicDir, filepath.Base(file))
		err := utils.CopyFile(src, dst)
		if err != nil {
			return fmt.Errorf("copy public asset %s: %w", src, err)
		}
	}
	return nil
}
