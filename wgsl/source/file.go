// Package source owns source text and location calculations.
package source

import "strings"

// File is normalized WGSL source together with cheap location helpers.
type File struct {
	Text string
}

func New(text string) File {
	return File{Text: strings.ReplaceAll(text, "\n\r", "\n")}
}

func (f File) LineAt(byteOffset int) int {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(f.Text) {
		byteOffset = len(f.Text)
	}
	return strings.Count(f.Text[:byteOffset], "\n") + 1
}
