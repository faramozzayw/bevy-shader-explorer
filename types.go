package main

type WgslDefResult struct {
	DefName    string `json:"defName"`
	Branch     string `json:"branch"`
	LineNumber int    `json:"lineNumber"`
}

type ShaderSearchableInfo struct {
	Link           string `json:"link"`
	Filename       string `json:"filename"`
	Exportable     bool   `json:"exportable"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	StageAttribute string `json:"stageAttribute"`
	Comment        string `json:"comment"`
}

type WgslFile struct {
	ImportPath *string `json:"importPath"`
	WgslPath   string  `json:"WgslFile"`

	Consts           []WgslConst `json:"consts"`
	ConstsShaderDefs bool        `json:"constsShaderDefs"`
	NotEmptyConsts   bool        `json:"notEmptyConsts"`

	Bindings           []WgslBinding `json:"bindings"`
	BindingsShaderDefs bool          `json:"bindingsShaderDefs"`
	NotEmptyBindings   bool          `json:"notEmptyBindings"`

	Functions         []WgslFunction `json:"functions"`
	NotEmptyFunctions bool           `json:"notEmptyFunctions"`

	Structures           []WgslStructure `json:"structures"`
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

type WgslConst struct {
	LineNumber    int             `json:"lineNumber"`
	Name          string          `json:"name"`
	TypeInfo      WgslTypeInfo    `json:"typeInfo"`
	Value         string          `json:"value"`
	HasShaderDefs bool            `json:"hasShaderDefs"`
	ShaderDefs    []WgslDefResult `json:"shaderDefs"`
}

type WgslStructure struct {
	Name             string          `json:"name"`
	Fields           []WgslNamedType `json:"fields"`
	LineNumber       int             `json:"lineNumber"`
	Comment          string          `json:"comment"`
	HasShaderDefs    bool            `json:"hasShaderDefs"`
	ShaderDefs       []WgslDefResult `json:"shaderDefs"`
	HasFields        bool            `json:"hasFields"`
	FieldsShaderDefs bool            `json:"fieldsShaderDefs"`
}

// field or param
type WgslNamedType struct {
	Annotations   []WgslAnnotation `json:"annotations"`
	Name          string           `json:"name"`
	TypeInfo      WgslTypeInfo     `json:"typeInfo"`
	HasShaderDefs bool             `json:"hasShaderDefs"`
	ShaderDefs    []WgslDefResult  `json:"shaderDefs"`
}

type WgslFunction struct {
	StageAttribute string          `json:"stageAttribute"`
	Name           string          `json:"name"`
	LineNumber     int             `json:"lineNumber"`
	Params         []WgslNamedType `json:"params"`
	ReturnTypeInfo WgslTypeInfo    `json:"returnTypeInfo"`
	HasShaderDefs  bool            `json:"hasShaderDefs"`
	ShaderDefs     []WgslDefResult `json:"shaderDefs"`
	Comment        string          `json:"comment"`
	HasParams      bool            `json:"hasParams"`
}

type WgslBinding struct {
	LineNumber    int              `json:"lineNumber"`
	Name          string           `json:"name"`
	BindingType   string           `json:"bindingType"`
	Annotations   []WgslAnnotation `json:"annotations"`
	TypeInfo      WgslTypeInfo     `json:"typeInfo"`
	HasShaderDefs bool             `json:"hasShaderDefs"`
	ShaderDefs    []WgslDefResult  `json:"shaderDefs"`
}

type WgslTypeInfo struct {
	Annotations   []WgslAnnotation `json:"annotations"`
	Type          string           `json:"type"`
	FullTypePath  string           `json:"fullTypePath"`
	TypeLink      string           `json:"typeLink"`
	TypeLinkBlank bool             `json:"typeLinkBlank"`
}

type WgslAnnotation struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
