package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/faramozzayw/bevy-shader-explorer/utils"
)

type packageRegistryEntry struct {
	PackageName string `json:"packageName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	BevyVersion string `json:"bevyVersion,omitempty"`
	Count       int    `json:"count"`
	DetailPath  string `json:"detailPath"`
}

func updatePackageRegistry(outputDir string, sections []homeSection, bevyVersions map[string]string) ([]packageRegistryEntry, error) {
	registryPath := filepath.Join(outputDir, "public", "packages.json")
	registry, err := readPackageRegistry(outputDir)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]packageRegistryEntry, len(registry))
	for _, entry := range registry {
		byKey[entry.PackageName+"\x00"+entry.Version] = entry
	}
	for _, section := range sections {
		for _, group := range section.Groups {
			key := group.PackageName + "\x00" + group.Version
			byKey[key] = packageRegistryEntry{PackageName: group.PackageName, Description: group.Description, Version: group.Version, BevyVersion: bevyVersions[key], Count: group.Count, DetailPath: group.DetailPath}
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
	if err != nil {
		return nil, fmt.Errorf("encode package registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(registryPath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("create package registry directory: %w", err)
	}
	if err := utils.WriteFileIfChanged(registryPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write package registry %s: %w", registryPath, err)
	}
	return registry, nil
}

// writePendingPackageRegistry records release entries without repeatedly
// rewriting the catalogue while a multi-release build is in progress.
func writePendingPackageRegistry(outputDir string, sections []homeSection, bevyVersions map[string]string) error {
	dir := filepath.Join(outputDir, "public", ".package-registry")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("create pending package registry: %w", err)
	}
	entries := make(map[string]packageRegistryEntry)
	for _, section := range sections {
		for _, group := range section.Groups {
			key := group.PackageName + "\x00" + group.Version
			entries[key] = packageRegistryEntry{PackageName: group.PackageName, Description: group.Description, Version: group.Version, BevyVersion: bevyVersions[key], Count: group.Count, DetailPath: group.DetailPath}
		}
	}
	for key, entry := range entries {
		path := filepath.Join(dir, packageSearchIndex(entry.PackageName, entry.Version)+".json")
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("encode pending package registry %s: %w", key, err)
		}
		if err := utils.WriteFileIfChanged(path, data, 0o644); err != nil {
			return fmt.Errorf("write pending package registry %s: %w", path, err)
		}
	}
	return nil
}

func readPendingPackageRegistry(outputDir string) ([]packageRegistryEntry, error) {
	dir := filepath.Join(outputDir, "public", ".package-registry")
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read pending package registry: %w", err)
	}
	entries := make([]packageRegistryEntry, 0, len(files))
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("read pending package registry %s: %w", file.Name(), err)
		}
		var entry packageRegistryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, fmt.Errorf("decode pending package registry %s: %w", file.Name(), err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func mergePackageRegistry(base, additions []packageRegistryEntry) []packageRegistryEntry {
	byKey := make(map[string]packageRegistryEntry, len(base)+len(additions))
	for _, entry := range append(base, additions...) {
		byKey[entry.PackageName+"\x00"+entry.Version] = entry
	}
	entries := make([]packageRegistryEntry, 0, len(byKey))
	for _, entry := range byKey {
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(a, b packageRegistryEntry) int {
		if c := strings.Compare(a.PackageName, b.PackageName); c != 0 {
			return c
		}
		return strings.Compare(b.Version, a.Version)
	})
	return entries
}

func writePackageRegistry(outputDir string, registry []packageRegistryEntry) error {
	path := filepath.Join(outputDir, "public", "packages.json")
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("encode package registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("create package registry directory: %w", err)
	}
	return utils.WriteFileIfChanged(path, data, 0o644)
}

func readPackageRegistry(outputDir string) ([]packageRegistryEntry, error) {
	registryPath := filepath.Join(outputDir, "public", "packages.json")
	registry := make([]packageRegistryEntry, 0)
	data, err := os.ReadFile(registryPath)
	if err == nil {
		if err := json.Unmarshal(data, &registry); err != nil {
			return nil, fmt.Errorf("decode package registry %s: %w", registryPath, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read package registry %s: %w", registryPath, err)
	}
	return registry, nil
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
