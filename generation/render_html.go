package generation

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	config "github.com/faramozzayw/bevy-shader-explorer/config"

	"github.com/aymerick/raymond"
	"github.com/faramozzayw/bevy-shader-explorer/shader"
	"github.com/faramozzayw/bevy-shader-explorer/utils"
)

type documentRenderer interface {
	Render(context GenerationContext, site documentationSite, registry []packageRegistryEntry, progress *generationProgress) error
}

type htmlDocumentRenderer struct {
	search searchIndexRenderer
}

func (renderer htmlDocumentRenderer) Render(context GenerationContext, site documentationSite, registry []packageRegistryEntry, progress *generationProgress) error {
	return renderHTMLDocumentation(context.Config, site, registry, progress, renderer.search)
}

type jsonDocumentRenderer struct{}

func (jsonDocumentRenderer) Render(context GenerationContext, site documentationSite, registry []packageRegistryEntry, progress *generationProgress) error {
	return writeDocumentationJSON(context.OutputDir, site)
}

func renderHTMLDocumentation(config config.Config, site documentationSite, registry []packageRegistryEntry, progress *generationProgress, search searchIndexRenderer) error {
	if !config.SkipCatalogue {
		if err := search.WritePublic(config.OutputDir, site.Search, site.Packages); err != nil {
			return err
		}
		if err := copyStaticAssets(filepath.Join(config.OutputDir, "public")); err != nil {
			return err
		}
	} else if err := copyOGMascot(filepath.Join(config.OutputDir, "public")); err != nil {
		return err
	} else if err := search.WritePending(config.OutputDir, site.Search, site.Packages); err != nil {
		return err
	}
	if !config.SkipCatalogue {
		if err := writeSiteOGImage(config); err != nil {
			return err
		}
	}

	compiledTemplate, err := raymond.Parse(WGSL_DOC_TEMPLATE_SOURCE)
	if err != nil {
		return fmt.Errorf("parse shader template: %w", err)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	pageErrors := make(chan error, len(site.Shaders))

	versionedOutput := config.OutputDir

	for _, shaderFile := range site.Shaders {
		shaderFile := shaderFile
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// A combined build may encounter the same package first as a
			// dependency and later as a canonical matrix source. Preserve an
			// already-rendered page during dependency passes so those later
			// canonical pages cannot be overwritten by a less precise source ref.
			if shaderFile.Dependency {
				pagePath := filepath.Join(versionedOutput, shaderFile.WgslPath)
				if _, err := os.Stat(pagePath); err == nil {
					progress.addDocumentation()
					return
				}
			}
			if err := renderShaderPage(compiledTemplate, shaderFile, versionedOutput); err != nil {
				pageErrors <- fmt.Errorf("render %s: %w", shaderFile.SourcePath, err)
			}
			progress.addDocumentation()
		}()
	}

	wg.Wait()
	close(pageErrors)
	if err := firstError(pageErrors); err != nil {
		return err
	}

	renderedPackages := make([]documentationPackagePage, 0, len(site.Packages))
	for _, page := range site.Packages {
		pagePath := filepath.Join(versionedOutput, page.DetailPath)
		if page.Dependency {
			if _, err := os.Stat(pagePath); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Join(versionedOutput, filepath.Dir(page.DetailPath)), os.ModePerm); err != nil {
			return fmt.Errorf("create package output directory: %w", err)
		}
		moduleGroups := groupShaderModules(page.AllFiles)
		if err := renderTemplateToFile(PACKAGE_DOC_TEMPLATE_SOURCE, packageTemplateModel{
			Name: page.PackageName, Files: page.AllFiles, ModuleGroups: moduleGroups,
			ShowModuleGroups: len(moduleGroups) > 1, Count: page.Count, Description: page.Description,
			SeoDescription: packageSEODescription(page.PackageName, page.Version, page.Description, page.Count),
			Dependencies:   page.Dependencies, HasDependencies: len(page.Dependencies) > 0, DependencyCount: len(page.Dependencies),
			DirectDependencies: page.DirectDependencies, TransitiveDependencies: page.TransitiveDependencies,
			HasDirectDependencies: len(page.DirectDependencies) > 0, HasTransitiveDependencies: len(page.TransitiveDependencies) > 0,
			DirectDependencyCount: len(page.DirectDependencies), TransitiveDependencyCount: len(page.TransitiveDependencies),
			Authors: page.Metadata.Authors, License: page.Metadata.License, Repository: page.Metadata.Repository, Homepage: page.Metadata.Homepage,
			PackageVersion: page.Version, SearchIndex: packageSearchIndex(page.PackageName, page.Version),
			SeoTitle: fmt.Sprintf("%s %s — Bevy Shader Explorer", page.PackageName, page.Version), Version: config.Version,
			ProjectVersion: page.Version, ProjectCount: page.Count, URLPrefix: joinDocURL("project", ""),
			PackageURLPrefix: joinDocURL("project", filepath.Join(page.PackageName, page.Version)), VersionOptions: page.VersionOptions,
			CanonicalURL: canonicalURL(config.SiteURL, page.DetailPath), OgImageURL: ogImageURL(config, page.PackageName, page.Version), IssueURL: config.IssueURL,
			StructuredData: jsonValue(map[string]interface{}{
				"@context": "https://schema.org", "@type": "SoftwareSourceCode",
				"name": page.PackageName, "version": page.Version, "description": page.Description,
				"url": canonicalURL(config.SiteURL, page.DetailPath), "codeRepository": page.Metadata.Repository,
				"isPartOf": map[string]interface{}{"@type": "WebSite", "name": "Bevy Shader Explorer", "url": canonicalURL(config.SiteURL, "")},
			}),
		}, pagePath); err != nil {
			return fmt.Errorf("render package %s %s: %w", page.PackageName, page.Version, err)
		}
		renderedPackages = append(renderedPackages, page)
	}
	if !config.SkipCatalogue {
		if err := writePackageOGImages(config, documentationSite{Packages: renderedPackages}, progress); err != nil {
			return err
		}
	}
	if config.SkipCatalogue {
		return nil
	}

	if err := renderTemplateToFile(HOME_DOC_TEMPLATE_SOURCE, newHomeTemplateModel(config, site.HomeSections, len(registry), site.TotalShaderCount, site.ProjectCount, site.DependencyCount, config.ProjectVersion), filepath.Join(versionedOutput, "index.html")); err != nil {
		return fmt.Errorf("render home page: %w", err)
	}

	if err := renderTemplateToFile(NOT_FOUND_TEMPLATE_SOURCE, notFoundTemplateModel{},
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

func renderShaderPage(compiledTemplate *raymond.Template, shaderFile shader.ShaderFile, outputDir string) error {
	fileOutputPath := strings.ReplaceAll(filepath.Join(outputDir, shaderFile.WgslPath), "src/", "")
	html, err := compiledTemplate.Exec(shaderFile)
	if err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(outputDir, shaderFile.WgslPath)), os.ModePerm); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := utils.WriteFileIfChanged(fileOutputPath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
