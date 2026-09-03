// Package document contains the parser-independent data rendered by the site.
package document

type DefResult struct {
	DefName    string `json:"defName"`
	Branch     string `json:"branch"`
	LineNumber int    `json:"lineNumber"`
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
}
type NamedType struct {
	Annotations   []Annotation `json:"annotations"`
	Name          string       `json:"name"`
	TypeInfo      TypeInfo     `json:"typeInfo"`
	HasShaderDefs bool         `json:"hasShaderDefs"`
	ShaderDefs    []DefResult  `json:"shaderDefs"`
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
