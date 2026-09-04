package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// MatchesShaderFile keeps the historical *.wgsl default while also treating
// WESL files as shaders. A custom file_filter remains exact and opt-in.
func MatchesShaderFile(filter, name string) bool {
	if filter == "*.wgsl" && strings.EqualFold(filepath.Ext(name), ".wesl") {
		return true
	}
	matched, _ := path.Match(filter, name)
	return matched
}

type Config struct {
	Name           string
	Description    string
	ProjectVersion string
	PackageName    string
	SourcePath     string
	// SourceGithubRoot is the repository root used for source links. It can
	// differ from SourcePath when a shader belongs to a workspace member.
	SourceGithubRoot     string
	SourceGithubSubpath  string
	FileFilter           string
	OutputDir            string
	SourceGithubURL      string
	SourceGithubRef      string
	Version              string
	Exclude              []string
	NoDeps               bool
	Offline              bool
	DependencyInclude    []string
	DependencyTransitive bool
}

// Load reads the optional project-local wgsl-docs.toml file. Defaults are
// returned when the file does not exist.
func Load(projectPath string) (Config, error) {
	cfg := Config{
		SourcePath:           projectPath,
		FileFilter:           "*.wgsl",
		OutputDir:            "./shader-docs",
		Version:              "project",
		Name:                 "WGSL Documentation",
		DependencyInclude:    []string{"bevy", "bevy_*"},
		DependencyTransitive: true,
	}
	filePath := filepath.Join(projectPath, "wgsl-docs.toml")
	var file fileConfig
	metadata, err := toml.DecodeFile(filePath, &file)
	if err != nil {
		if !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("parse %s: %w", filePath, err)
		}
	} else {
		if unknown := metadata.Undecoded(); len(unknown) > 0 {
			return Config{}, fmt.Errorf("unknown settings in %s: %v", filePath, unknown)
		}
		if file.Output != "" {
			cfg.OutputDir = resolveRelative(projectPath, file.Output)
		}
		if file.FileFilter != "" {
			cfg.FileFilter = file.FileFilter
		}
		cfg.Exclude = file.Exclude
		cfg.NoDeps = file.Dependencies.Enabled != nil && !*file.Dependencies.Enabled
		cfg.Offline = file.Dependencies.Offline
		if file.Dependencies.Include != nil {
			cfg.DependencyInclude = file.Dependencies.Include
		}
		if file.Dependencies.Transitive != nil {
			cfg.DependencyTransitive = *file.Dependencies.Transitive
		}
	}
	if file.Project != "" {
		cfg.SourcePath = resolveRelative(projectPath, file.Project)
	}
	cfg.SourceGithubRoot = cfg.SourcePath
	cargoName, cargoDescription, cargoVersion := loadCargoMetadata(cfg.SourcePath)
	if cargoName != "" {
		cfg.Name = cargoName
		cfg.PackageName = cargoName
	}
	cfg.Description = cargoDescription
	cfg.ProjectVersion = cargoVersion
	cfg.SourceGithubURL = loadCargoRepository(cfg.SourcePath)
	if file.Name != "" {
		cfg.Name = file.Name
	}
	if file.Description != "" {
		cfg.Description = file.Description
	}
	return cfg, nil
}

func resolveRelative(root, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}

type fileConfig struct {
	Name         string           `toml:"name"`
	Description  string           `toml:"description"`
	Project      string           `toml:"project"`
	Output       string           `toml:"output"`
	FileFilter   string           `toml:"file_filter"`
	Exclude      []string         `toml:"exclude"`
	Dependencies dependencyConfig `toml:"dependencies"`
}

type cargoManifest struct {
	Package struct {
		Name        string      `toml:"name"`
		Description string      `toml:"description"`
		Version     interface{} `toml:"version"`
		Repository  interface{} `toml:"repository"`
	} `toml:"package"`
	Workspace struct {
		Package struct {
			Version string `toml:"version"`
		} `toml:"package"`
	} `toml:"workspace"`
}

func loadCargoRepository(projectPath string) string {
	var manifest cargoManifest
	if _, err := toml.DecodeFile(filepath.Join(projectPath, "Cargo.toml"), &manifest); err != nil {
		return ""
	}
	if repository, ok := manifest.Package.Repository.(string); ok {
		return repository
	}
	return ""
}

func loadCargoMetadata(projectPath string) (string, string, string) {
	var manifest cargoManifest
	if _, err := toml.DecodeFile(filepath.Join(projectPath, "Cargo.toml"), &manifest); err != nil {
		return "", "", ""
	}
	version := versionValue(manifest.Package.Version)
	if version == "" {
		version = manifest.Workspace.Package.Version
	}
	return manifest.Package.Name, manifest.Package.Description, version
}

func versionValue(value interface{}) string {
	if version, ok := value.(string); ok {
		return version
	}
	return ""
}

type dependencyConfig struct {
	Enabled    *bool    `toml:"enabled"`
	Offline    bool     `toml:"offline"`
	Include    []string `toml:"include"`
	Transitive *bool    `toml:"transitive"`
}
