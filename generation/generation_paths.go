package generation

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func canonicalURL(siteURL, docPath string) string {
	docPath = "/" + strings.TrimLeft(filepath.ToSlash(docPath), "/")
	if siteURL == "" {
		return docPath
	}
	return strings.TrimRight(siteURL, "/") + docPath
}

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

func writeSEOFiles(outputDir, siteURL string) error {
	robots := "User-agent: *\nAllow: /\n"
	if siteURL != "" {
		robots += "Sitemap: " + strings.TrimRight(siteURL, "/") + "/sitemap.xml\n"
	}
	if err := os.WriteFile(filepath.Join(outputDir, "robots.txt"), []byte(robots), 0o644); err != nil {
		return fmt.Errorf("write robots.txt: %w", err)
	}
	if siteURL == "" {
		return nil
	}

	var urls []string
	err := filepath.WalkDir(outputDir, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filePath != outputDir && filepath.Base(filePath) == "public" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(filePath) != ".html" || filepath.Base(filePath) == "404.html" {
			return nil
		}
		relative, err := filepath.Rel(outputDir, filePath)
		if err != nil {
			return err
		}
		urls = append(urls, canonicalURL(siteURL, relative))
		return nil
	})
	if err != nil {
		return fmt.Errorf("collect sitemap URLs: %w", err)
	}
	// Deterministic output makes generated artifacts easy to review.
	slices.Sort(urls)
	var sitemap strings.Builder
	sitemap.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, pageURL := range urls {
		sitemap.WriteString("  <url><loc>")
		sitemap.WriteString(xmlEscape(pageURL))
		sitemap.WriteString("</loc></url>\n")
	}
	sitemap.WriteString("</urlset>\n")
	if err := os.WriteFile(filepath.Join(outputDir, "sitemap.xml"), []byte(sitemap.String()), 0o644); err != nil {
		return fmt.Errorf("write sitemap.xml: %w", err)
	}
	return nil
}

func xmlEscape(value string) string {
	var escaped strings.Builder
	_ = xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}
