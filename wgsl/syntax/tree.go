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
	tree, err := ParseWESL(source, source)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var blocks []ConditionalBlock
	parseConditional := func(text string, line int) ConditionalBlock {
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
	var scan func([]Node)
	scan = func(children []Node) {
		var current *ConditionalBlock
		branchHasContent := false
		braced := false
		close := func(endLine int) {
			if current == nil {
				return
			}
			current.EndifLine = endLine
			blocks = append(blocks, *current)
			current = nil
			branchHasContent = false
			braced = false
		}
		for index := 0; index < len(children); index++ {
			child := children[index]
			text := strings.TrimSpace(child.Text())
			// Braced WESL conditionals inside a function are represented as a
			// compound_statement whose first child is the @if attribute. The
			// sibling sequence contains the optional @else branch.
			if child.Kind() == "compound_statement" {
				inner := child.Children()
				if len(inner) > 0 && inner[0].Kind() == "attribute" && strings.HasPrefix(strings.TrimSpace(inner[0].Text()), "@if") {
					block := parseConditional(strings.TrimSpace(inner[0].Text()), inner[0].Line())
					block.EndifLine = child.EndLine() + 1
					elseNode := Node{}
					elseIndex := index + 1
					if elseIndex < len(children) && children[elseIndex].Kind() == "attribute" && strings.HasPrefix(strings.TrimSpace(children[elseIndex].Text()), "@else") {
						elseNode = children[elseIndex]
					} else if elseIndex < len(children) && children[elseIndex].Kind() == "compound_statement" {
						elseChildren := children[elseIndex].Children()
						if len(elseChildren) > 0 && elseChildren[0].Kind() == "attribute" && strings.HasPrefix(strings.TrimSpace(elseChildren[0].Text()), "@else") {
							elseNode = elseChildren[0]
						}
					}
					if elseNode.Valid() {
						line := elseNode.Line()
						block.ElseLine = &line
						if children[elseIndex].Kind() == "compound_statement" {
							block.EndifLine = children[elseIndex].EndLine() + 1
							index = elseIndex
						} else if elseIndex+1 < len(children) && children[elseIndex+1].Kind() == "compound_statement" {
							block.EndifLine = children[elseIndex+1].EndLine() + 1
							index = elseIndex + 1
						}
					}
					blocks = append(blocks, block)
					scan(inner[1:])
					continue
				}
			}
			if child.Kind() == "attribute" {
				if strings.HasPrefix(text, "@if") {
					if current != nil {
						close(child.Line())
					}
					currentBlock := parseConditional(text, child.Line())
					current = &currentBlock
					branchHasContent = false
					braced = false
				} else if strings.HasPrefix(text, "@else") && current != nil {
					line := child.Line()
					current.ElseLine = &line
					branchHasContent = false
					braced = false
				} else if current != nil && branchHasContent && !braced {
					close(child.Line())
				}
			} else if current != nil {
				if strings.HasPrefix(text, "{") && !branchHasContent {
					braced = true
				} else if braced && strings.HasPrefix(text, "}") {
					close(child.Line())
				} else {
					branchHasContent = true
				}
			}
			scan(child.Children())
		}
		if current != nil {
			close(len(strings.Split(source, "\n")) + 1)
		}
	}
	scan(tree.Root().Children())
	return blocks, nil
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
