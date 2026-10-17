package shader

import (
	"testing"

	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
)

func TestResolveTypeLinksBuildsResolvedImports(t *testing.T) {
	file := ShaderFile{
		File: document.File{
			DeclaredImports: document.DeclaredImports{
				"view": {"bevy_pbr::mesh_view_bindings::view"},
			},
		},
	}
	file.ResolveTypeLinks(map[string]string{
		"bevy_pbr::mesh_view_bindings": "/bevy_pbr/0.20.0/mesh_view_bindings.html",
	})

	if len(file.ResolvedImports) != 1 {
		t.Fatalf("expected 1 resolved import, got %#v", file.ResolvedImports)
	}
	ref := file.ResolvedImports[0]
	if ref.Name != "view" || ref.Path != "bevy_pbr::mesh_view_bindings::view" {
		t.Fatalf("unexpected import ref: %#v", ref)
	}
	if ref.Link != "/bevy_pbr/0.20.0/mesh_view_bindings.html#view" {
		t.Fatalf("expected resolved link with anchor, got %q", ref.Link)
	}
}

func TestResolveTypeLinksLeavesUnresolvedImportLinkEmpty(t *testing.T) {
	file := ShaderFile{
		File: document.File{
			DeclaredImports: document.DeclaredImports{
				"unknown_symbol": {"some_crate::unknown_module::unknown_symbol"},
			},
		},
	}
	file.ResolveTypeLinks(map[string]string{"bevy_pbr": "/bevy_pbr/0.20.0/index.html"})

	if len(file.ResolvedImports) != 1 {
		t.Fatalf("expected 1 resolved import, got %#v", file.ResolvedImports)
	}
	if file.ResolvedImports[0].Link != "" {
		t.Fatalf("expected no link for an unresolvable import, got %q", file.ResolvedImports[0].Link)
	}
}

func TestAttachUsageSkipsFilesWithoutImportPath(t *testing.T) {
	file := ShaderFile{File: document.File{Consts: []document.Const{{Name: "SCALE"}}}}
	file.AttachUsage(map[string][]document.UsageRef{"anything::SCALE": {{Filename: "x.wgsl"}}})

	if file.Consts[0].UsedBy != nil {
		t.Fatalf("expected no usage attached without an import path, got %#v", file.Consts[0].UsedBy)
	}
}

func TestAttachUsageAttachesMatchingSymbols(t *testing.T) {
	importPath := "bevy_pbr::mesh_view_bindings"
	file := ShaderFile{
		File: document.File{
			ImportPath: &importPath,
			Bindings:   []document.Binding{{Name: "view"}},
			Functions:  []document.Function{{Name: "unused_fn"}},
		},
	}
	usedBy := map[string][]document.UsageRef{
		"bevy_pbr::mesh_view_bindings::view": {
			{Link: "/a.html", PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "a.wgsl"},
		},
	}
	file.AttachUsage(usedBy)

	if file.Bindings[0].UsedBySummary != "Used by 1 shader in bevy_pbr 0.20.0" {
		t.Fatalf("unexpected summary: %q", file.Bindings[0].UsedBySummary)
	}
	if len(file.Bindings[0].UsedBy) != 1 || file.Bindings[0].UsedBy[0].Filename != "a.wgsl" {
		t.Fatalf("unexpected used-by refs: %#v", file.Bindings[0].UsedBy)
	}
	if file.Functions[0].UsedBy != nil {
		t.Fatalf("expected no usage for an unimported function, got %#v", file.Functions[0].UsedBy)
	}
}

func TestAttachUsageCountsWholeModuleImports(t *testing.T) {
	importPath := "bevy_pbr::environment_map"
	file := ShaderFile{
		File: document.File{
			ImportPath: &importPath,
			Functions:  []document.Function{{Name: "environment_map_light"}},
		},
	}
	usedBy := map[string][]document.UsageRef{
		"bevy_pbr::environment_map": {
			{Link: "/pbr_functions.html", PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "pbr_functions.wgsl"},
		},
	}
	file.AttachUsage(usedBy)

	if file.Functions[0].UsedBySummary != "Used by 1 shader in bevy_pbr 0.20.0" {
		t.Fatalf("unexpected summary: %q", file.Functions[0].UsedBySummary)
	}
	if len(file.Functions[0].UsedBy) != 1 || file.Functions[0].UsedBy[0].Filename != "pbr_functions.wgsl" {
		t.Fatalf("expected the module-level import to count as usage, got %#v", file.Functions[0].UsedBy)
	}
}

func TestAttachUsageDedupesSymbolAndModuleImportOfTheSameFile(t *testing.T) {
	importPath := "bevy_pbr::mesh_view_bindings"
	file := ShaderFile{
		File: document.File{
			ImportPath: &importPath,
			Bindings:   []document.Binding{{Name: "view"}},
		},
	}
	usedBy := map[string][]document.UsageRef{
		"bevy_pbr::mesh_view_bindings::view": {{Link: "/a.html", PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "a.wgsl"}},
		"bevy_pbr::mesh_view_bindings":       {{Link: "/a.html", PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "a.wgsl"}},
	}
	file.AttachUsage(usedBy)

	if len(file.Bindings[0].UsedBy) != 1 {
		t.Fatalf("expected the duplicate reference from the same file to be deduped, got %#v", file.Bindings[0].UsedBy)
	}
}
