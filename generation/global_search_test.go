package generation

import "testing"

func TestCombineGlobalSearchInfoGroupsVersionsOnly(t *testing.T) {
	items := combineGlobalSearchInfo([]ShaderSearchableInfo{
		{PackageName: "bevy_pbr", PackageVersion: "0.18.1", Filename: "mesh.wgsl", Name: "view", Type: "binding", Link: "/bevy_pbr/0.18.1/mesh.html"},
		{PackageName: "bevy_pbr", PackageVersion: "0.19.1", Filename: "mesh.wgsl", Name: "view", Type: "binding", Link: "/bevy_pbr/0.19.1/mesh.html"},
		{PackageName: "bevy_pbr", PackageVersion: "0.19.1", Filename: "other.wgsl", Name: "view", Type: "binding", Link: "/bevy_pbr/0.19.1/other.html"},
	})

	if len(items) != 2 {
		t.Fatalf("got %d grouped results, want 2", len(items))
	}
	if !items[0].MultiVersion || len(items[0].Versions) != 2 {
		t.Fatalf("grouped result did not retain both versions: %#v", items[0])
	}
	if items[1].MultiVersion {
		t.Fatal("different source files should not be combined")
	}
}
