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
		if !isImportLine && !isDirective {
			masked[index] = maskInlineExpressions(masked[index])
		}
	}

	return strings.Join(masked, "")
}

// maskInlineExpressions replaces Bevy's #{...} interpolation syntax with a
// parser-safe numeric literal while preserving the original byte width. The
// extractor uses source positions against the original source, so preserving
// width keeps the complete interpolation (including its closing brace) in the
// displayed declaration value.
func maskInlineExpressions(line string) string {
	var masked []byte
	for start := 0; start+1 < len(line); {
		if line[start] != '#' || line[start+1] != '{' {
			start++
			continue
		}
		end := start + 2
		depth := 1
		for end < len(line) && depth > 0 {
			switch line[end] {
			case '{':
				depth++
			case '}':
				depth--
			}
			end++
		}
		if depth != 0 {
			start += 2
			continue
		}
		if masked == nil {
			masked = []byte(line)
		}
		masked[start] = '0'
		for index := start + 1; index < end; index++ {
			masked[index] = ' '
		}
		start = end
	}
	if masked == nil {
		return line
	}
	return string(masked)
}
