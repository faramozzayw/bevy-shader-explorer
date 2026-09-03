package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWithoutConfig(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FileFilter != "*.wgsl" || !cfg.DependencyTransitive || len(cfg.DependencyInclude) != 2 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadProjectConfig(t *testing.T) {
	root := t.TempDir()
	contents := `project = "crates/demo"
name = "Friendly Aqua"
description = "Custom shader documentation"
output = "docs"
file_filter = "*.shader.wgsl"
exclude = ["vendor", "target"]

[dependencies]
enabled = false
offline = true
include = ["bevy", "bevy_*", "my-shaders"]
transitive = false
`
	if err := os.WriteFile(filepath.Join(root, "wgsl-docs.toml"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourcePath != filepath.Join(root, "crates/demo") || cfg.OutputDir != filepath.Join(root, "docs") {
		t.Fatalf("paths were not resolved relative to config: %+v", cfg)
	}
	if cfg.Name != "Friendly Aqua" || cfg.Description != "Custom shader documentation" {
		t.Fatalf("metadata overrides not loaded: %+v", cfg)
	}
	if cfg.NoDeps != true || !cfg.Offline || cfg.DependencyTransitive || len(cfg.DependencyInclude) != 3 {
		t.Fatalf("dependency settings not loaded: %+v", cfg)
	}
}

func TestLoadUsesCargoPackageMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte(`[package]
name = "bevy-aqua"
version = "0.1.0"
description = "Ocean shaders"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "bevy-aqua" || cfg.Description != "Ocean shaders" {
		t.Fatalf("Cargo metadata not used: %+v", cfg)
	}
}

func TestLoadRejectsUnknownSettings(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "wgsl-docs.toml"), []byte("unknown = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("Load() accepted an unknown setting")
	}
}
