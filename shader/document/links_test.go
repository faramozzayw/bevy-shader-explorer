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
}
