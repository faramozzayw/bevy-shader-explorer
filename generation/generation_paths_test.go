package generation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShaderVersionOptionsLinksMatchingShader(t *testing.T) {
	outputDir := t.TempDir()
	shaderPath := filepath.ToSlash(filepath.Join("bevy_pbr", "0.19.1", "mesh.wgsl.html"))
	matchingPath := filepath.Join(outputDir, "bevy_pbr", "0.20.0", "mesh.wgsl.html")
	if err := os.MkdirAll(filepath.Dir(matchingPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(matchingPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	options := shaderVersionOptions(outputDir, "bevy_pbr", "0.19.1", shaderPath)
	urls := map[string]string{}
	for _, option := range options {
		urls[option["label"]] = option["url"]
	}

	if got := urls["0.19.1"]; got != "bevy_pbr/0.19.1/mesh.wgsl.html" {
		t.Fatalf("current shader URL = %q", got)
	}
	if got := urls["0.20.0"]; got != "bevy_pbr/0.20.0/mesh.wgsl.html" {
		t.Fatalf("matching shader URL = %q", got)
	}
}

func TestShaderVersionOptionsFallsBackToPackage(t *testing.T) {
	outputDir := t.TempDir()
	shaderPath := filepath.ToSlash(filepath.Join("bevy_pbr", "0.19.1", "mesh.wgsl.html"))
	if err := os.MkdirAll(filepath.Join(outputDir, "bevy_pbr", "0.20.0"), 0o755); err != nil {
		t.Fatal(err)
	}

	options := shaderVersionOptions(outputDir, "bevy_pbr", "0.19.1", shaderPath)
	var fallbackURL string
	for _, option := range options {
		if option["label"] == "0.20.0" {
			fallbackURL = option["url"]
		}
	}
	if fallbackURL != "bevy_pbr/0.20.0/index.html" {
		t.Fatalf("fallback package URL = %q", fallbackURL)
	}
	if got := options[0]["url"]; got != "bevy_pbr/0.19.1/mesh.wgsl.html" {
		t.Fatalf("current shader URL = %q", got)
	}
}
