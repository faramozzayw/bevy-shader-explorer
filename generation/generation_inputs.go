package generation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	config "github.com/faramozzayw/bevy-shader-explorer/config"
	"github.com/faramozzayw/bevy-shader-explorer/discovery"
	"github.com/faramozzayw/bevy-shader-explorer/utils"

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
	err = utils.WriteFileIfChanged(outputPath, []byte(html), 0644)
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

// getShaderInputs discovers all shader inputs and returns the Cargo metadata used
// for that discovery. Keeping metadata with the inputs prevents the caller from
// resolving the same Cargo graph a second time while building page models.
func getShaderInputs(config config.Config) ([]shaderInput, discovery.CargoMetadata, error) {
	projectFiles, err := getWgslFilesList(config)
	if err != nil {
		return nil, discovery.CargoMetadata{}, err
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
		return inputs, discovery.CargoMetadata{}, nil
	}
	manifestPath := filepath.Join(config.SourcePath, "Cargo.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		return inputs, discovery.CargoMetadata{}, nil
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
		return inputs, discovery.CargoMetadata{}, nil
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
	packageMetadata := make(map[string]discovery.CargoPackage, len(metadata.Packages))
	for _, pkg := range metadata.Packages {
		packageMetadata[pkg.Name+"\x00"+pkg.Version] = pkg
	}
	dependencies, err := discovery.DiscoverDependencyShaders(packages, config.Exclude)
	if err != nil {
		log.Printf("warning: dependency shader discovery skipped: %v", err)
		return inputs, metadata, nil
	}
	repositoryRoots := make(map[string]string)
	for _, dependency := range dependencies {
		pkg := packageMetadata[dependency.Package+"\x00"+dependency.Version]
		manifest := pkg.ManifestPath
		if manifest == "" {
			continue
		}
		dependencyConfig := config
		dependencyConfig.SourcePath = filepath.Dir(manifest)
		repositoryRoot, ok := repositoryRoots[manifest]
		if !ok {
			repositoryRoot = discovery.RepositoryRootForPackage(manifest)
			repositoryRoots[manifest] = repositoryRoot
		}
		dependencyConfig.SourceGithubRoot = repositoryRoot
		// Cargo metadata records the upstream repository for each resolved
		// package. Preserve it so dependency shader pages can link back to the
		// exact source file just like project shaders do.
		dependencyConfig.SourceGithubURL = pkg.Repository
		dependencyConfig.SourceGithubSubpath = discovery.RepositorySubpathForPackageFromRoot(manifest, dependencyConfig.SourceGithubURL, dependency.Package, repositoryRoot)
		if dependencyConfig.SourceGithubURL != config.SourceGithubURL {
			dependencyConfig.SourceGithubRef = packageSourceRef(pkg.Source)
		}
		if seenPaths[dependency.Path] {
			continue
		}
		seenPaths[dependency.Path] = true
		inputs = append(inputs, shaderInput{Path: dependency.Path, Config: dependencyConfig, Prefix: filepath.Join(dependency.Package, dependency.Version), Dependency: true, PackageName: dependency.Package, PackageDescription: pkg.Description, PackageVersion: dependency.Version})
	}
	return inputs, metadata, nil
}

func copyItemsToPublic(config *config.Config, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error {
	publicDir := filepath.Join(config.OutputDir, "public")
	err := os.MkdirAll(publicDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("create public directory: %w", err)
	}

	if err := writeSearchIndexes(publicDir, searchInfo, packages); err != nil {
		return err
	}
	return copyStaticAssets(publicDir)
}

func writeSearchIndexes(publicDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error {
	byPackage := make(map[string][]ShaderSearchableInfo)
	for _, page := range packages {
		byPackage[packageSearchIndex(page.PackageName, page.Version)] = make([]ShaderSearchableInfo, 0)
	}
	for _, item := range searchInfo {
		key := packageSearchIndex(item.PackageName, item.PackageVersion)
		byPackage[key] = append(byPackage[key], item)
	}
	for key, items := range byPackage {
		searchInfoJSON, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal search index %s: %w", key, err)
		}
		err = utils.WriteFileIfChanged(filepath.Join(publicDir, fmt.Sprintf("search-info-%s.json", key)), searchInfoJSON, 0644)
		if err != nil {
			return fmt.Errorf("write search index %s: %w", key, err)
		}
	}
	return nil
}

func writePendingSearchIndexes(outputDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error {
	dir := filepath.Join(outputDir, "public", ".search-index")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("create pending search indexes: %w", err)
	}
	return writeSearchIndexes(dir, searchInfo, packages)
}

func finalizeSearchIndexes(outputDir string) error {
	dir := filepath.Join(outputDir, "public", ".search-index")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read pending search indexes: %w", err)
	}
	publicDir := filepath.Join(outputDir, "public")
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read pending search index %s: %w", entry.Name(), err)
		}
		if err := utils.WriteFileIfChanged(filepath.Join(publicDir, entry.Name()), data, 0o644); err != nil {
			return fmt.Errorf("write search index %s: %w", entry.Name(), err)
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove pending search indexes: %w", err)
	}
	return nil
}

func packageSearchIndex(packageName, version string) string {
	value := ogSlugPattern.ReplaceAllString(packageName+"-"+version, "-")
	return strings.Trim(value, "-")
}

func copyStaticAssets(publicDir string) error {
	if err := os.MkdirAll(publicDir, os.ModePerm); err != nil {
		return fmt.Errorf("create public directory: %w", err)
	}
	manifestPath := filepath.Join(publicDir, ".assets-manifest.json")
	previousManifest := map[string]string{}
	if data, readErr := os.ReadFile(manifestPath); readErr == nil {
		_ = json.Unmarshal(data, &previousManifest)
	}
	currentManifest := make(map[string]string, len(copyToPublic))
	for _, file := range copyToPublic {
		src := file
		dst := filepath.Join(publicDir, filepath.Base(file))
		data, readErr := os.ReadFile(src)
		if readErr != nil {
			return fmt.Errorf("read public asset %s: %w", src, readErr)
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		currentManifest[src] = digest
		if previousManifest[src] == digest {
			continue
		}
		err := utils.CopyFile(src, dst)
		if err != nil {
			return fmt.Errorf("copy public asset %s: %w", src, err)
		}
	}
	manifest, err := json.MarshalIndent(currentManifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode asset manifest: %w", err)
	}
	if err := utils.WriteFileIfChanged(manifestPath, manifest, 0644); err != nil {
		return fmt.Errorf("write asset manifest: %w", err)
	}
	return nil
}

func copyOGMascot(publicDir string) error {
	if err := os.MkdirAll(publicDir, os.ModePerm); err != nil {
		return fmt.Errorf("create public directory: %w", err)
	}
	return utils.CopyFile("mascot2.jpeg", filepath.Join(publicDir, "mascot2.jpeg"))
}
