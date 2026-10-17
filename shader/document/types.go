// Package document contains the parser-independent data rendered by the site.
package document

import "strings"

type Source struct {
	Path string
	Text string
}

func NewSource(path, text string) Source {
	return Source{Path: path, Text: strings.ReplaceAll(text, "\r\n", "\n")}
}

// ShaderCondition is the normalized form of a shader-definition expression.
// Expression is retained for display/debugging; Clauses are the canonical
// boolean representation used for evaluation, and Atoms drive controls/pills.
type ShaderCondition struct {
	Expression string         `json:"expression"`
	Atoms      []string       `json:"atoms"`
	Clauses    [][]ShaderTerm `json:"clauses"`
}

type ShaderTerm struct {
	Name    string `json:"name"`
	Negated bool   `json:"negated,omitempty"`
}

// File is the parser-owned shader document. Site metadata is kept outside
// this type so parsing remains independent from rendering and deployment.
type File struct {
	ImportPath      *string          `json:"importPath"`
	Consts          []Const          `json:"consts"`
	Bindings        []Binding        `json:"bindings"`
	Functions       []Function       `json:"functions"`
	Structures      []Structure      `json:"structures"`
	DeclaredImports DeclaredImports  `json:"declaredImports"`
	ResolvedImports []ImportRef      `json:"resolvedImports,omitempty"`
	Filename        string           `json:"filename"`
	ShaderDefs      []string         `json:"shaderDefs"`
	ShaderDefBlocks []ShaderDefBlock `json:"shaderDefBlocks"`
}

type DeclaredImports = map[string][]string

type ImportRef struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Link string `json:"link,omitempty"`
}

type UsageRef struct {
	Link           string `json:"link"`
	PackageName    string `json:"packageName"`
	PackageVersion string `json:"packageVersion"`
	Filename       string `json:"filename"`
}

func NewShaderCondition(expression string) ShaderCondition {
	condition := ShaderCondition{Expression: strings.TrimSpace(expression)}
	seen := make(map[string]bool)
	for _, alternative := range strings.Split(expression, "||") {
		var clause []ShaderTerm
		for _, termText := range strings.Split(alternative, "&&") {
			termText = strings.TrimSpace(strings.Trim(termText, "()"))
			negated := strings.HasPrefix(termText, "!")
			name := strings.TrimSpace(strings.TrimPrefix(termText, "!"))
			name = strings.Trim(name, "() ")
			if name == "" {
				continue
			}
			term := ShaderTerm{Name: name, Negated: negated}
			clause = append(clause, term)
			if !seen[name] {
				seen[name] = true
				condition.Atoms = append(condition.Atoms, name)
			}
		}
		if len(clause) > 0 {
			condition.Clauses = append(condition.Clauses, clause)
		}
	}
	return condition
}

type DefResult struct {
	Condition  ShaderCondition `json:"condition"`
	Branch     string          `json:"branch"`
	LineNumber int             `json:"lineNumber"`
}

// ShaderDefBlock preserves a top-level shader-definition source range.
type ShaderDefBlock struct {
	Condition ShaderCondition `json:"condition"`
	Negated   bool            `json:"negated,omitempty"`
	IfLine    int             `json:"ifLine"`
	ElseLine  *int            `json:"elseLine,omitempty"`
	EndifLine int             `json:"endifLine"`
}

type Annotation struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type TypeInfo struct {
	Annotations   []Annotation `json:"annotations"`
	Type          string       `json:"type"`
	FullTypePath  string       `json:"fullTypePath"`
	TypeLink      string       `json:"typeLink"`
	TypeLinkBlank bool         `json:"typeLinkBlank"`
	TypeParts     []TypePart   `json:"typeParts,omitempty"`
}

// TypePart preserves a type expression while allowing each nested type name
// to link to its documentation independently.
type TypePart struct {
	Text          string `json:"text"`
	TypeLink      string `json:"typeLink,omitempty"`
	TypeLinkBlank bool   `json:"typeLinkBlank,omitempty"`
}
type NamedType struct {
	Annotations []Annotation `json:"annotations"`
	Name        string       `json:"name"`
	TypeInfo    TypeInfo     `json:"typeInfo"`
	ShaderDefs  []DefResult  `json:"shaderDefs"`
}
type Const struct {
	LineNumber    int         `json:"lineNumber"`
	Name          string      `json:"name"`
	TypeInfo      TypeInfo    `json:"typeInfo"`
	Value         string      `json:"value"`
	ShaderDefs    []DefResult `json:"shaderDefs"`
	UsedBy        []UsageRef  `json:"usedBy,omitempty"`
	UsedBySummary string      `json:"usedBySummary,omitempty"`
}
type Structure struct {
	Name          string      `json:"name"`
	Fields        []NamedType `json:"fields"`
	LineNumber    int         `json:"lineNumber"`
	Comment       string      `json:"comment"`
	ShaderDefs    []DefResult `json:"shaderDefs"`
	UsedBy        []UsageRef  `json:"usedBy,omitempty"`
	UsedBySummary string      `json:"usedBySummary,omitempty"`
}
type Function struct {
	StageAttribute string      `json:"stageAttribute"`
	WorkgroupSize  []string    `json:"workgroupSize"`
	Name           string      `json:"name"`
	LineNumber     int         `json:"lineNumber"`
	Params         []NamedType `json:"params"`
	ReturnTypeInfo TypeInfo    `json:"returnTypeInfo"`
	ShaderDefs     []DefResult `json:"shaderDefs"`
	Comment        string      `json:"comment"`
	UsedBy         []UsageRef  `json:"usedBy,omitempty"`
	UsedBySummary  string      `json:"usedBySummary,omitempty"`
}
type Binding struct {
	LineNumber    int          `json:"lineNumber"`
	Name          string       `json:"name"`
	BindingType   string       `json:"bindingType"`
	Annotations   []Annotation `json:"annotations"`
	TypeInfo      TypeInfo     `json:"typeInfo"`
	ShaderDefs    []DefResult  `json:"shaderDefs"`
	UsedBy        []UsageRef   `json:"usedBy,omitempty"`
	UsedBySummary string       `json:"usedBySummary,omitempty"`
}
