package extract

import (
	"strings"

	utils "main/utils"
	"main/wgsl/document"
	"main/wgsl/syntax"
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
	DefName   string
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

func Parse(code, parserCode string, comments map[int]string, shaderDefs []ShaderDefBlock) (Result, error) {
	extractor, err := newSyntaxExtractor(code, parserCode, comments, shaderDefs)
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
			result = append(result, DefResult{DefName: def.DefName, Branch: "if", LineNumber: def.IfdefLine})
		}
		if def.ElseLine != nil && line > *def.ElseLine && line < def.EndifLine {
			result = append(result, DefResult{DefName: def.DefName, Branch: "else", LineNumber: *def.ElseLine})
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
	lineComments map[int]string
	shaderDefs   []ShaderDefBlock
}

func newSyntaxExtractor(code, parserCode string, lineComments map[int]string, shaderDefs []ShaderDefBlock) (*syntaxExtractor, error) {
	tree, err := syntax.Parse(code, parserCode)
	if err != nil {
		return nil, err
	}
	return &syntaxExtractor{root: tree.Root(), tree: tree, lineComments: lineComments, shaderDefs: shaderDefs}, nil
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

func childOfKind(node syntax.Node, kind string) syntax.Node {
	for _, child := range node.Children() {
		if child.Kind() == kind {
			return child
		}
	}
	return syntax.Node{}
}

func descendantsOfKind(node syntax.Node, kind string) []syntax.Node {
	return node.Descendants(kind)
}

func annotationsFrom(node syntax.Node) []Annotation {
	var annotations []Annotation
	for _, child := range node.Children() {
		if child.Kind() != "attribute" && !strings.HasSuffix(child.Kind(), "_attr") {
			continue
		}
		text := strings.TrimPrefix(child.Text(), "@")
		name, value, hasValue := strings.Cut(text, "(")
		if hasValue {
			value = strings.TrimSuffix(value, ")")
		}
		annotations = append(annotations, Annotation{Name: strings.TrimSpace(name), Value: strings.TrimSpace(value)})
	}
	return annotations
}

func (e *syntaxExtractor) namedType(node syntax.Node, annotations []Annotation) NamedType {
	line := e.line(node)
	shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
	name := childOfKind(node, "ident")
	if !name.Valid() {
		name = childOfKind(node, "member_ident")
	}
	typ := childOfKind(node, "type_specifier")
	fullType := e.text(typ)
	return NamedType{
		Annotations:   annotations,
		Name:          e.text(name),
		HasShaderDefs: len(shaderDefs) > 0,
		ShaderDefs:    shaderDefs,
		TypeInfo:      TypeInfo{Type: utils.RemovePath(fullType), FullTypePath: fullType},
	}
}

func (e *syntaxExtractor) consts() []Const {
	var result []Const
	for _, node := range descendantsOfKind(e.root, "global_value_decl") {
		declaration := childOfKind(node, "optionally_typed_ident")
		if !declaration.Valid() {
			continue
		}
		line := e.line(node)
		shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
		value := e.text(node)
		if _, after, found := strings.Cut(value, "="); found {
			value = strings.TrimSpace(strings.TrimSuffix(after, ";"))
		}
		item := e.namedType(declaration, annotationsFrom(node))
		typ := item.TypeInfo.Type
		if typ == "" {
			typ = inferConstType(value)
		}
		result = append(result, Const{LineNumber: line, Name: item.Name, Value: value, HasShaderDefs: len(shaderDefs) > 0, ShaderDefs: shaderDefs, TypeInfo: TypeInfo{Type: typ, FullTypePath: item.TypeInfo.FullTypePath}})
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
				fields = append(fields, e.namedType(member, annotationsFrom(member)))
			}
		}
		fieldsShaderDefs := false
		for _, field := range fields {
			fieldsShaderDefs = fieldsShaderDefs || field.HasShaderDefs
		}
		result = append(result, Structure{Name: e.text(childOfKind(node, "ident")), Fields: fields, LineNumber: line, Comment: strings.Join(getItemComments(line, e.lineComments), "\n"), HasShaderDefs: len(shaderDefs) > 0, ShaderDefs: shaderDefs, HasFields: len(fields) > 0, FieldsShaderDefs: fieldsShaderDefs})
	}
	return result
}

func (e *syntaxExtractor) functions() []Function {
	var result []Function
	for _, node := range descendantsOfKind(e.root, "function_decl") {
		line := e.line(node)
		shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
		attrs := annotationsFrom(node)
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
		result = append(result, Function{StageAttribute: stage, WorkgroupSize: workgroup, HasWorkgroupSize: len(workgroup) > 0, Name: e.text(childOfKind(header, "ident")), LineNumber: line, Params: params, ReturnTypeInfo: returnType, HasShaderDefs: len(shaderDefs) > 0, ShaderDefs: shaderDefs, Comment: strings.Join(getItemComments(line, e.lineComments), "\n"), HasParams: len(params) > 0})
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
		attrs := annotationsFrom(node)
		if !hasBindingAttributes(attrs) {
			continue
		}
		declaration := childOfKind(node, "variable_decl")
		if !declaration.Valid() {
			continue
		}
		identifier := childOfKind(declaration, "optionally_typed_ident")
		if !identifier.Valid() {
			continue
		}
		line := e.line(node)
		shaderDefs := getShaderDefsByLine(e.shaderDefs, line)
		bindingType := ""
		if qualifier := childOfKind(declaration, "template_list"); qualifier.Valid() {
			bindingType = strings.Trim(strings.TrimSpace(e.text(qualifier)), "<>")
		}
		item := e.namedType(identifier, nil)
		result = append(result, Binding{LineNumber: line, Name: item.Name, BindingType: bindingType, Annotations: attrs, TypeInfo: item.TypeInfo, HasShaderDefs: len(shaderDefs) > 0, ShaderDefs: shaderDefs})
	}
	return result
}

func returnTypeNode(header syntax.Node) syntax.Node {
	children := header.Children()
	for index, child := range children {
		if child.Kind() == "param_list" && index+1 < len(children) {
			for _, candidate := range children[index+1:] {
				if candidate.Kind() == "template_elaborated_ident" {
					return candidate
				}
			}
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
