package generation

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/faramozzayw/bevy-shader-explorer/discovery"
	"github.com/faramozzayw/bevy-shader-explorer/wgsl"
)

func findPackageMetadata(metadata discovery.CargoMetadata, name, version string) discovery.CargoPackage {
	for _, pkg := range metadata.Packages {
		if pkg.Name == name && pkg.Version == version {
			return pkg
		}
	}
	return discovery.CargoPackage{}
}

func packageSourceRef(source string) string {
	if hash := strings.LastIndexByte(source, '#'); hash >= 0 && hash+1 < len(source) {
		return source[hash+1:]
	}
	return ""
}

func dependencyLink(name, version string) map[string]string {
	return map[string]string{
		"name":    name,
		"version": version,
		"url":     filepath.ToSlash(filepath.Join(name, version, "index.html")),
	}
}

func sortDependencyLinks(dependencies []map[string]string) {
	slices.SortFunc(dependencies, func(a, b map[string]string) int {
		return strings.Compare(a["name"], b["name"])
	})
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
		dependencies = append(dependencies, dependencyLink(file.ProjectName, file.ProjectVersion))
	}
	sortDependencyLinks(dependencies)
	return dependencies
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
			item := dependencyLink(pkg.Name, pkg.Version)
			if depth[dep.PackageID] == 1 {
				direct = append(direct, item)
			} else {
				transitive = append(transitive, item)
			}
		}
	}
	sortDependencyLinks(direct)
	sortDependencyLinks(transitive)
	return direct, transitive
}

func bevyDependencyVersion(metadata discovery.CargoMetadata, name, version string) string {
	ids := make(map[string]string, len(metadata.Packages))
	packages := make(map[string]discovery.CargoPackage, len(metadata.Packages))
	for _, pkg := range metadata.Packages {
		ids[pkg.Name+"\x00"+pkg.Version] = pkg.ID
		packages[pkg.ID] = pkg
	}
	rootID := ids[name+"\x00"+version]
	if rootID == "" {
		return ""
	}
	if name == "bevy" {
		return version
	}
	if metadata.Resolve == nil {
		return ""
	}
	nodes := make(map[string]discovery.CargoNode, len(metadata.Resolve.Nodes))
	for _, node := range metadata.Resolve.Nodes {
		nodes[node.ID] = node
	}
	seen := map[string]bool{rootID: true}
	queue := []string{rootID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, dependency := range nodes[id].Deps {
			if seen[dependency.PackageID] {
				continue
			}
			seen[dependency.PackageID] = true
			pkg, ok := packages[dependency.PackageID]
			if !ok {
				continue
			}
			if pkg.Name == "bevy" {
				return pkg.Version
			}
			queue = append(queue, dependency.PackageID)
		}
	}
	return ""
}
