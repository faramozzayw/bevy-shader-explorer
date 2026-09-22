package generation

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	configpkg "github.com/faramozzayw/bevy-shader-explorer/config"
)

//go:embed templates/og-card.svg.tmpl
var ogCardTemplateSource string

var ogCardTemplate = template.Must(template.New("og-card").Parse(ogCardTemplateSource))

type ogDescriptionLine struct {
	DeltaY string
	Text   string
}

type ogCardData struct {
	Title            string
	VersionLine      string
	DescriptionLines []ogDescriptionLine
}

var ogSlugPattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func ogImageURL(cfg configpkg.Config, packageName, version string) string {
	if packageName == "site" {
		return canonicalURL(cfg.SiteURL, filepath.Join("public", "og", ogImageName("site", version)))
	}
	return canonicalURL(cfg.SiteURL, filepath.Join("public", "og", ogImageName(packageName, version)))
}

func ogShaderImageURL(cfg configpkg.Config, packageName, version string) string {
	return ogImageURL(cfg, packageName, version)
}

func ogImageName(packageName, version string) string {
	if packageName == "site" {
		return "site" + ogImageExtension()
	}
	name := ogSlugPattern.ReplaceAllString(packageName+"-"+version, "-")
	return strings.Trim(name, "-") + ogImageExtension()
}

func ogImageExtension() string {
	if _, err := exec.LookPath("rsvg-convert"); err == nil {
		return ".png"
	}
	if _, err := exec.LookPath("resvg"); err == nil {
		return ".png"
	}
	return ".svg"
}

func writeSiteOGImage(cfg configpkg.Config) error {
	const siteTagline = "Explore Bevy's WGSL and WESL shaders with fast, searchable documentation."
	siteImage := filepath.Join(cfg.OutputDir, "public", "og", ogImageName("site", ""))
	if _, err := os.Stat(siteImage); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("check homepage social card: %w", err)
		}
		if err := writeOGCard(cfg.OutputDir, "Shader Explorer", "", "", siteTagline, "site"); err != nil {
			return err
		}
	}
	return nil
}

func writePackageOGImages(cfg configpkg.Config, site documentationSite, progress *generationProgress) error {
	if len(site.Packages) == 0 {
		return nil
	}
	workerCount := 4
	if workerCount > len(site.Packages) {
		workerCount = len(site.Packages)
	}
	jobs := make(chan documentationPackagePage)
	errs := make(chan error, len(site.Packages))
	var workers sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for page := range jobs {
				if err := writeOGCard(cfg.OutputDir, page.PackageName, page.Version, page.BevyVersion, page.Description, page.PackageName); err != nil {
					errs <- fmt.Errorf("render package OG image %s %s: %w", page.PackageName, page.Version, err)
					continue
				}
				if progress != nil {
					progress.addOG()
				}
			}
		}()
	}
	for _, page := range site.Packages {
		jobs <- page
	}
	close(jobs)
	workers.Wait()
	close(errs)
	return firstError(errs)
}

func writeOGCard(outputDir, title, version, bevyVersion, description, packageName string) error {
	base := strings.TrimSuffix(ogImageName(packageName, version), ogImageExtension())
	return writeOGCardAt(outputDir, base, title, version, bevyVersion, description)
}

func writeOGCardAt(outputDir, base, title, version, bevyVersion, description string) error {
	dir := filepath.Join(outputDir, "public", "og")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create social card directory: %w", err)
	}
	svgPath := filepath.Join(dir, base+".svg")
	pngPath := filepath.Join(dir, base+".png")
	title = truncateText(title, 28)
	descriptionLines := wrapText(description, 58, 3)
	versionLine := "WGSL / WESL documentation"
	if version != "" {
		versionLine = "version " + version
		if bevyVersion != "" {
			versionLine += "  ·  Bevy " + bevyVersion
		}
	}
	if len(descriptionLines) == 0 {
		descriptionLines = []string{"Searchable shader documentation for the Bevy ecosystem."}
	}
	data := ogCardData{
		Title:            title,
		VersionLine:      versionLine,
		DescriptionLines: make([]ogDescriptionLine, len(descriptionLines)),
	}
	for i, line := range descriptionLines {
		deltaY := "36"
		if i == 0 {
			deltaY = "0"
		}
		data.DescriptionLines[i] = ogDescriptionLine{DeltaY: deltaY, Text: line}
	}
	var svgBuffer bytes.Buffer
	if err := ogCardTemplate.Execute(&svgBuffer, data); err != nil {
		return fmt.Errorf("render social card template: %w", err)
	}
	svg := svgBuffer.Bytes()
	if err := os.WriteFile(svgPath, svg, 0o644); err != nil {
		return fmt.Errorf("write social card SVG: %w", err)
	}
	if _, err := exec.LookPath("rsvg-convert"); err != nil {
		if _, err := exec.LookPath("resvg"); err != nil {
			_ = os.Remove(pngPath)
			return nil
		}
	}
	renderSVGPath := svgPath
	if mascot, err := os.ReadFile(filepath.Join(outputDir, "public", "mascot2.jpeg")); err == nil {
		renderSVGPath = svgPath + ".render.svg"
		data := base64.StdEncoding.EncodeToString(mascot)
		renderSVG := strings.Replace(string(svg), "../mascot2.jpeg", "data:image/jpeg;base64,"+data, 1)
		if err := os.WriteFile(renderSVGPath, []byte(renderSVG), 0o644); err != nil {
			return fmt.Errorf("write renderable social card SVG: %w", err)
		}
		defer os.Remove(renderSVGPath)
	}
	if renderer, err := exec.LookPath("rsvg-convert"); err == nil {
		if err := exec.Command(renderer, renderSVGPath, "-o", pngPath).Run(); err != nil {
			return fmt.Errorf("render social card PNG: %w", err)
		}
	} else if renderer, err := exec.LookPath("resvg"); err == nil {
		if err := exec.Command(renderer, renderSVGPath, pngPath).Run(); err != nil {
			return fmt.Errorf("render social card PNG: %w", err)
		}
	}
	if err := os.Remove(svgPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove intermediate social card SVG: %w", err)
	}
	return nil
}

func truncateText(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func wrapText(value string, maxChars, maxLines int) []string {
	words := strings.Fields(value)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, maxLines)
	for len(words) > 0 && len(lines) < maxLines {
		line := words[0]
		words = words[1:]
		for len(words) > 0 && len(line)+1+len(words[0]) <= maxChars {
			line += " " + words[0]
			words = words[1:]
		}
		lines = append(lines, line)
	}
	if len(words) > 0 {
		last := lines[len(lines)-1]
		lines[len(lines)-1] = truncateText(last, maxChars-3) + "..."
	}
	return lines
}
