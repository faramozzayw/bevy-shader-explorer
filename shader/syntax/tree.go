// Package syntax hides Tree-sitter behind WGSL-neutral tree operations.
package syntax

import (
	"fmt"
	"strings"
	"unsafe"

	wgsl "github.com/gpuweb/tree-sitter-wgsl/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	wesl "github.com/webgpu-tools/tree-sitter-wesl/bindings/go"
)

// ConditionalBlock describes a WESL @if/@else attribute pair and the
// top-level declaration branch it controls.
type ConditionalBlock struct {
	Name      string
	Negated   bool
	IfLine    int
	ElseLine  *int
	EndifLine int
}

// LineRange identifies the source lines occupied by a syntax node.
type LineRange struct {
	Start int
	End   int
}

type Tree struct {
	source []byte
	parser *tree_sitter.Parser
	tree   *tree_sitter.Tree
	root   *tree_sitter.Node
}

func Parse(sourceText, parserText string) (*Tree, error) {
	return parse(sourceText, parserText, wgsl.Language(), "WGSL")
}

// ParseWESL parses WESL source. WESL is a WGSL-compatible superset, but has
// its own grammar for imports, conditional attributes, and other extensions.
func ParseWESL(sourceText, parserText string) (*Tree, error) {
	return parse(sourceText, parserText, wesl.Language(), "WESL")
}

// FunctionRanges returns the source ranges of WGSL function declarations.
// The ranges are used to distinguish conditional directives in function
// bodies from directives controlling top-level declarations.
func FunctionRanges(sourceText, parserText string) ([]LineRange, error) {
	tree, err := Parse(sourceText, parserText)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	return functionRanges(tree), nil
}

// WESLFunctionRanges returns the source ranges of WESL function declarations.
func WESLFunctionRanges(sourceText, parserText string) ([]LineRange, error) {
	return weslFunctionRanges(sourceText), nil
}

func functionRanges(tree *Tree) []LineRange {
	functions := tree.Root().Descendants("function_decl")
	ranges := make([]LineRange, 0, len(functions))
	for _, function := range functions {
		ranges = append(ranges, LineRange{Start: function.Line(), End: function.EndLine()})
	}
	return ranges
}

func parse(sourceText, parserText string, languagePtr unsafe.Pointer, languageName string) (*Tree, error) {
	parser := tree_sitter.NewParser()
	if err := parser.SetLanguage(tree_sitter.NewLanguage(languagePtr)); err != nil {
		parser.Close()
		return nil, fmt.Errorf("configure %s parser: %w", languageName, err)
	}
	source := []byte(sourceText)
	tree := parser.Parse([]byte(parserText), nil)
	return &Tree{source: source, parser: parser, tree: tree, root: tree.RootNode()}, nil
}

func (t *Tree) Close() {
	t.tree.Close()
	t.parser.Close()
}

func (t *Tree) Root() Node { return Node{raw: t.root, tree: t} }

// WESLConditionalBlocks extracts conditional attributes from the Tree-sitter
// tree. WESL represents @if/@else as attributes attached to the following
// declaration, so the declaration boundaries are available directly from the
// root's named-child sequence without line-oriented source heuristics.
func WESLConditionalBlocks(source string) ([]ConditionalBlock, error) {
	return scanWESLConditionals(source), nil
}

// WESL's conditional attributes decorate the following syntax node, rather
// than introducing a preprocessor block. Keeping this scanner source-based is
// intentional: older tree-sitter-wesl grammars recover malformed nodes when a
// file contains conditional statements inside functions, which can otherwise
// make one function consume the rest of the file.
func scanWESLConditionals(source string) []ConditionalBlock {
	lines := strings.Split(source, "\n")
	depths := make([]int, len(lines))
	depth := 0
	for i, line := range lines {
		depths[i] = depth
		depth += braceDelta(line)
		if depth < 0 {
			depth = 0
		}
	}

	blocks := make([]ConditionalBlock, 0)
	pending := make(map[int]int)
	for index, line := range lines {
		code := strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		for offset := 0; offset < len(code); {
			at := strings.IndexByte(code[offset:], '@')
			if at < 0 {
				break
			}
			at += offset
			kind, expression, end, ok := parseWESLAttribute(code, at)
			if !ok {
				offset = at + 1
				continue
			}
			attributeDepth := depths[index] + braceDelta(code[:at])
			if kind == "if" {
				_, nodeEnd := decoratedNodeRange(lines, index, end)
				blocks = append(blocks, ConditionalBlock{Name: expression, IfLine: index + 1, EndifLine: nodeEnd + 1})
				pending[attributeDepth] = len(blocks) - 1
			} else if kind == "else" {
				if blockIndex, exists := pending[attributeDepth]; exists {
					lineNumber := index + 1
					_, nodeEnd := decoratedNodeRange(lines, index, end)
					blocks[blockIndex].ElseLine = &lineNumber
					blocks[blockIndex].EndifLine = nodeEnd + 1
				}
			}
			offset = end
		}
	}
	return blocks
}

