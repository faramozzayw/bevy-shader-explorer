package extract

import (
	"strings"

	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
	"github.com/faramozzayw/bevy-shader-explorer/shader/syntax"
	utils "github.com/faramozzayw/bevy-shader-explorer/utils"
)

type Annotation = document.Annotation
type TypeInfo = document.TypeInfo
type NamedType = document.NamedType
type Const = document.Const
type Structure = document.Structure
type Function = document.Function
type Binding = document.Binding
type DefResult = document.DefResult

type ShaderDefBlock struct {
	Condition document.ShaderCondition
	Negated   bool
	IfdefLine int
	ElseLine  *int
	EndifLine int
}

type Result struct {
	Consts     []Const
	Structures []Structure
	Functions  []Function
	Bindings   []Binding
}

type syntaxParser func(sourceText, parserText string) (*syntax.Tree, error)

// ParseWithParser extracts declarations using the grammar adapter supplied by
// the language package. The shared extractor owns traversal; WGSL and WESL
// packages own the Tree-sitter grammar selection.
func ParseWithParser(code, parserCode string, comments map[int]string, shaderDefs []ShaderDefBlock, parser func(string, string) (*syntax.Tree, error)) (Result, error) {
	return parseWithGrammar(code, parserCode, comments, shaderDefs, parser)
}

func parseWithGrammar(code, parserCode string, comments map[int]string, shaderDefs []ShaderDefBlock, parser syntaxParser) (Result, error) {
	extractor, err := newSyntaxExtractor(code, parserCode, comments, shaderDefs, parser)
	if err != nil {
		return Result{}, err
	}
	defer extractor.Close()
	return Result{Consts: extractor.consts(), Structures: extractor.structures(), Functions: extractor.functions(), Bindings: extractor.bindings()}, nil
}

func getShaderDefsByLine(defs []ShaderDefBlock, line int) []DefResult {
	var result []DefResult
	for _, def := range defs {
		ifEnd := def.EndifLine
		if def.ElseLine != nil {
			ifEnd = *def.ElseLine
		}
		if line > def.IfdefLine && line < ifEnd {
			branch := "if"
			if def.Negated {
				branch = "else"
			}
			result = append(result, DefResult{
				Condition:  def.Condition,
				Branch:     branch,
				LineNumber: def.IfdefLine,
			})
		}
		if def.ElseLine != nil && line > *def.ElseLine && line < def.EndifLine {
			result = append(result, DefResult{
				Condition:  def.Condition,
				Branch:     "else",
				LineNumber: *def.ElseLine,
			})
		}
	}
	return result
}

func getItemComments(line int, comments map[int]string) []string {
	var result []string
	for current := line - 1; current > 0 && comments[current] != ""; current-- {
		result = append([]string{comments[current]}, result...)
	}
	return result
}

// syntaxExtractor is the single source of declaration locations. Tree-sitter
// keeps source positions attached to nodes, avoiding text searches that select
// the first occurrence of a repeated field or parameter name.
type syntaxExtractor struct {
	root         syntax.Node
	tree         *syntax.Tree
	source       string
	lineComments map[int]string
	shaderDefs   []ShaderDefBlock
}

func newSyntaxExtractor(code, parserCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock, parser syntaxParser) (*syntaxExtractor, error) {
	tree, err := parser(code, parserCode)
	if err != nil {
		return nil, err
	}
	return &syntaxExtractor{root: tree.Root(), tree: tree, source: code, lineComments: lineComments, shaderDefs: shaderDefs}, nil
}

func (e *syntaxExtractor) Close() {
	e.tree.Close()
}

func (e *syntaxExtractor) text(node syntax.Node) string {
	if !node.Valid() {
		return ""
	}
	return node.Text()
}

func (e *syntaxExtractor) line(node syntax.Node) int {
	return node.Line()
}

func (e *syntaxExtractor) declarationValue(node syntax.Node) string {
	lines := strings.Split(e.source, "\n")
	line := e.line(node) - 1
	if line < 0 || line >= len(lines) {
		return ""
	}
	text := strings.Join(lines[line:], "\n")
	if end := strings.IndexByte(text, ';'); end >= 0 {
		text = text[:end]
	}
	_, value, found := strings.Cut(text, "=")
	if !found {
		return ""
	}
	return strings.TrimSpace(value)
}

