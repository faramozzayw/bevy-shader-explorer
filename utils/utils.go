package utils

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

//go:embed wgpu-types.json
var WgpuTypesData []byte
var wgpuTypes map[string]string
var wgpuTypesOnce sync.Once

func LoadWgslTypes() {
	wgpuTypesOnce.Do(func() {
		err := json.Unmarshal(WgpuTypesData, &wgpuTypes)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse wgpu-types.json: %v", err))
		}
	})
}

func SplitParams(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}

	var parts []string
	var current strings.Builder
	depth := 0

	for _, char := range s {
		switch char {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
		}
		current.WriteRune(char)
	}

	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		parts = append(parts, trimmed)
	}

	return parts
}

func RemovePath(s string) string {
	parts := strings.Split(strings.TrimSpace(s), "::")
	return parts[len(parts)-1]
}

func GetTypeLink(t string) string {
	baseType := strings.TrimSpace(t)
	if i := strings.Index(baseType, "<"); i != -1 {
		baseType = baseType[:i]
	}
	if link, ok := wgpuTypes[baseType]; ok {
		return link
	}
	return ""
}

func NormalizeLink(link string) string {
	link = strings.ReplaceAll(link, "src/", "")
	if strings.HasPrefix(link, "/") {
		return link
	} else {
		return "/" + link
	}
}

func CopyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %v", err)
	}

	err = WriteFileIfChanged(dst, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to write destination file: %v", err)
	}

	return nil
}

// WriteFileIfChanged avoids replacing an output file whose content is already
// current. Besides reducing I/O this preserves mtimes for deploy tooling that
// uses them to detect changed artifacts.
func WriteFileIfChanged(path string, data []byte, perm os.FileMode) error {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == string(data) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, data, perm)
}

func DedupPathParts(path string) string {
	seen := map[string]struct{}{}
	parts := strings.Split(path, "/")
	var result []string

	for _, part := range parts {
		if _, exists := seen[part]; !exists {
			seen[part] = struct{}{}
			result = append(result, part)
		}
	}

	return strings.Join(result, "/")
}

func ValueOrDefault[T any](ptr *T, fallback T) T {
	if ptr != nil {
		return *ptr
	}
	return fallback
}

func PrintAsJson[T any](v T) {
	printJson, err := json.MarshalIndent(v, "", "  ")

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(printJson))
}
