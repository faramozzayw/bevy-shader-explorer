package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllReleasesResolvesRefPatternAndOverride(t *testing.T) {
	sources := []source{{
		Name: "demo", Repo: "https://example.invalid/demo", Root: "sources/demo",
		RefPattern:     "release-{version}",
		Versions:       []string{"1.2.3", "1.2.4"},
		Releases:       []versionRef{{Version: "1.2.4", Ref: "special"}},
		PackageAliases: map[string]string{"example": "demo_example"},
	}}
	got := allReleases(sources)
	if len(got) != 2 || got[0].Ref != "release-1.2.3" || got[1].Ref != "special" {
		t.Fatalf("resolved refs = %q, %q", got[0].Ref, got[1].Ref)
	}
	if got[0].PackageAliases["example"] != "demo_example" {
		t.Fatalf("package aliases = %#v", got[0].PackageAliases)
	}
}

func TestLoadSourcesReadsRefPattern(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "matrix.toml")
	contents := "[[sources]]\nname = \"demo\"\nrepo = \"https://example.invalid/demo\"\nroot = \"sources/demo\"\nref_pattern = \"v{version}\"\npackage_aliases = { example = \"demo_example\" }\n\n[[sources.releases]]\nversion = \"2.0.0\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	sources, err := loadSources(root, "matrix.toml")
	if err != nil {
		t.Fatal(err)
	}
	if got := allReleases(sources)[0].Ref; got != "v2.0.0" {
		t.Fatalf("resolved ref = %q, want v2.0.0", got)
	}
	if got := sources[0].PackageAliases["example"]; got != "demo_example" {
		t.Fatalf("package alias = %q, want demo_example", got)
	}
}

func TestLoadSiteURL(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "matrix.toml"), []byte("site_url = \"https://docs.example/\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadSiteURL(root, "matrix.toml")
	if err != nil || got != "https://docs.example" {
		t.Fatalf("site URL = %q, err = %v", got, err)
	}
}

func TestReleaseMatrixContainsKnownRefs(t *testing.T) {
	sources, err := loadSources(filepath.Join("..", ".."), "shader-sources.toml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"sources/aqua":                    "v0.1.3",
		"sources/bevy/0.19.1":             "release-0.19.1",
		"sources/hanabi/0.19.0":           "v0.19.0",
		"sources/bevy_water/0.16.0":       "bevy_0.16",
		"sources/bevy_mod_outline/0.19.0": "bevy-0.19",
	}
	for dir, ref := range want {
		found := false
		for _, item := range allReleases(sources) {
			if item.Dir == dir {
				found = true
				if item.Ref != ref {
					t.Errorf("%s ref = %q, want %q", dir, item.Ref, ref)
				}
			}
		}
		if !found {
			t.Errorf("release %s missing from matrix", dir)
		}
	}
}