func parseWESLAttribute(line string, at int) (kind, expression string, end int, ok bool) {
	rest := line[at:]
	if strings.HasPrefix(rest, "@else") {
		return "else", "", at + len("@else"), true
	}
	if !strings.HasPrefix(rest, "@if") && !strings.HasPrefix(rest, "@elif") {
		return "", "", at + 1, false
	}
	nameEnd := strings.IndexByte(rest, '(')
	if nameEnd < 0 {
		return "", "", at + 1, false
	}
	open := at + nameEnd
	depth := 0
	for i := open; i < len(line); i++ {
		switch line[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				kind = "if"
				if strings.HasPrefix(rest, "@elif") {
					kind = "else"
				}
				return kind, strings.TrimSpace(line[open+1 : i]), i + 1, true
			}
		}
	}
	return "", "", at + 1, false
}

func decoratedNodeRange(lines []string, attributeLine, attributeEnd int) (int, int) {
	start := attributeLine
	if strings.TrimSpace(lines[attributeLine][attributeEnd:]) == "" {
		start++
		for start < len(lines) && strings.TrimSpace(strings.SplitN(lines[start], "//", 2)[0]) == "" {
			start++
		}
	}
	if start >= len(lines) {
		return attributeLine + 1, attributeLine + 1
	}
	braceSeen, balance := false, 0
	for line := start; line < len(lines); line++ {
		code := strings.SplitN(lines[line], "//", 2)[0]
		if !braceSeen && strings.HasPrefix(strings.TrimSpace(code), "}") {
			return start, line
		}
		for _, char := range code {
			switch char {
			case '{':
				braceSeen = true
				balance++
			case '}':
				if braceSeen {
					balance--
				}
			}
		}
		if braceSeen && balance == 0 {
			return start, line + 1
		}
		if !braceSeen && (strings.Contains(code, ";") || strings.Contains(code, ",")) {
			return start, line + 1
		}
	}
	return start, len(lines)
}

func braceDelta(line string) int {
	code := strings.SplitN(line, "//", 2)[0]
	delta := 0
	quoted := false
	for i := 0; i < len(code); i++ {
		if code[i] == '"' && (i == 0 || code[i-1] != '\\') {
			quoted = !quoted
			continue
		}
		if quoted {
			continue
		}
		switch code[i] {
		case '{':
			delta++
		case '}':
			delta--
		}
	}
	return delta
}

func weslFunctionRanges(source string) []LineRange {
	lines := strings.Split(source, "\n")
	depth := 0
	var ranges []LineRange
	for index, line := range lines {
		code := strings.SplitN(line, "//", 2)[0]
		if strings.Contains(code, "fn ") || strings.HasPrefix(strings.TrimSpace(code), "fn ") {
			if strings.Contains(code, "{") {
				_, end := decoratedNodeRange(lines, index, 0)
				ranges = append(ranges, LineRange{Start: index + 1, End: end})
			}
		}
		depth += braceDelta(line)
		_ = depth
	}
	return ranges
}

type weslConditionalScanner struct {
	blocks  []ConditionalBlock
	endLine int
}

func (scanner *weslConditionalScanner) scan(children []Node, scopeEnd int) {
	var current *ConditionalBlock
	branchHasContent := false
	braced := false
	close := func(endLine int) {
		if current == nil {
			return
		}
		current.EndifLine = endLine
		scanner.blocks = append(scanner.blocks, *current)
		current = nil
		branchHasContent = false
		braced = false
	}

	for index := 0; index < len(children); index++ {
		child := children[index]
		text := strings.TrimSpace(child.Text())
		if block, nextIndex, ok := bracedWESLConditional(children, index); ok {
			scanner.blocks = append(scanner.blocks, block)
			scanner.scan(child.Children()[1:], child.EndLine()+1)
			index = nextIndex
			continue
		}
		if child.Kind() == "attribute" {
			switch {
			case strings.HasPrefix(text, "@if"):
				close(child.Line())
				block := parseWESLConditional(text, child.Line())
				current = &block
			case strings.HasPrefix(text, "@else") && current != nil:
				line := child.Line()
				current.ElseLine = &line
				branchHasContent = false
				braced = false
			case current != nil && branchHasContent && !braced:
				close(child.Line())
			}
		} else if current != nil {
			switch {
			case strings.HasPrefix(text, "{") && !branchHasContent:
				braced = true
			case braced && strings.HasPrefix(text, "}"):
				close(child.Line())
			default:
				branchHasContent = true
			}
		}
		scanner.scan(child.Children(), child.EndLine()+1)
	}
	close(scopeEnd)
}

