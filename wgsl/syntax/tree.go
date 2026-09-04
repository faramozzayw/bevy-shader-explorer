// Package syntax hides Tree-sitter behind WGSL-neutral tree operations.
package syntax

import (
	"fmt"
	"unsafe"

	wgsl "github.com/gpuweb/tree-sitter-wgsl/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	wesl "github.com/webgpu-tools/tree-sitter-wesl/bindings/go"
)

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

type Node struct {
	raw  *tree_sitter.Node
	tree *Tree
}

func (n Node) Valid() bool  { return n.raw != nil }
func (n Node) Kind() string { return n.raw.Kind() }
func (n Node) Text() string { return n.raw.Utf8Text(n.tree.source) }
func (n Node) Line() int    { return int(n.raw.StartPosition().Row) + 1 }

func (n Node) Field(name string) Node {
	return Node{raw: n.raw.ChildByFieldName(name), tree: n.tree}
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
