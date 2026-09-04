// Package discovery finds shader-bearing project and dependency sources.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"main/config"
)

// ShaderSource identifies a WGSL file and the package that owns it.
type ShaderSource struct {
	Path       string
	Package    string
	Version    string
	Dependency bool
}

// CargoMetadata is the stable subset of `cargo metadata` used by discovery.
type CargoMetadata struct {
	Packages      []CargoPackage `json:"packages"`
	WorkspaceRoot string         `json:"workspace_root"`
	Resolve       *CargoResolve  `json:"resolve"`
}

// cargoMetadataCommand is kept injectable so discovery can be tested without
// requiring a Cargo installation, registry access, or a populated cache.
var cargoMetadataCommand = runCargoMetadata

// FilterCargoPackages selects packages whose names match patterns and, when
// transitive is enabled, all packages reachable through their dependencies.
func FilterCargoPackages(metadata CargoMetadata, patterns []string, transitive bool) []CargoPackage {
	packagesByID := make(map[string]CargoPackage, len(metadata.Packages))
	for _, pkg := range metadata.Packages {
		packagesByID[pkg.ID] = pkg
	}
	nodesByID := make(map[string]CargoNode, len(metadata.Resolve.Nodes))
	if metadata.Resolve != nil {
		for _, node := range metadata.Resolve.Nodes {
			nodesByID[node.ID] = node
		}
	}

	selected := make(map[string]bool)
	queue := make([]string, 0)
	for _, pkg := range metadata.Packages {
		if matchesPackagePattern(pkg.Name, patterns) {
			selected[pkg.ID] = true
			queue = append(queue, pkg.ID)
		}
	}
	if transitive {
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			for _, dependency := range nodesByID[id].Deps {
				if !selected[dependency.PackageID] {
					selected[dependency.PackageID] = true
					queue = append(queue, dependency.PackageID)
				}
			}
		}
	}

	result := make([]CargoPackage, 0, len(selected))
	for id := range selected {
		if pkg, ok := packagesByID[id]; ok {
			result = append(result, pkg)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].Version < result[j].Version
	})
	return result
}

func matchesPackagePattern(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// DiscoverDependencyShaders scans only the supplied package roots for WGSL
// files. Package roots come from Cargo manifest paths, so unrelated resolved
// packages are never traversed.
func DiscoverDependencyShaders(packages []CargoPackage, excludes []string) ([]ShaderSource, error) {
	var result []ShaderSource
	for _, pkg := range packages {
		root := filepath.Dir(pkg.ManifestPath)
		files, err := discoverWGSLFiles(root, excludes)
		if err != nil {
			return nil, fmt.Errorf("scan package %s: %w", pkg.Name, err)
		}
		for _, file := range files {
			result = append(result, ShaderSource{Path: file, Package: pkg.Name, Version: pkg.Version, Dependency: true})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result, nil
}

// PackageForPath returns the workspace package whose manifest directory owns
// filePath. The longest matching root wins, which handles nested fixtures.
func PackageForPath(packages []CargoPackage, filePath string) (CargoPackage, bool) {
	var match CargoPackage
	best := -1
	for _, pkg := range packages {
		root := filepath.Dir(pkg.ManifestPath)
		rel, err := filepath.Rel(root, filePath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if len(root) > best {
			match, best = pkg, len(root)
		}
	}
	return match, best >= 0
}

func discoverWGSLFiles(root string, excludes []string) ([]string, error) {
	var result []string
	err := filepath.WalkDir(root, func(filePath string, entryDirent fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entryDirent.IsDir() {
			if filePath != root && excludedPath(root, filePath, excludes) {
				return filepath.SkipDir
			}
			return nil
		}
		if config.MatchesShaderFile("*.wgsl", filepath.Base(filePath)) && !excludedPath(root, filePath, excludes) {
			result = append(result, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(result)
	return result, nil
}

func excludedPath(root, filePath string, excludes []string) bool {
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

type CargoPackage struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	ManifestPath string   `json:"manifest_path"`
	Source       string   `json:"source"`
	Description  string   `json:"description"`
	Authors      []string `json:"authors"`
	License      string   `json:"license"`
	Repository   string   `json:"repository"`
	Homepage     string   `json:"homepage"`
}

type CargoResolve struct {
	Nodes []CargoNode `json:"nodes"`
}

type CargoNode struct {
	ID   string            `json:"id"`
	Deps []CargoDependency `json:"deps"`
}

type CargoDependency struct {
	PackageID string `json:"pkg"`
}

// ReadCargoMetadata runs Cargo for a project and decodes its resolved package
// graph. Cargo remains responsible for resolution, including network access in
// normal mode and cache-only behavior when offline is true.
func ReadCargoMetadata(ctx context.Context, projectPath string, offline bool) (CargoMetadata, error) {
	manifestPath := projectPath
	if filepath.Base(manifestPath) != "Cargo.toml" {
		manifestPath = filepath.Join(manifestPath, "Cargo.toml")
	}

	output, err := cargoMetadataCommand(ctx, manifestPath, offline)
	if err != nil {
		mode := ""
		if offline {
			mode = " in offline mode"
		}
		return CargoMetadata{}, fmt.Errorf("cargo metadata%s: %w: %s", mode, err, strings.TrimSpace(string(output)))
	}

	var metadata CargoMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return CargoMetadata{}, fmt.Errorf("decode cargo metadata: %w", err)
	}
	return metadata, nil
}

func runCargoMetadata(ctx context.Context, manifestPath string, offline bool) ([]byte, error) {
	args := []string{"metadata", "--format-version", "1", "--manifest-path", manifestPath}
	if offline {
		args = append(args, "--offline")
	}
	command := exec.CommandContext(ctx, "cargo", args...)
	output, err := command.Output()
	if err == nil {
		// Cargo emits progress and warnings on stderr even when metadata JSON is
		// valid. Only stdout is part of the machine-readable response.
		return output, nil
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return append(output, exitError.Stderr...), err
	}
	return output, err
}
