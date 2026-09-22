package bevy

import (
	"sort"
	"strings"
)

// DefinitionBlock describes a Bevy conditional section in source text.
type DefinitionBlock struct {
	Name      string
	IfLine    int
	ElseLine  *int
	EndifLine int
}

func DefinitionBlocks(source string) []DefinitionBlock {
	var blocks, stack []DefinitionBlock
	for index, line := range strings.Split(source, "\n") {
		text, lineNumber := strings.TrimSpace(line), index+1
		switch {
		case strings.HasPrefix(text, "#ifdef"):
			stack = append(stack, DefinitionBlock{Name: strings.TrimSpace(text[6:]), IfLine: lineNumber})
		case strings.HasPrefix(text, "#else") && len(stack) > 0:
			current := &stack[len(stack)-1]
			if current.ElseLine == nil {
				current.ElseLine = &lineNumber
			}
		case strings.HasPrefix(text, "#endif") && len(stack) > 0:
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			current.EndifLine = lineNumber
			blocks = append(blocks, current)
		}
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].IfLine < blocks[j].IfLine })
	return blocks
}
