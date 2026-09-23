package generation

import (
	"path/filepath"
	"slices"
	"strings"
)

type moduleGroup struct {
	Name   string
	Files  []map[string]string
	Count  int
	IsRoot bool
}

func groupShaderModules(files []map[string]string) []moduleGroup {
	groups := make(map[string][]map[string]string)
	for _, file := range files {
		relative := file["relative"]
		if relative == "" {
			relative = file["file"]
		}
		directory := filepath.ToSlash(filepath.Dir(relative))
		if directory == "." || directory == "" {
			directory = "Root"
		}
		groups[directory] = append(groups[directory], file)
	}

	result := make([]moduleGroup, 0, len(groups))
	for name, entries := range groups {
		slices.SortFunc(entries, func(a, b map[string]string) int {
			return strings.Compare(a["file"], b["file"])
		})
		result = append(result, moduleGroup{Name: name, Files: entries, Count: len(entries), IsRoot: name == "Root"})
	}
	slices.SortFunc(result, func(a, b moduleGroup) int {
		if a.Name == "Root" {
			return -1
		}
		if b.Name == "Root" {
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result
}
