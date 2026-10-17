package highlight

import (
	"html"
	"sort"
	"strings"
	"unicode"

	"github.com/faramozzayw/bevy-shader-explorer/shader/syntax"
	wgsl "github.com/faramozzayw/bevy-shader-explorer/shader/wgsl"
)

type Line struct {
	Number int
	HTML   string
}

type tokenKind int

const (
	kindPlain tokenKind = iota
	kindKeyword
	kindType
	kindAttribute
	kindDirective
	kindComment
	kindNumber
)

var cssClass = map[tokenKind]string{
	kindKeyword:   "tok-kw",
	kindType:      "tok-type",
	kindAttribute: "tok-attr",
	kindDirective: "tok-directive",
	kindComment:   "tok-comment",
	kindNumber:    "tok-number",
}

var keywords = map[string]bool{
	"alias": true, "break": true, "case": true, "const": true,
	"const_assert": true, "continue": true, "continuing": true, "default": true,
	"diagnostic": true, "discard": true, "else": true, "enable": true,
	"false": true, "fn": true, "for": true, "if": true, "let": true,
	"loop": true, "override": true, "requires": true, "return": true,
	"struct": true, "switch": true, "true": true, "var": true, "while": true,
	"as": true, "import": true,
}

var types = map[string]bool{
	"bool": true, "i32": true, "u32": true, "f32": true, "f16": true,
	"array": true, "ptr": true, "atomic": true,
	"sampler": true, "sampler_comparison": true,
	"texture_1d": true, "texture_2d": true, "texture_2d_array": true,
	"texture_3d": true, "texture_cube": true, "texture_cube_array": true,
	"texture_multisampled_2d": true, "texture_depth_multisampled_2d": true,
	"texture_depth_2d": true, "texture_depth_2d_array": true,
	"texture_depth_cube": true, "texture_depth_cube_array": true,
	"texture_storage_1d": true, "texture_storage_2d": true,
	"texture_storage_2d_array": true, "texture_storage_3d": true,
	"texture_external": true,
}

func init() {
	for _, prefix := range []string{"vec2", "vec3", "vec4"} {
		types[prefix] = true
		for _, suffix := range []string{"i", "u", "f", "h"} {
			types[prefix+suffix] = true
		}
	}
	for rows := 2; rows <= 4; rows++ {
		for cols := 2; cols <= 4; cols++ {
			base := "mat" + string(rune('0'+rows)) + "x" + string(rune('0'+cols))
			types[base] = true
			for _, suffix := range []string{"f", "h"} {
				types[base+suffix] = true
			}
		}
	}
}

var numberKinds = map[string]bool{
	"decimal_int_literal": true, "hex_int_literal": true,
	"decimal_float_literal": true, "hex_float_literal": true,
	"int_literal": true, "float_literal": true,
}

var identifierKinds = map[string]bool{
	"identifier": true, "ident_pattern_token": true,
}

var commentKinds = map[string]bool{
	"line_comment": true, "block_comment": true,
}

type rawToken struct {
	kind  tokenKind
	start int
	end   int
}

func Lines(source, filename string) []Line {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	if strings.HasSuffix(source, "\n") {
		source = source[:len(source)-1]
	}
	if source == "" {
		return nil
	}

	tokens := tokenize(source, filename)
	return render(source, tokens)
}

func tokenize(source, filename string) []rawToken {
	tree, ok := parseTree(source, filename)
	if !ok {
		return []rawToken{{kind: kindPlain, start: 0, end: len(source)}}
	}
	defer tree.Close()

	leaves := collectLeaves(tree.Root())
	sort.Slice(leaves, func(i, j int) bool { return leaves[i].start < leaves[j].start })

	tokens := make([]rawToken, 0, len(leaves)*2)
	pos := 0
	for _, leaf := range leaves {
		if leaf.start > pos {
			tokens = append(tokens, scanGap(source, pos, leaf.start)...)
		}
		if leaf.end <= leaf.start {
			continue
		}
		tokens = append(tokens, leaf)
		pos = leaf.end
	}
	if pos < len(source) {
		tokens = append(tokens, scanGap(source, pos, len(source))...)
	}
	return tokens
}

func parseTree(source, filename string) (*syntax.Tree, bool) {
	if strings.HasSuffix(filename, ".wesl") {
		tree, err := syntax.ParseWESL(source, source)
		if err != nil {
			return nil, false
		}
		return tree, true
	}
	tree, err := syntax.Parse(source, wgsl.MaskDirectives(source))
	if err != nil {
		return nil, false
	}
	return tree, true
}

