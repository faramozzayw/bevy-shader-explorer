package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

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

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to write destination file: %v", err)
	}

	return nil
}

// checks if any item has shader definitions
func AnyShaderDefs[T any](input []T) bool {
	for _, v := range input {
		val := reflect.ValueOf(v)
		field := val.FieldByName("HasShaderDefs")
		if field.IsValid() && field.Kind() == reflect.Bool && field.Bool() {
			return true
		}
	}
	return false
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
