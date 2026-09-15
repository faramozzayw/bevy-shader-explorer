package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writePackageVersionsManifest(outputDir string) error {
	packages := make(map[string][]map[string]string)
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("read output directory for package versions: %w", err)
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
		return fmt.Errorf("encode package versions: %w", err)
	}
	publicDir := filepath.Join(outputDir, "public")
	if err := os.MkdirAll(publicDir, os.ModePerm); err != nil {
		return fmt.Errorf("create public directory for package versions: %w", err)
	}
	manifestPath := filepath.Join(publicDir, "package-versions.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("write package versions %s: %w", manifestPath, err)
	}
	return nil
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
