package parser

import "testing"

func TestExtractImportPathTrimsTrailingCarriageReturn(t *testing.T) {
	path := ExtractImportPath("#define_import_path bevy_hanabi::vfx_common\r\nstruct SimParams {\r\n")
	if path == nil || *path != "bevy_hanabi::vfx_common" {
		got := "nil"
		if path != nil {
			got = *path
		}
		t.Fatalf("ExtractImportPath = %q, want %q", got, "bevy_hanabi::vfx_common")
	}
}

func TestExtractImportPathTrimsTrailingWhitespace(t *testing.T) {
	path := ExtractImportPath("#define_import_path bevy_pbr::mesh_view_bindings   \nstruct Foo {}")
	if path == nil || *path != "bevy_pbr::mesh_view_bindings" {
		got := "nil"
		if path != nil {
			got = *path
		}
		t.Fatalf("ExtractImportPath = %q, want %q", got, "bevy_pbr::mesh_view_bindings")
	}
}

func TestExtractImportPathMissing(t *testing.T) {
	if path := ExtractImportPath("struct Foo {}"); path != nil {
		t.Fatalf("expected nil import path, got %q", *path)
	}
}