func parseWESLConditional(text string, line int) ConditionalBlock {
	name := text[strings.IndexByte(text, '(')+1:]
	if end := strings.IndexByte(name, ')'); end >= 0 {
		name = strings.TrimSpace(name[:end])
	}
	negated := strings.HasPrefix(name, "!")
	if negated {
		name = strings.TrimSpace(strings.TrimPrefix(name, "!"))
	}
	return ConditionalBlock{Name: name, Negated: negated, IfLine: line}
}

func bracedWESLConditional(children []Node, index int) (ConditionalBlock, int, bool) {
	child := children[index]
	if child.Kind() != "compound_statement" {
		return ConditionalBlock{}, index, false
	}
	inner := child.Children()
	if len(inner) == 0 || inner[0].Kind() != "attribute" || !strings.HasPrefix(strings.TrimSpace(inner[0].Text()), "@if") {
		return ConditionalBlock{}, index, false
	}

	block := parseWESLConditional(strings.TrimSpace(inner[0].Text()), inner[0].Line())
	block.EndifLine = child.EndLine() + 1
	nextIndex := index
	elseIndex := index + 1
	var elseNode Node
	if elseIndex < len(children) && isWESLAttribute(children[elseIndex], "@else") {
		elseNode = children[elseIndex]
	} else if elseIndex < len(children) && children[elseIndex].Kind() == "compound_statement" {
		elseChildren := children[elseIndex].Children()
		if len(elseChildren) > 0 && isWESLAttribute(elseChildren[0], "@else") {
			elseNode = elseChildren[0]
		}
	}
	if elseNode.Valid() {
		line := elseNode.Line()
		block.ElseLine = &line
		if children[elseIndex].Kind() == "compound_statement" {
			block.EndifLine = children[elseIndex].EndLine() + 1
			nextIndex = elseIndex
		} else if elseIndex+1 < len(children) && children[elseIndex+1].Kind() == "compound_statement" {
			block.EndifLine = children[elseIndex+1].EndLine() + 1
			nextIndex = elseIndex + 1
		}
	}
	return block, nextIndex, true
}

func isWESLAttribute(node Node, prefix string) bool {
	return node.Kind() == "attribute" && strings.HasPrefix(strings.TrimSpace(node.Text()), prefix)
}

type Node struct {
	raw  *tree_sitter.Node
	tree *Tree
}

func (n Node) Valid() bool  { return n.raw != nil }
func (n Node) Kind() string { return n.raw.Kind() }
func (n Node) Text() string { return n.raw.Utf8Text(n.tree.source) }
func (n Node) Line() int    { return int(n.raw.StartPosition().Row) + 1 }
func (n Node) EndLine() int { return int(n.raw.EndPosition().Row) + 1 }

func (n Node) Field(name string) Node {
	return Node{raw: n.raw.ChildByFieldName(name), tree: n.tree}
}

func (n Node) PreviousNamedSibling() Node {
	return Node{raw: n.raw.PrevNamedSibling(), tree: n.tree}
}

func (n Node) NextNamedSibling() Node {
	return Node{raw: n.raw.NextNamedSibling(), tree: n.tree}
}

func (n Node) Parent() Node {
	return Node{raw: n.raw.Parent(), tree: n.tree}
}

func (n Node) Children() []Node {
	children := make([]Node, 0, n.raw.NamedChildCount())
	for i := uint(0); i < n.raw.NamedChildCount(); i++ {
		children = append(children, Node{raw: n.raw.NamedChild(i), tree: n.tree})
	}
	return children
}

func (n Node) IsNamed() bool { return n.raw.IsNamed() }

func (n Node) StartByte() int { return int(n.raw.StartByte()) }

func (n Node) EndByte() int { return int(n.raw.EndByte()) }

func (n Node) ChildCount() uint { return n.raw.ChildCount() }

func (n Node) Child(i uint) Node {
	return Node{raw: n.raw.Child(i), tree: n.tree}
}

func (n Node) FirstChild(kind string) Node {
	for _, child := range n.Children() {
		if child.Kind() == kind {
			return child
		}
	}
	return Node{}
}

func (n Node) Descendants(kind string) []Node {
	var result []Node
	var visit func(Node)
	visit = func(current Node) {
		for _, child := range current.Children() {
			if child.Kind() == kind {
				result = append(result, child)
			}
			visit(child)
		}
	}
	visit(n)
	return result
}
