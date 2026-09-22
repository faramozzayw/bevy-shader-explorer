package bevy

import "strings"

// MaskDirectives replaces Bevy-only directive text with spaces while preserving
// every newline and byte offset. The official WGSL parser can therefore parse
// the remaining WGSL while extracted declarations still point into the original
// source file.
func MaskDirectives(source string) string {
	lines := strings.SplitAfter(source, "\n")
	masked := make([]string, len(lines))
	importDepth := 0

	for index, line := range lines {
		content := strings.TrimSuffix(line, "\n")
		trimmed := strings.TrimSpace(content)
		isImportLine := importDepth > 0 || strings.HasPrefix(trimmed, "#import")
		isDirective := strings.HasPrefix(trimmed, "#")

		if isImportLine || isDirective {
			masked[index] = strings.Repeat(" ", len(content))
			if strings.HasSuffix(line, "\n") {
				masked[index] += "\n"
			}
		} else {
			masked[index] = line
		}

		if isImportLine {
			importDepth += strings.Count(content, "{") - strings.Count(content, "}")
		}
	}

	return strings.Join(masked, "")
}
