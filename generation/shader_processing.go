package generation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/faramozzayw/bevy-shader-explorer/utils"
	"github.com/faramozzayw/bevy-shader-explorer/wgsl"
)

// parseShaderInputs parses files concurrently while storing results by input
// index. Keeping the original order makes collision resolution and generated
// navigation deterministic regardless of worker scheduling.
func parseShaderInputs(inputs []shaderInput, projectVersion, outputDir string, progress *generationProgress) ([]wgsl.WgslFile, error) {
	files := make([]wgsl.WgslFile, len(inputs))
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers > len(inputs) {
		workers = len(inputs)
	}
	if workers == 0 {
		return files, nil
	}

	jobs := make(chan int)
	errs := make(chan error, len(inputs))
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				input := inputs[index]
				file, err := parseShaderInputCached(input)
				if err != nil {
					errs <- fmt.Errorf("parse %s: %w", input.Path, err)
					continue
				}
				if input.Prefix != "" {
					file.WgslPath = filepath.Join(input.Prefix, file.WgslPath)
				}
				file.Dependency = input.Dependency
				file.SourcePath = input.Path
				file.SourceRoot = input.Config.SourcePath
				file.OutputPrefix = input.Prefix
				file.ProjectName = input.PackageName
				file.ProjectDescription = input.PackageDescription
				file.ProjectVersion = input.PackageVersion
				if file.ProjectVersion == "" {
					file.ProjectVersion = projectVersion
				}
				file.ProjectURLPrefix = joinDocURL("project", "")
				file.PackageURLPrefix = joinDocURL("project", filepath.Join(input.PackageName, input.PackageVersion))
				file.VersionOptions = packageVersionOptions(outputDir, input.PackageName, input.PackageVersion)
				file.SeoDescription = fmt.Sprintf("WGSL shader module %s from %s %s.", file.Filename, file.ProjectName, file.ProjectVersion)
				file.SeoTitle = fmt.Sprintf("%s — %s", file.Filename, file.ProjectName)
				file.WgslPath = strings.Replace(file.WgslPath, "src/", "", 1)
				file.WgslPath = utils.DedupPathParts(file.WgslPath)
				files[index] = file
				progress.addReading()
			}
		}()
	}
	for index := range inputs {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	close(errs)
	if err := firstError(errs); err != nil {
		return nil, err
	}
	return files, nil
}

const shaderParseCacheVersion = "shader-parse-v3"

// parseShaderInputCached reuses the parser model for identical source content
// and link-affecting configuration. The cache is outside dist so it never
// becomes a deployable artifact, and its key includes the source bytes plus
// every setting that can change ParseWGSLFile's result.
func parseShaderInputCached(input shaderInput) (wgsl.WgslFile, error) {
	content, err := os.ReadFile(input.Path)
	if err != nil {
		return wgsl.WgslFile{}, fmt.Errorf("read source: %w", err)
	}
	hash := sha256.New()
	hash.Write([]byte(shaderParseCacheVersion))
	hash.Write([]byte("\x00"))
	hash.Write(content)
	for _, value := range []string{
		input.Config.SourcePath, input.Config.SourceGithubRoot, input.Config.SourceGithubSubpath,
		input.Config.SourceGithubURL, input.Config.SourceGithubRef, input.Config.Version,
		input.Config.FileFilter, input.Config.ProjectVersion, input.Path,
	} {
		hash.Write([]byte("\x00"))
		hash.Write([]byte(value))
	}
	cacheDir := filepath.Join(os.TempDir(), "bevy-shader-explorer-cache")
	cachePath := filepath.Join(cacheDir, fmt.Sprintf("%x.json", hash.Sum(nil)))
	if cached, readErr := os.ReadFile(cachePath); readErr == nil {
		var file wgsl.WgslFile
		if json.Unmarshal(cached, &file) == nil {
			file.BuildShaderDefs()
			return file, nil
		}
	}
	file, err := wgsl.ParseWGSLFile(&input.Config, input.Path)
	if err != nil {
		return wgsl.WgslFile{}, err
	}
	file.BuildShaderDefs()
	if data, marshalErr := json.Marshal(file); marshalErr == nil {
		if mkdirErr := os.MkdirAll(cacheDir, 0o755); mkdirErr == nil {
			_ = os.WriteFile(cachePath, data, 0o644)
		}
	}
	return file, nil
}

