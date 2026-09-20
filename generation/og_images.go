package generation

import (
	"encoding/base64"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	configpkg "main/config"
)

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

func writePackageOGImages(cfg configpkg.Config, site documentationSite) error {
	for _, page := range site.Packages {
		if err := writeOGCard(cfg.OutputDir, page.PackageName, page.Version, page.BevyVersion, page.Description, page.PackageName); err != nil {
			return err
		}
	}
	return nil
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
	descriptionSVG := make([]string, len(descriptionLines))
	for i, line := range descriptionLines {
		descriptionSVG[i] = fmt.Sprintf(`<tspan x="80" dy="%s">%s</tspan>`, func() string {
			if i == 0 {
				return "0"
			}
			return "36"
		}(), html.EscapeString(line))
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
<defs><linearGradient id="shade" x1="0" y1="0" x2="1" y2="0"><stop stop-color="#07111c" stop-opacity=".94"/><stop offset=".62" stop-color="#07111c" stop-opacity=".72"/><stop offset="1" stop-color="#07111c" stop-opacity=".08"/></linearGradient></defs>
<image href="../mascot2.jpeg" x="0" y="0" width="1200" height="630" preserveAspectRatio="xMidYMid slice"/><rect width="1200" height="630" fill="url(#shade)"/>
<text x="78" y="105" fill="#9edff2" font-family="monospace" font-size="27" font-weight="bold">SHADER EXPLORER</text>
<text x="78" y="260" fill="#ffffff" font-family="monospace" font-size="58" font-weight="bold">%s</text>
<text x="80" y="322" fill="#c4eaf5" font-family="monospace" font-size="28">%s</text>
<text x="80" y="420" fill="#f1f7f9" font-family="sans-serif" font-size="27">%s</text>
</svg>
`, html.EscapeString(title), html.EscapeString(versionLine), strings.Join(descriptionSVG, ""))
	if err := os.WriteFile(svgPath, []byte(svg), 0o644); err != nil {
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
		renderSVG := strings.Replace(svg, "../mascot2.jpeg", "data:image/jpeg;base64,"+data, 1)
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
