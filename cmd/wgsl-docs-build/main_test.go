package main

import (
	"path/filepath"
	"testing"
)

func TestReleaseMatrixContainsKnownRefs(t *testing.T) {
	sources, err := loadSources(filepath.Join("..", ".."), "wgsl-docs-build.toml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
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