func firstError(errs <-chan error) error {
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveWgslPathCollisions(files []wgsl.WgslFile) {
	byStem := make(map[string][]int)
	for i, file := range files {
		byStem[strings.TrimSuffix(file.WgslPath, ".html")] = append(byStem[strings.TrimSuffix(file.WgslPath, ".html")], i)
	}
	for _, indexes := range byStem {
		if len(indexes) < 2 {
			continue
		}
		extensions := make(map[string]bool)
		for _, index := range indexes {
			extensions[strings.TrimPrefix(filepath.Ext(files[index].Filename), ".")] = true
		}
		if len(extensions) > 1 {
			for _, index := range indexes {
				extension := strings.TrimPrefix(filepath.Ext(files[index].Filename), ".")
				files[index].WgslPath = strings.TrimSuffix(files[index].WgslPath, ".html") + "." + extension + ".html"
			}
		}
	}

	byPath := make(map[string][]int)
	for i, file := range files {
		byPath[file.WgslPath] = append(byPath[file.WgslPath], i)
	}
	for _, indexes := range byPath {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			dir, err := filepath.Rel(files[index].SourceRoot, filepath.Dir(files[index].SourcePath))
			if err != nil {
				continue
			}
			parts := []string{"root"}
			if dir != "." {
				parts = strings.Split(filepath.ToSlash(dir), "/")
			}
			for width := 1; width <= len(parts); width++ {
				suffix := strings.Join(parts[len(parts)-width:], "/")
				unique := true
				for _, other := range indexes {
					if other == index {
						continue
					}
					otherDir, _ := filepath.Rel(files[other].SourceRoot, filepath.Dir(files[other].SourcePath))
					otherParts := []string{"root"}
					if otherDir != "." {
						otherParts = strings.Split(filepath.ToSlash(otherDir), "/")
					}
					if len(otherParts) >= width && strings.Join(otherParts[len(otherParts)-width:], "/") == suffix {
						unique = false
						break
					}
				}
				if unique {
					files[index].WgslPath = filepath.ToSlash(filepath.Join(files[index].OutputPrefix, suffix, filepath.Base(files[index].WgslPath)))
					break
				}
			}
		}
	}
}

// moduleLabels returns the compact names shown in package shader lists. Most
// files can use their basename, but repeated basenames need a path suffix so
// users can tell them apart (for example, deferred/top and prepass/top).
func moduleLabels(files []wgsl.WgslFile) []string {
	labels := make([]string, len(files))
	groups := make(map[string][]int)
	for i, file := range files {
		labels[i] = strings.TrimSuffix(filepath.Base(file.WgslPath), ".html")
		key := file.ProjectName + "\x00" + file.ProjectVersion + "\x00" + labels[i]
		groups[key] = append(groups[key], i)
	}

	for _, indexes := range groups {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			parts := sourceDirectoryParts(files[index])
			for width := 1; width <= len(parts); width++ {
				suffix := strings.Join(parts[len(parts)-width:], "/")
				unique := true
				for _, other := range indexes {
					if other == index {
						continue
					}
					otherParts := sourceDirectoryParts(files[other])
					if len(otherParts) >= width && strings.Join(otherParts[len(otherParts)-width:], "/") == suffix {
						unique = false
						break
					}
				}
				if unique {
					labels[index] = suffix + "/" + labels[index]
					break
				}
			}
		}
	}
	return labels
}

func sourceDirectoryParts(file wgsl.WgslFile) []string {
	if file.SourcePath == "" || file.SourceRoot == "" {
		dir := filepath.ToSlash(filepath.Dir(file.WgslPath))
		if dir == "." || dir == "" {
			return []string{"root"}
		}
		return strings.Split(dir, "/")
	}
	dir, err := filepath.Rel(file.SourceRoot, filepath.Dir(file.SourcePath))
	if err != nil || dir == "." || dir == "" {
		return []string{"root"}
	}
	return strings.Split(filepath.ToSlash(dir), "/")
}

func appendSearchInfo(searchInfo *[]ShaderSearchableInfo, imports *map[string]string, file wgsl.WgslFile) {
	link := utils.NormalizeLink(file.Link)
	exportable := file.ImportPath != nil
	if exportable {
		(*imports)[*file.ImportPath] = link
	}
	for _, fn := range file.Functions {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: fn.Name, Type: "function", StageAttribute: fn.StageAttribute, Comment: fn.Comment})
	}
	for _, item := range file.Structures {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: item.Name, Type: "struct"})
	}
	for _, item := range file.Consts {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: item.Name, Type: "const"})
	}
	for _, item := range file.Bindings {
		*searchInfo = append(*searchInfo, ShaderSearchableInfo{Link: link, PackageName: file.ProjectName, PackageVersion: file.ProjectVersion, Filename: file.Filename, Exportable: exportable, Name: item.Name, Type: "binding"})
	}
}
