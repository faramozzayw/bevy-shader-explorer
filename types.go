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

type WgslFileItems struct {
	ImportPath *string         `json:"importPath"`
	Consts     []WgslConst     `json:"consts"`
	Functions  []WgslFunction  `json:"functions"`
	Structures []WgslStructure `json:"structures"`
	Bindings   []WgslBinding   `json:"bindings"`
}

type WgslFile struct {
	ImportPath *string `json:"importPath"`

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
	Type          string          `json:"type"`
	Value         string          `json:"value"`
	HasShaderDefs bool            `json:"hasShaderDefs"`
	ShaderDefs    []WgslDefResult `json:"shaderDefs"`
	TypeLink      string          `json:"typeLink"`
}

type WgslStructure struct {
	Name             string          `json:"name"`
	Fields           []WgslType      `json:"fields"`
	LineNumber       int             `json:"lineNumber"`
	Comment          string          `json:"comment"`
	HasShaderDefs    bool            `json:"hasShaderDefs"`
	ShaderDefs       []WgslDefResult `json:"shaderDefs"`
	HasAnnotations   bool            `json:"hasAnnotations"`
	HasFields        bool            `json:"hasFields"`
	FieldsShaderDefs bool            `json:"fieldsShaderDefs"`
}

type WgslType struct {
	Annotation    string          `json:"annotation"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	HasShaderDefs bool            `json:"hasShaderDefs"`
	ShaderDefs    []WgslDefResult `json:"shaderDefs"`
	TypeLink      string          `json:"typeLink"`
}

type WgslFunction struct {
	StageAttribute string          `json:"stageAttribute"`
	Name           string          `json:"name"`
	LineNumber     int             `json:"lineNumber"`
	Params         []WgslType      `json:"params"`
	ReturnType     string          `json:"returnType"`
	ReturnTypeLink string          `json:"returnTypeLink"`
	HasShaderDefs  bool            `json:"hasShaderDefs"`
	ShaderDefs     []WgslDefResult `json:"shaderDefs"`
	Comment        string          `json:"comment"`
	HasParams      bool            `json:"hasParams"`
}

type WgslBinding struct {
	LineNumber    int             `json:"lineNumber"`
	Name          string          `json:"name"`
	GroupIndex    string          `json:"groupIndex"`
	BindingIndex  string          `json:"bindingIndex"`
	BindingType   string          `json:"bindingType"` // optional
	Type          string          `json:"type"`
	HasShaderDefs bool            `json:"hasShaderDefs"`
	ShaderDefs    []WgslDefResult `json:"shaderDefs"`
	TypeLink      string          `json:"typeLink"`
}
