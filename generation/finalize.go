package generation

import (
	"fmt"
	"os"
	"path/filepath"

	"main/config"
)

// Finalize writes catalogue-wide artifacts once, after all release workers have
// rendered their package and shader pages.
func Finalize(config config.Config) error {
	SetupHandlebars()
	registry, err := readPackageRegistry(config.OutputDir)
	if err != nil {
		return err
	}
	pending, err := readPendingPackageRegistry(config.OutputDir)
	if err != nil {
		return err
	}
	registry = mergePackageRegistry(registry, pending)
	if len(registry) == 0 {
		return fmt.Errorf("no generated packages found in %s", config.OutputDir)
	}
	if err := writePackageRegistry(config.OutputDir, registry); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(config.OutputDir, "public", ".package-registry")); err != nil {
		return fmt.Errorf("remove pending package registry: %w", err)
	}
	if err := copyStaticAssets(filepath.Join(config.OutputDir, "public")); err != nil {
		return err
	}
	if err := finalizeSearchIndexes(config.OutputDir); err != nil {
		return err
	}
	packages := make([]documentationPackagePage, 0, len(registry))
	for _, entry := range registry {
		packages = append(packages, documentationPackagePage{
			PackageName: entry.PackageName,
			Version:     entry.Version,
			BevyVersion: entry.BevyVersion,
			Description: entry.Description,
			Count:       entry.Count,
			DetailPath:  entry.DetailPath,
		})
	}
	site := documentationSite{
		Version:          config.Version,
		HomeSections:     packageRegistrySections(registry),
		Packages:         packages,
		TotalShaderCount: packageRegistryShaderCount(registry),
	}
	progress := newGenerationProgress(config.Name, config.Version, progressOptions{Finalizing: true, OG: true})
	defer progress.finish()
	progress.setOGTotal(len(site.Packages))
	if err := writePackageOGImages(config, site, progress); err != nil {
		return err
	}
	if err := writeSiteOGImage(config); err != nil {
		return err
	}
	if err := renderTemplateToFile(HOME_DOC_TEMPLATE_SOURCE, map[string]interface{}{
		"sections":         site.HomeSections,
		"packageCount":     len(registry),
		"totalShaderCount": site.TotalShaderCount,
		"projectCount":     0,
		"dependencyCount":  0,
		"name":             config.Name,
		"description":      config.Description,
		"version":          config.Version,
		"projectVersion":   config.ProjectVersion,
		"urlPrefix":        joinDocURL("project", ""),
		"canonicalURL":     canonicalURL(config.SiteURL, ""),
		"ogImageURL":       ogImageURL(config, "site", config.ProjectVersion),
		"issueURL":         config.IssueURL,
		"structuredData": jsonValue(map[string]interface{}{
			"@context": "https://schema.org", "@type": "WebSite", "name": "Shader Explorer",
			"description": config.Description, "url": canonicalURL(config.SiteURL, ""),
		}),
	}, filepath.Join(config.OutputDir, "index.html")); err != nil {
		return fmt.Errorf("render final home page: %w", err)
	}
	if err := renderTemplateToFile(NOT_FOUND_TEMPLATE_SOURCE, map[string]interface{}{}, filepath.Join(config.OutputDir, "404.html")); err != nil {
		return fmt.Errorf("render final not-found page: %w", err)
	}
	if err := writePackageVersionsManifest(config.OutputDir); err != nil {
		return err
	}
	if err := writeSEOFiles(config.OutputDir, config.SiteURL); err != nil {
		return err
	}
	progress.completeFinalizing()
	return nil
}
