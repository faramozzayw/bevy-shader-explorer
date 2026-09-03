package document

import "testing"

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
