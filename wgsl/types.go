package wgsl

type DefResult struct {
	DefName    string `json:"defName"`
	Branch     string `json:"branch"`
	LineNumber int    `json:"lineNumber"`
}

type WgslFile struct {
	Version    string  `json:"version"`
	ImportPath *string `json:"importPath"`
	WgslPath   string  `json:"wgslFile"`

	Consts           []Const `json:"consts"`
	ConstsShaderDefs bool    `json:"constsShaderDefs"`
	NotEmptyConsts   bool    `json:"notEmptyConsts"`

	Bindings           []Binding `json:"bindings"`
	BindingsShaderDefs bool      `json:"bindingsShaderDefs"`
	NotEmptyBindings   bool      `json:"notEmptyBindings"`

	Functions         []Function `json:"functions"`
	NotEmptyFunctions bool       `json:"notEmptyFunctions"`

	Structures           []Structure     `json:"structures"`
	StructuresShaderDefs bool            `json:"structuresShaderDefs"`
	NotEmptyStructures   bool            `json:"notEmptyStructures"`
	DeclaredImports      DeclaredImports `json:"declaredImports"`

	Filename   string `json:"filename"`
	GithubLink string `json:"githubLink"`
	Link       string `json:"link"`
}

type ShaderDefBlock struct {
	DefName   string `json:"defName"`
	IfdefLine int    `json:"ifdefLine"`
	ElseLine  *int   `json:"elseLine,omitempty"`
	EndifLine int    `json:"endifLine"`
}

type Const struct {
	LineNumber    int         `json:"lineNumber"`
	Name          string      `json:"name"`
	TypeInfo      TypeInfo    `json:"typeInfo"`
	Value         string      `json:"value"`
	HasShaderDefs bool        `json:"hasShaderDefs"`
	ShaderDefs    []DefResult `json:"shaderDefs"`
}

type Structure struct {
	Name             string      `json:"name"`
	Fields           []NamedType `json:"fields"`
	LineNumber       int         `json:"lineNumber"`
	Comment          string      `json:"comment"`
	HasShaderDefs    bool        `json:"hasShaderDefs"`
	ShaderDefs       []DefResult `json:"shaderDefs"`
	HasFields        bool        `json:"hasFields"`
	FieldsShaderDefs bool        `json:"fieldsShaderDefs"`
}

// field or param
type NamedType struct {
	Annotations   []Annotation `json:"annotations"`
	Name          string       `json:"name"`
	TypeInfo      TypeInfo     `json:"typeInfo"`
	HasShaderDefs bool         `json:"hasShaderDefs"`
	ShaderDefs    []DefResult  `json:"shaderDefs"`
}

type Function struct {
	StageAttribute   string      `json:"stageAttribute"`
	WorkgroupSize    []string    `json:"workgroupSize"`
	HasWorkgroupSize bool        `json:"hasWorkgroupSize"`
	Name             string      `json:"name"`
	LineNumber       int         `json:"lineNumber"`
	Params           []NamedType `json:"params"`
	ReturnTypeInfo   TypeInfo    `json:"returnTypeInfo"`
	HasShaderDefs    bool        `json:"hasShaderDefs"`
	ShaderDefs       []DefResult `json:"shaderDefs"`
	Comment          string      `json:"comment"`
	HasParams        bool        `json:"hasParams"`
}

type Binding struct {
	LineNumber    int          `json:"lineNumber"`
	Name          string       `json:"name"`
	BindingType   string       `json:"bindingType"`
	Annotations   []Annotation `json:"annotations"`
	TypeInfo      TypeInfo     `json:"typeInfo"`
	HasShaderDefs bool         `json:"hasShaderDefs"`
	ShaderDefs    []DefResult  `json:"shaderDefs"`
}

type TypeInfo struct {
	Annotations   []Annotation `json:"annotations"`
	Type          string       `json:"type"`
	FullTypePath  string       `json:"fullTypePath"`
	TypeLink      string       `json:"typeLink"`
	TypeLinkBlank bool         `json:"typeLinkBlank"`
}

type Annotation struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
