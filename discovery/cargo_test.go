package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCargoMetadataUsesCargoResolver(t *testing.T) {
	original := cargoMetadataCommand
	t.Cleanup(func() { cargoMetadataCommand = original })
	var gotManifest string
	var gotOffline bool
	cargoMetadataCommand = func(_ context.Context, manifest string, offline bool) ([]byte, error) {
		gotManifest, gotOffline = manifest, offline
		return []byte(`{"packages":[{"id":"bevy","name":"bevy","version":"0.19.0","manifest_path":"/tmp/bevy/Cargo.toml"}],"workspace_root":"/tmp","resolve":{"nodes":[]}}`), nil
	}

	metadata, err := ReadCargoMetadata(context.Background(), "/tmp/project", true)
	if err != nil {
		t.Fatalf("ReadCargoMetadata() error = %v", err)
	}
	if gotManifest != filepath.Join("/tmp/project", "Cargo.toml") || !gotOffline {
		t.Fatalf("cargo invocation = (%q, offline=%v)", gotManifest, gotOffline)
	}
	if len(metadata.Packages) != 1 || metadata.Packages[0].Name != "bevy" {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
}

func TestReadCargoMetadataReportsResolverOutput(t *testing.T) {
	original := cargoMetadataCommand
	t.Cleanup(func() { cargoMetadataCommand = original })
	cargoMetadataCommand = func(context.Context, string, bool) ([]byte, error) {
		return []byte("failed to fetch crates.io index"), errors.New("exit status 101")
	}
	_, err := ReadCargoMetadata(context.Background(), "/tmp/project", false)
	if err == nil || !strings.Contains(err.Error(), "failed to fetch crates.io index") {
		t.Fatalf("expected Cargo output in error, got %v", err)
	}
}

func TestFilterCargoPackagesFollowsTransitiveDependencies(t *testing.T) {
	metadata := CargoMetadata{
		Packages: []CargoPackage{
			{ID: "app", Name: "my-app", Version: "1.0.0"},
			{ID: "bevy", Name: "bevy", Version: "0.16.0"},
			{ID: "render", Name: "bevy_render", Version: "0.16.0"},
			{ID: "log", Name: "tracing", Version: "0.1.0"},
		},
		Resolve: &CargoResolve{Nodes: []CargoNode{
			{ID: "bevy", Deps: []CargoDependency{{PackageID: "render"}}},
			{ID: "render", Deps: []CargoDependency{{PackageID: "log"}}},
		}},
	}

	packages := FilterCargoPackages(metadata, []string{"bevy", "bevy_*"}, true)
	if len(packages) != 3 {
		t.Fatalf("selected %d packages, want 3", len(packages))
	}
	if packages[0].Name != "bevy" || packages[1].Name != "bevy_render" || packages[2].Name != "tracing" {
		t.Fatalf("unexpected package order: %#v", packages)
	}
}

func TestDiscoverDependencyShadersScansSelectedRootsOnly(t *testing.T) {
	root := t.TempDir()
	bevyRoot := filepath.Join(root, "bevy")
	otherRoot := filepath.Join(root, "other")
	if err := os.MkdirAll(filepath.Join(bevyRoot, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		filepath.Join(bevyRoot, "Cargo.toml"),
		filepath.Join(bevyRoot, "shader.wgsl"),
		filepath.Join(bevyRoot, "target", "generated.wgsl"),
		filepath.Join(otherRoot, "other.wgsl"),
	} {
		if err := os.WriteFile(file, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	shaders, err := DiscoverDependencyShaders([]CargoPackage{{Name: "bevy_render", Version: "0.16.0", ManifestPath: filepath.Join(bevyRoot, "Cargo.toml")}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(shaders) != 1 || shaders[0].Path != filepath.Join(bevyRoot, "shader.wgsl") {
		t.Fatalf("unexpected shaders: %#v", shaders)
	}
	if shaders[0].Package != "bevy_render" || !shaders[0].Dependency {
		t.Fatalf("unexpected source ownership: %#v", shaders[0])
	}
}

func TestFilterCargoPackagesCanDisableTransitiveDependencies(t *testing.T) {
	metadata := CargoMetadata{
		Packages: []CargoPackage{
			{ID: "bevy", Name: "bevy", Version: "0.16.0"},
			{ID: "render", Name: "bevy_render", Version: "0.16.0"},
		},
		Resolve: &CargoResolve{Nodes: []CargoNode{{ID: "bevy", Deps: []CargoDependency{{PackageID: "render"}}}}},
	}

	packages := FilterCargoPackages(metadata, []string{"bevy"}, false)
	if len(packages) != 1 || packages[0].Name != "bevy" {
		t.Fatalf("unexpected non-transitive selection: %#v", packages)
	}
}
