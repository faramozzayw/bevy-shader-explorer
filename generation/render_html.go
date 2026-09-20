package generation

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	config "main/config"

	"github.com/aymerick/raymond"
	progressbar "github.com/schollz/progressbar/v3"
)

func renderDocumentation(config config.Config, site documentationSite, registry []packageRegistryEntry) error {
	if config.Format == "json" {
		return writeDocumentationJSON(config.OutputDir, site)
	}
	if err := copyItemsToPublic(&config, site.Search); err != nil {
		return err
	}
	if err := writeOGImages(config, site); err != nil {
		return err
	}

	compiledTemplate, err := raymond.Parse(WGSL_DOC_TEMPLATE_SOURCE)
	if err != nil {
		return fmt.Errorf("parse shader template: %w", err)
	}

	processingBar := progressbar.Default(int64(len(site.Shaders)), "🛠️ Generating Documentation")

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	pageErrors := make(chan error, len(site.Shaders))

	versionedOutput := config.OutputDir

	for _, wgslFile := range site.Shaders {
		wgslFile := wgslFile
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// A combined build may encounter the same package first as a
			// dependency and later as a canonical matrix source. Preserve an
			// already-rendered page during dependency passes so those later
			// canonical pages cannot be overwritten by a less precise source ref.
			if wgslFile.Dependency {
				pagePath := filepath.Join(versionedOutput, wgslFile.WgslPath)
				if _, err := os.Stat(pagePath); err == nil {
					processingBar.Add(1)
					return
				}
			}
			if err := wgslFile.GenerateWgslPage(compiledTemplate, versionedOutput); err != nil {
				pageErrors <- fmt.Errorf("render %s: %w", wgslFile.SourcePath, err)
			}
			processingBar.Add(1)
		}()
	}

	wg.Wait()
	close(pageErrors)
	if err := firstError(pageErrors); err != nil {
		return err
	}

	for _, page := range site.Packages {
		if err := os.MkdirAll(filepath.Join(versionedOutput, filepath.Dir(page.DetailPath)), os.ModePerm); err != nil {
			return fmt.Errorf("create package output directory: %w", err)
		}
		if err := renderTemplateToFile(PACKAGE_DOC_TEMPLATE_SOURCE, map[string]interface{}{
			"name":                      page.PackageName,
			"files":                     page.AllFiles,
			"count":                     page.Count,
			"description":               page.Description,
			"dependencies":              page.Dependencies,
			"hasDependencies":           len(page.Dependencies) > 0,
			"dependencyCount":           len(page.Dependencies),
			"directDependencies":        page.DirectDependencies,
			"transitiveDependencies":    page.TransitiveDependencies,
			"hasDirectDependencies":     len(page.DirectDependencies) > 0,
			"hasTransitiveDependencies": len(page.TransitiveDependencies) > 0,
			"directDependencyCount":     len(page.DirectDependencies),
			"transitiveDependencyCount": len(page.TransitiveDependencies),
			"authors":                   page.Metadata.Authors, "license": page.Metadata.License,
			"repository": page.Metadata.Repository, "homepage": page.Metadata.Homepage,
			"packageVersion":   page.Version,
			"seoTitle":         fmt.Sprintf("%s %s — Shader Explorer", page.PackageName, page.Version),
			"version":          config.Version,
			"projectVersion":   page.Version,
			"projectCount":     page.Count,
			"urlPrefix":        joinDocURL("project", ""),
			"packageURLPrefix": joinDocURL("project", filepath.Join(page.PackageName, page.Version)),
			"versionOptions":   page.VersionOptions,
			"canonicalURL":     canonicalURL(config.SiteURL, page.DetailPath),
			"ogImageURL":       ogImageURL(config, page.PackageName, page.Version),
			"structuredData": jsonValue(map[string]interface{}{
				"@context": "https://schema.org", "@type": "SoftwareSourceCode",
				"name": page.PackageName, "version": page.Version, "description": page.Description,
				"url": canonicalURL(config.SiteURL, page.DetailPath), "codeRepository": page.Metadata.Repository,
			}),
		}, filepath.Join(versionedOutput, page.DetailPath)); err != nil {
			return fmt.Errorf("render package %s %s: %w", page.PackageName, page.Version, err)
		}
	}

	if err := renderTemplateToFile(HOME_DOC_TEMPLATE_SOURCE, map[string]interface{}{
		"sections":         site.HomeSections,
		"packageCount":     len(registry),
		"totalShaderCount": site.TotalShaderCount,
		"projectCount":     site.ProjectCount,
		"dependencyCount":  site.DependencyCount,
		"name":             config.Name,
		"description":      config.Description,
		"skipHomeButton":   true,
		"version":          config.Version,
		"projectVersion":   config.ProjectVersion,
		"urlPrefix":        joinDocURL("project", ""),
		"canonicalURL":     canonicalURL(config.SiteURL, ""),
		"ogImageURL":       ogImageURL(config, "site", config.ProjectVersion),
		"structuredData": jsonValue(map[string]interface{}{
			"@context": "https://schema.org", "@type": "WebSite", "name": "Shader Explorer",
			"description": config.Description, "url": canonicalURL(config.SiteURL, ""),
		}),
	}, filepath.Join(versionedOutput, "index.html")); err != nil {
		return fmt.Errorf("render home page: %w", err)
	}

	if err := renderTemplateToFile(NOT_FOUND_TEMPLATE_SOURCE, map[string]interface{}{},
		filepath.Join(config.OutputDir, "404.html")); err != nil {
		return fmt.Errorf("render not-found page: %w", err)
	}
	if err := writePackageVersionsManifest(config.OutputDir); err != nil {
		return err
	}

	if err := writeSEOFiles(config.OutputDir, config.SiteURL); err != nil {
		return err
	}
	return nil
}
