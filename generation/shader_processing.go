package generation

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	config "github.com/faramozzayw/bevy-shader-explorer/config"
	wgsl "github.com/faramozzayw/bevy-shader-explorer/shader"
	"github.com/faramozzayw/bevy-shader-explorer/utils"
)

type shaderFileParser interface {
	Parse(*config.Config, wgsl.Source) (wgsl.ShaderFile, error)
}

type defaultShaderFileParser struct{}

func (defaultShaderFileParser) Parse(cfg *config.Config, source wgsl.Source) (wgsl.ShaderFile, error) {
	return wgsl.ParseSource(cfg, source)
}

type shaderDocumentEnricher interface {
	Enrich(wgsl.ShaderFile, shaderInput, GenerationContext) wgsl.ShaderFile
}

type defaultShaderDocumentEnricher struct{}

func (defaultShaderDocumentEnricher) Enrich(file wgsl.ShaderFile, input shaderInput, context GenerationContext) wgsl.ShaderFile {
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
		file.ProjectVersion = context.ProjectVersion
	}
	file.ProjectURLPrefix = joinDocURL("project", "")
	file.PackageURLPrefix = joinDocURL("project", filepath.Join(input.PackageName, input.PackageVersion))
	file.VersionOptions = packageVersionOptions(context.OutputDir, input.PackageName, input.PackageVersion)
	file.SeoDescription = fmt.Sprintf("WGSL shader module %s from %s %s.", file.Filename, file.ProjectName, file.ProjectVersion)
	file.SeoTitle = fmt.Sprintf("%s — %s", file.Filename, file.ProjectName)
	file.WgslPath = strings.Replace(file.WgslPath, "src/", "", 1)
	file.WgslPath = utils.DedupPathParts(file.WgslPath)
	return file
}

// parseShaderInputs parses files concurrently while storing results by input
// index. Keeping the original order makes collision resolution and generated
// navigation deterministic regardless of worker scheduling.
func parseShaderInputs(context GenerationContext, inputs []shaderInput, progress *generationProgress, parser shaderFileParser, cache shaderParseCache, enricher shaderDocumentEnricher) ([]wgsl.ShaderFile, error) {
	files := make([]wgsl.ShaderFile, len(inputs))
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
				file, err := parseShaderInputCached(input, parser, cache)
				if err != nil {
					errs <- fmt.Errorf("parse %s: %w", input.Path, err)
					continue
				}
				file = enricher.Enrich(file, input, context)
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

// parseShaderInputCached reuses the parser model for identical source content
// and link-affecting configuration. The cache is outside dist so it never
// becomes a deployable artifact, and its key includes the source bytes plus
// every setting that can change ParseWGSLFile's result.
func parseShaderInputCached(input shaderInput, parser shaderFileParser, cache shaderParseCache) (wgsl.ShaderFile, error) {
	content, err := os.ReadFile(input.Path)
	if err != nil {
		return wgsl.ShaderFile{}, fmt.Errorf("read source: %w", err)
	}
	cachePath := cache.path(input, content)
	if file, ok := cache.load(cachePath); ok {
		file.SourceText = string(content)
		return file, nil
	}
	file, err := parser.Parse(&input.Config, wgsl.Source{Path: input.Path, Text: string(content)})
	if err != nil {
		return wgsl.ShaderFile{}, err
	}
	cache.store(cachePath, file)
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

func resolveWgslPathCollisions(files []wgsl.ShaderFile) {
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
func moduleLabels(files []wgsl.ShaderFile) []string {
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

func sourceDirectoryParts(file wgsl.ShaderFile) []string {
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

func appendSearchInfo(searchInfo *[]ShaderSearchableInfo, imports *map[string]string, file wgsl.ShaderFile) {
	link := "/" + strings.TrimPrefix(utils.NormalizeLink(file.Link), "/")
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
