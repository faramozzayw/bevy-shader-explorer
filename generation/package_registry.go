package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type packageRegistryEntry struct {
	PackageName string `json:"packageName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Count       int    `json:"count"`
	DetailPath  string `json:"detailPath"`
}

func updatePackageRegistry(outputDir string, sections []homeSection) ([]packageRegistryEntry, error) {
	registryPath := filepath.Join(outputDir, "public", "packages.json")
	registry := make([]packageRegistryEntry, 0)
	if data, err := os.ReadFile(registryPath); err == nil {
		if err := json.Unmarshal(data, &registry); err != nil {
			return nil, fmt.Errorf("decode package registry %s: %w", registryPath, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read package registry %s: %w", registryPath, err)
	}
	byKey := make(map[string]packageRegistryEntry, len(registry))
	for _, entry := range registry {
		byKey[entry.PackageName+"\x00"+entry.Version] = entry
	}
	for _, section := range sections {
		for _, group := range section.Groups {
			key := group.PackageName + "\x00" + group.Version
			byKey[key] = packageRegistryEntry{PackageName: group.PackageName, Description: group.Description, Version: group.Version, Count: group.Count, DetailPath: group.DetailPath}
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
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write package registry %s: %w", registryPath, err)
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
