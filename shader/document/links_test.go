package document

import (
	"testing"

	"github.com/faramozzayw/bevy-shader-explorer/utils"
)

func TestResolveTypeLinkForLocalStructure(t *testing.T) {
	typeInfo := TypeInfo{Type: "Settings"}
	typeInfo.ResolveTypeLink(nil, []string{"Settings"})

	if typeInfo.TypeLink != "#Settings" {
		t.Fatalf("TypeLink = %q, want %q", typeInfo.TypeLink, "#Settings")
	}
}

func TestResolveTypeLinkForImportedStructure(t *testing.T) {
	typeInfo := TypeInfo{Type: "Settings", FullTypePath: "pbr::Settings"}
	typeInfo.ResolveTypeLink(map[string]string{"pbr": "/0.15.3/pbr.html"}, nil)

	if typeInfo.TypeLink != "/0.15.3/pbr.html#Settings" || !typeInfo.TypeLinkBlank {
		t.Fatalf("unexpected imported link: %#v", typeInfo)
	}
}

func TestResolveTypeLinkDefaultsMissingFullPath(t *testing.T) {
	typeInfo := TypeInfo{Type: "Settings"}
	typeInfo.ResolveTypeLink(nil, nil)

	if typeInfo.FullTypePath != "Settings" {
		t.Fatalf("FullTypePath = %q, want %q", typeInfo.FullTypePath, "Settings")
	}
}

func TestResolveTypeLinksNestedGenericParts(t *testing.T) {
	utils.LoadWgslTypes()
	typeInfo := TypeInfo{Type: "array<Settings>", FullTypePath: "array<Settings>"}
	typeInfo.ResolveTypeLink(nil, []string{"Settings"})

	if len(typeInfo.TypeParts) != 4 {
		t.Fatalf("got %d type parts, want 4: %#v", len(typeInfo.TypeParts), typeInfo.TypeParts)
	}
	if typeInfo.TypeParts[0].TypeLink == "" || typeInfo.TypeParts[0].Text != "array" {
		t.Fatalf("outer generic type was not linked: %#v", typeInfo.TypeParts[0])
	}
	if typeInfo.TypeParts[2].TypeLink != "#Settings" || typeInfo.TypeParts[2].Text != "Settings" {
		t.Fatalf("inner type was not linked: %#v", typeInfo.TypeParts[2])
	}
}

func TestResolveTypeLinksNestedBuiltins(t *testing.T) {
	utils.LoadWgslTypes()
	typeInfo := TypeInfo{Type: "vec4<f32>", FullTypePath: "vec4<f32>"}
	typeInfo.ResolveTypeLink(nil, nil)

	if typeInfo.TypeParts[0].TypeLink == "" || typeInfo.TypeParts[2].TypeLink == "" {
		t.Fatalf("expected both generic type names to be linked: %#v", typeInfo.TypeParts)
	}
}

func TestResolveTypeLinksWebGPUBuiltins(t *testing.T) {
	utils.LoadWgslTypes()
	for _, name := range []string{
		"atomic", "texture_1d", "texture_2d", "texture_2d_array", "texture_3d",
		"texture_cube", "texture_cube_array", "texture_multisampled_2d",
		"texture_storage_1d", "texture_storage_2d", "texture_storage_2d_array",
		"texture_storage_3d", "texture_depth_2d", "texture_depth_2d_array",
		"texture_depth_multisampled_2d", "texture_external", "vec2f", "vec4u", "mat4x4h",
	} {
		if link := utils.GetTypeLink(name); link == "" {
			t.Errorf("missing WebGPU/WGSL built-in type link for %q", name)
		}
	}
	if link := utils.GetTypeLink("rgba32float"); link != "https://www.w3.org/TR/WGSL/#texel-formats" {
		t.Fatalf("unexpected texture format link: %q", link)
	}
}

func TestResolveModuleLinkPicksLongestPrefix(t *testing.T) {
	declaredImportPaths := map[string]string{
		"bevy_pbr":                     "/bevy_pbr/0.20.0/index.html",
		"bevy_pbr::mesh_view_bindings": "/bevy_pbr/0.20.0/mesh_view_bindings.html",
	}
	link, ok := ResolveModuleLink("bevy_pbr::mesh_view_bindings::view", declaredImportPaths)
	if !ok || link != "/bevy_pbr/0.20.0/mesh_view_bindings.html" {
		t.Fatalf("link = %q, ok = %v", link, ok)
	}
}

func TestResolveModuleLinkMisses(t *testing.T) {
	if link, ok := ResolveModuleLink("other::module::symbol", map[string]string{"bevy_pbr": "/x.html"}); ok {
		t.Fatalf("expected no match, got %q", link)
	}
}

func TestResolveUsageGroupsSinglePackage(t *testing.T) {
	refs, summary := ResolveUsage(map[string][]UsageRef{
		"bevy_pbr::mesh_view_bindings::view": {
			{Link: "/b.html", PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "b.wgsl"},
			{Link: "/a.html", PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "a.wgsl"},
		},
	}, "bevy_pbr::mesh_view_bindings::view")

	if summary != "Used by 2 shaders in bevy_pbr 0.20.0" {
		t.Fatalf("unexpected summary: %q", summary)
	}
	if len(refs) != 2 || refs[0].Filename != "a.wgsl" || refs[1].Filename != "b.wgsl" {
		t.Fatalf("expected refs sorted by filename, got %#v", refs)
	}
}

func TestResolveUsageMixedPackagesOmitsPackageFromSummary(t *testing.T) {
	_, summary := ResolveUsage(map[string][]UsageRef{
		"bevy_pbr::mesh_view_bindings::view": {
			{PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "a.wgsl"},
			{PackageName: "bevy_hanabi", PackageVersion: "0.19.0", Filename: "b.wgsl"},
		},
	}, "bevy_pbr::mesh_view_bindings::view")

	if summary != "Used by 2 shaders" {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func TestResolveUsageSingularNoun(t *testing.T) {
	_, summary := ResolveUsage(map[string][]UsageRef{
		"k": {{PackageName: "bevy_pbr", PackageVersion: "0.20.0", Filename: "a.wgsl"}},
	}, "k")

	if summary != "Used by 1 shader in bevy_pbr 0.20.0" {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func TestResolveUsageMissingKey(t *testing.T) {
	refs, summary := ResolveUsage(map[string][]UsageRef{}, "missing")
	if refs != nil || summary != "" {
		t.Fatalf("expected empty result, got refs=%#v summary=%q", refs, summary)
	}
}