func childOfKind(node syntax.Node, kind string) syntax.Node {
	for _, child := range node.Children() {
		if child.Kind() == kind {
			return child
		}
	}
	// WESL's grammar uses the clearer `identifier` node name where the
	// original WGSL grammar uses `ident`/`member_ident`.
	if kind == "ident" || kind == "member_ident" {
		return childOfKind(node, "identifier")
	}
	return syntax.Node{}
}

func descendantsOfKind(node syntax.Node, kind string) []syntax.Node {
	return node.Descendants(kind)
}

func annotationsFrom(node syntax.Node) []Annotation {
	var annotations []Annotation
	if node.Valid() && node.Kind() == "attribute" {
		return []Annotation{annotationFromText(node.Text())}
	}
	for _, child := range node.Children() {
		if child.Kind() != "attribute" && !strings.HasSuffix(child.Kind(), "_attr") {
			continue
		}
		annotations = append(annotations, annotationFromText(child.Text()))
	}
	return annotations
}

func annotationFromText(text string) Annotation {
	text = strings.TrimPrefix(text, "@")
	name, value, hasValue := strings.Cut(text, "(")
	if hasValue {
		value = strings.TrimSuffix(value, ")")
	}
	return Annotation{Name: strings.TrimSpace(name), Value: strings.TrimSpace(value)}
}

// annotationsForDeclaration handles both grammars. WGSL keeps attributes as
// children of the declaration, while WESL's decorated declaration rule emits
// them as preceding sibling nodes.
func annotationsForDeclaration(node syntax.Node) []Annotation {
	annotations := annotationsFrom(node)
	for current := node; current.Valid(); current = current.Parent() {
		for sibling := current.PreviousNamedSibling(); sibling.Valid() && sibling.Kind() == "attribute"; sibling = sibling.PreviousNamedSibling() {
			annotation := annotationsFrom(sibling)
			if len(annotation) > 0 {
				annotations = append(annotation, annotations...)
			}
		}
	}
	filtered := annotations[:0]
	for _, annotation := range annotations {
		// WESL conditional directives are represented as attributes by the
		// grammar, but they are rendered separately as shader-def pills.
		if annotation.Name == "if" || annotation.Name == "else" {
			continue
		}
		filtered = append(filtered, annotation)
	}
	return filtered
}

func (e *syntaxExtractor) namedType(node syntax.Node, annotations []Annotation) NamedType {
	name := childOfKind(node, "ident")
	if !name.Valid() {
		name = childOfKind(node, "member_ident")
	}
	if !name.Valid() {
		name = node.Field("name")
	}
	line := e.line(node)
	if name.Valid() {
		line = e.line(name)
	}
	shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
	typ := childOfKind(node, "type_specifier")
	if !typ.Valid() {
		typ = node.Field("type")
	}
	fullType := e.text(typ)
	return NamedType{
		Annotations: annotations,
		Name:        e.text(name),
		ShaderDefs:  shaderDefs,
		TypeInfo:    TypeInfo{Type: utils.RemovePath(fullType), FullTypePath: fullType},
	}
}

func (e *syntaxExtractor) consts() []Const {
	var result []Const
	for _, node := range descendantsOfKind(e.root, "global_value_decl") {
		declaration := childOfKind(node, "optionally_typed_ident")
		if !declaration.Valid() {
			declaration = node
		}
		line := e.line(node)
		shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
		value := e.declarationValue(node)
		if value == "" {
			value = e.text(node)
			if _, after, found := strings.Cut(value, "="); found {
				value = strings.TrimSpace(strings.TrimSuffix(after, ";"))
			}
		}
		item := e.namedType(declaration, annotationsForDeclaration(node))
		typ := item.TypeInfo.Type
		if typ == "" {
			typ = inferConstType(value)
		}
		result = append(result, Const{LineNumber: line, Name: item.Name, Value: value, ShaderDefs: shaderDefs, TypeInfo: TypeInfo{Type: typ, FullTypePath: item.TypeInfo.FullTypePath}})
	}
	return result
}

func inferConstType(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "u") {
		return "u32"
	}
	if strings.HasPrefix(value, "vec") {
		if end := strings.Index(value, "("); end > 0 {
			return value[:end]
		}
	}
	if value == "true" || value == "false" {
		return "bool"
	}
	if strings.Contains(value, ".") {
		return "AbstractFloat"
	}
	if value != "" {
		return "AbstractInt"
	}
	return ""
}

