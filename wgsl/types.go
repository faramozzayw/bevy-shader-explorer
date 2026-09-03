package wgsl

import "main/wgsl/document"

type DefResult = document.DefResult
type Annotation = document.Annotation
type TypeInfo = document.TypeInfo
type NamedType = document.NamedType
type Const = document.Const
type Structure = document.Structure
type Function = document.Function
type Binding = document.Binding

type DeclaredImports = map[string][]string

type WgslFile struct {
	ProjectName           string          `json:"projectName"`
	ProjectDescription    string          `json:"projectDescription"`
	ProjectVersion        string          `json:"projectVersion"`
	ProjectURLPrefix      string          `json:"projectURLPrefix"`
	PackageURLPrefix      string          `json:"packageURLPrefix"`
	ProjectShaderCount    int             `json:"projectShaderCount"`
	DependencyShaderCount int             `json:"dependencyShaderCount"`
	ProjectCount          int             `json:"projectCount"`
	DependencyCount       int             `json:"dependencyCount"`
	Version               string          `json:"version"`
	ImportPath            *string         `json:"importPath"`
	WgslPath              string          `json:"wgslFile"`
	Consts                []Const         `json:"consts"`
	ConstsShaderDefs      bool            `json:"constsShaderDefs"`
	NotEmptyConsts        bool            `json:"notEmptyConsts"`
	Bindings              []Binding       `json:"bindings"`
	BindingsShaderDefs    bool            `json:"bindingsShaderDefs"`
	NotEmptyBindings      bool            `json:"notEmptyBindings"`
	Functions             []Function      `json:"functions"`
	NotEmptyFunctions     bool            `json:"notEmptyFunctions"`
	Structures            []Structure     `json:"structures"`
	StructuresShaderDefs  bool            `json:"structuresShaderDefs"`
	NotEmptyStructures    bool            `json:"notEmptyStructures"`
	DeclaredImports       DeclaredImports `json:"declaredImports"`
	Filename              string          `json:"filename"`
	GithubLink            string          `json:"githubLink"`
	Link                  string          `json:"link"`
	Dependency            bool            `json:"dependency"`
}
