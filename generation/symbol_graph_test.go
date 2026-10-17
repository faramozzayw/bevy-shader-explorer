package generation

import (
	"testing"

	wgsl "github.com/faramozzayw/bevy-shader-explorer/shader"
	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
)

func TestAppendUsageEdgesRecordsOneEdgePerImportingFile(t *testing.T) {
	usedBy := make(map[string][]document.UsageRef)
	file := wgsl.ShaderFile{
		File: document.File{
			DeclaredImports: document.DeclaredImports{
				"view":    {"bevy_pbr::mesh_view_bindings::view"},
				"globals": {"bevy_pbr::mesh_view_bindings::globals"},
			},
			Filename: "particle.wgsl",
		},
		ProjectName:    "bevy_hanabi",
		ProjectVersion: "0.19.0",
		Link:           "bevy_hanabi/0.19.0/particle.html",
	}

	appendUsageEdges(usedBy, file)

	if len(usedBy["bevy_pbr::mesh_view_bindings::view"]) != 1 {
		t.Fatalf("expected one edge for view, got %#v", usedBy["bevy_pbr::mesh_view_bindings::view"])
	}
	ref := usedBy["bevy_pbr::mesh_view_bindings::view"][0]
	if ref.PackageName != "bevy_hanabi" || ref.PackageVersion != "0.19.0" || ref.Filename != "particle.wgsl" {
		t.Fatalf("unexpected usage ref: %#v", ref)
	}
	if ref.Link != "/bevy_hanabi/0.19.0/particle.html" {
		t.Fatalf("expected a leading slash on the link, got %q", ref.Link)
	}
	if len(usedBy["bevy_pbr::mesh_view_bindings::globals"]) != 1 {
		t.Fatalf("expected one edge for globals, got %#v", usedBy["bevy_pbr::mesh_view_bindings::globals"])
	}
}

func TestAppendUsageEdgesDedupesSameSymbolImportedTwice(t *testing.T) {
	usedBy := make(map[string][]document.UsageRef)
	file := wgsl.ShaderFile{
		File: document.File{
			DeclaredImports: document.DeclaredImports{
				"view":      {"bevy_pbr::mesh_view_bindings::view"},
				"view_also": {"bevy_pbr::mesh_view_bindings::view"},
			},
			Filename: "particle.wgsl",
		},
		ProjectName: "bevy_hanabi", ProjectVersion: "0.19.0", Link: "bevy_hanabi/0.19.0/particle.html",
	}

	appendUsageEdges(usedBy, file)

	if got := len(usedBy["bevy_pbr::mesh_view_bindings::view"]); got != 1 {
		t.Fatalf("expected a single deduped edge, got %d", got)
	}
}

func TestAppendUsageEdgesSkipsFilesWithNoImports(t *testing.T) {
	usedBy := make(map[string][]document.UsageRef)
	appendUsageEdges(usedBy, wgsl.ShaderFile{})

	if len(usedBy) != 0 {
		t.Fatalf("expected no edges, got %#v", usedBy)
	}
}