func (e *syntaxExtractor) structures() []Structure {
	var result []Structure
	for _, node := range descendantsOfKind(e.root, "struct_decl") {
		line := e.line(node)
		shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
		var fields []NamedType
		for _, member := range descendantsOfKind(node, "struct_member") {
			if childOfKind(member, "member_ident").Valid() {
				fields = append(fields, e.namedType(member, annotationsForDeclaration(member)))
			}
		}
		name := childOfKind(node, "ident")
		if !name.Valid() {
			name = node.Field("name")
		}
		result = append(result, Structure{Name: e.text(name), Fields: fields, LineNumber: line, Comment: strings.Join(getItemComments(line, e.lineComments), "\n"), ShaderDefs: shaderDefs})
	}
	return result
}

func (e *syntaxExtractor) functions() []Function {
	var result []Function
	for _, node := range descendantsOfKind(e.root, "function_decl") {
		line := e.line(node)
		shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
		attrs := annotationsForDeclaration(node)
		stage, workgroup := functionAttributes(attrs)
		header := childOfKind(node, "function_header")
		var params []NamedType
		if list := childOfKind(header, "param_list"); list.Valid() {
			for _, parameter := range list.Children() {
				if parameter.Kind() == "param" {
					params = append(params, e.namedType(parameter, annotationsFrom(parameter)))
				}
			}
		}
		returnType := TypeInfo{Type: "void"}
		if declared := returnTypeNode(header); declared.Valid() {
			returnType.Annotations = annotationsFrom(header)
			returnType.Type = utils.RemovePath(e.text(declared))
			returnType.FullTypePath = e.text(declared)
		}
		name := childOfKind(header, "ident")
		if !name.Valid() {
			name = header.Field("name")
		}
		result = append(result, Function{StageAttribute: stage, WorkgroupSize: workgroup, Name: e.text(name), LineNumber: line, Params: params, ReturnTypeInfo: returnType, ShaderDefs: shaderDefs, Comment: strings.Join(getItemComments(line, e.lineComments), "\n")})
	}
	return result
}

func functionAttributes(attrs []Annotation) (string, []string) {
	var stage string
	var workgroup []string
	for _, attr := range attrs {
		switch attr.Name {
		case "vertex", "fragment", "compute":
			stage = attr.Name
		case "workgroup_size":
			for _, value := range strings.Split(attr.Value, ",") {
				workgroup = append(workgroup, strings.TrimSpace(value))
			}
		}
	}
	return stage, workgroup
}

func (e *syntaxExtractor) bindings() []Binding {
	var result []Binding
	for _, node := range descendantsOfKind(e.root, "global_variable_decl") {
		attrs := annotationsForDeclaration(node)
		if !hasBindingAttributes(attrs) {
			continue
		}
		declaration := childOfKind(node, "variable_decl")
		if !declaration.Valid() {
			continue
		}
		identifier := childOfKind(declaration, "optionally_typed_ident")
		if !identifier.Valid() {
			identifier = declaration
		}
		line := e.line(node)
		shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
		bindingType := ""
		if qualifier := childOfKind(declaration, "template_list"); qualifier.Valid() {
			bindingType = strings.Trim(strings.TrimSpace(e.text(qualifier)), "<>")
		}
		item := e.namedType(identifier, nil)
		result = append(result, Binding{LineNumber: line, Name: item.Name, BindingType: bindingType, Annotations: attrs, TypeInfo: item.TypeInfo, ShaderDefs: shaderDefs})
	}
	return result
}

func returnTypeNode(header syntax.Node) syntax.Node {
	if declared := header.Field("return_type"); declared.Valid() {
		return declared
	}
	children := header.Children()
	for _, child := range children {
		if child.Kind() == "template_elaborated_ident" || child.Kind() == "type_specifier" {
			return child
		}
	}
	return syntax.Node{}
}

func hasBindingAttributes(attrs []Annotation) bool {
	group, binding := false, false
	for _, attr := range attrs {
		group = group || attr.Name == "group"
		binding = binding || attr.Name == "binding"
	}
	return group && binding
}