func collectLeaves(n syntax.Node) []rawToken {
	var leaves []rawToken
	var walk func(syntax.Node)
	walk = func(current syntax.Node) {
		if current.Kind() == "preproc_custom" {
			leaves = append(leaves, rawToken{kind: kindDirective, start: current.StartByte(), end: current.EndByte()})
			return
		}
		if current.ChildCount() == 0 {
			leaves = append(leaves, classifyLeaf(current))
			return
		}
		for i := uint(0); i < current.ChildCount(); i++ {
			walk(current.Child(i))
		}
	}
	walk(n)
	return leaves
}

func classifyLeaf(n syntax.Node) rawToken {
	start, end := n.StartByte(), n.EndByte()
	kind := n.Kind()
	text := n.Text()

	switch {
	case commentKinds[kind]:
		return rawToken{kind: kindComment, start: start, end: end}
	case numberKinds[kind]:
		return rawToken{kind: kindNumber, start: start, end: end}
	case identifierKinds[kind]:
		if isAttributeLeaf(n) {
			return rawToken{kind: kindAttribute, start: start, end: end}
		}
		if types[text] {
			return rawToken{kind: kindType, start: start, end: end}
		}
	case !n.IsNamed():
		if (text == "@" || isIdentLike(text)) && isAttributeLeaf(n) {
			return rawToken{kind: kindAttribute, start: start, end: end}
		}
		if keywords[text] {
			return rawToken{kind: kindKeyword, start: start, end: end}
		}
	}
	return rawToken{kind: kindPlain, start: start, end: end}
}

func isAttributeLeaf(n syntax.Node) bool {
	parent := n.Parent()
	if !parent.Valid() {
		return false
	}
	kind := parent.Kind()
	return kind == "attribute" || strings.HasSuffix(kind, "_attr")
}

func isIdentLike(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func scanGap(source string, start, end int) []rawToken {
	var tokens []rawToken
	atLineStart := start == 0 || source[start-1] == '\n'
	plainStart := -1
	flushPlain := func(stop int) {
		if plainStart >= 0 && plainStart < stop {
			tokens = append(tokens, rawToken{kind: kindPlain, start: plainStart, end: stop})
		}
		plainStart = -1
	}

	i := start
	for i < end {
		switch {
		case strings.HasPrefix(source[i:end], "//"):
			flushPlain(i)
			stop := end
			if j := strings.IndexByte(source[i:end], '\n'); j >= 0 {
				stop = i + j
			}
			tokens = append(tokens, rawToken{kind: kindComment, start: i, end: stop})
			i = stop
			atLineStart = false

		case strings.HasPrefix(source[i:end], "/*"):
			flushPlain(i)
			stop := end
			if j := strings.Index(source[i:end], "*/"); j >= 0 {
				stop = i + j + 2
			}
			tokens = append(tokens, rawToken{kind: kindComment, start: i, end: stop})
			i = stop
			atLineStart = false

		case source[i] == '#' && atLineStart:
			flushPlain(i)
			stop := end
			if j := strings.IndexByte(source[i:end], '\n'); j >= 0 {
				stop = i + j
			}
			tokens = append(tokens, rawToken{kind: kindDirective, start: i, end: stop})
			i = stop
			atLineStart = false

		default:
			if plainStart < 0 {
				plainStart = i
			}
			if source[i] == '\n' {
				atLineStart = true
			} else if source[i] != ' ' && source[i] != '\t' && source[i] != '\r' {
				atLineStart = false
			}
			i++
		}
	}
	flushPlain(end)
	return tokens
}

func render(source string, tokens []rawToken) []Line {
	var lines []Line
	var buf strings.Builder
	number := 1
	flush := func() {
		lines = append(lines, Line{Number: number, HTML: buf.String()})
		buf.Reset()
		number++
	}
	for _, tok := range tokens {
		class := cssClass[tok.kind]
		segments := strings.Split(source[tok.start:tok.end], "\n")
		for i, segment := range segments {
			if segment != "" {
				escaped := html.EscapeString(segment)
				if class != "" {
					buf.WriteString(`<span class="`)
					buf.WriteString(class)
					buf.WriteString(`">`)
					buf.WriteString(escaped)
					buf.WriteString(`</span>`)
				} else {
					buf.WriteString(escaped)
				}
			}
			if i < len(segments)-1 {
				flush()
			}
		}
	}
	flush()
	return lines
}
