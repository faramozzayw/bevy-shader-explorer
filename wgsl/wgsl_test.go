package wgsl

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"main/config"
	"main/wgsl/bevy"
	"main/wgsl/extract"
)

func TestGetGithubLinkUsesConfiguredSourceRef(t *testing.T) {
	cfg := config.Config{SourcePath: "/project", SourceGithubURL: "https://github.com/bevyengine/bevy", SourceGithubRef: "release-0.19.1"}
	got := GetGithubLink(&cfg, "/project/assets/shaders", "extended_material_bindless.wgsl")
	assert.Equal(t, "https://github.com/bevyengine/bevy/blob/release-0.19.1/assets/shaders/extended_material_bindless.wgsl", got)
}

func TestGetGithubLinkUsesRepositoryRootForWorkspaceMembers(t *testing.T) {
	cfg := config.Config{
		SourcePath:       "/project/crates/bevy_pbr",
		SourceGithubRoot: "/project",
		SourceGithubURL:  "https://github.com/bevyengine/bevy",
		SourceGithubRef:  "release-0.19.1",
	}
	got := GetGithubLink(&cfg, "/project/crates/bevy_pbr/src/light_probe", "environment_filter.wgsl")
	assert.Equal(t, "https://github.com/bevyengine/bevy/blob/release-0.19.1/crates/bevy_pbr/src/light_probe/environment_filter.wgsl", got)
}

func TestGetGithubLinkUsesRepositorySubpathForNestedCrate(t *testing.T) {
	cfg := config.Config{
		SourcePath:          "/registry/wgpu-29.0.4",
		SourceGithubRoot:    "/registry/wgpu-29.0.4",
		SourceGithubSubpath: "wgpu",
		SourceGithubURL:     "https://github.com/gfx-rs/wgpu",
		SourceGithubRef:     "v29",
	}
	got := GetGithubLink(&cfg, "/registry/wgpu-29.0.4/src/util", "blit.wgsl")
	assert.Equal(t, "https://github.com/gfx-rs/wgpu/blob/v29/wgpu/src/util/blit.wgsl", got)
}

//go:embed testdata/all_items.wgsl
var allItemsFixture string

//go:embed testdata/shader_definitions.wgsl
var shaderDefinitionsFixture string

//go:embed testdata/edge_cases.wgsl
var edgeCasesFixture string

func TestTreeSitterExtractionUsesDeclarationSpans(t *testing.T) {
	code := `
struct First { value: f32, }
struct Second { value: vec4<f32>, }

const COUNT: u32 = 1u;

@group(0) @binding(1)
var<uniform> settings: Second;

@compute @workgroup_size(8, 4, 1)
fn main(@builtin(global_invocation_id) value: vec3<u32>) {
}
`
	declarations, err := extract.Parse(code, code, extractComments(strings.Split(code, "\n")), nil)
	if !assert.NoError(t, err) {
		return
	}

	structures := declarations.Structures
	assert.Len(t, structures, 2)
	assert.Equal(t, "vec4<f32>", structures[1].Fields[0].TypeInfo.Type)
	assert.Equal(t, 3, structures[1].LineNumber)

	consts := declarations.Consts
	if assert.Len(t, consts, 1) {
		assert.Equal(t, 5, consts[0].LineNumber)
	}

	bindings := declarations.Bindings
	if assert.Len(t, bindings, 1) {
		assert.Equal(t, "uniform", bindings[0].BindingType)
		assert.Equal(t, "Second", bindings[0].TypeInfo.Type)
	}

	functions := declarations.Functions
	if assert.Len(t, functions, 1) {
		assert.Equal(t, []string{"8", "4", "1"}, functions[0].WorkgroupSize)
		assert.Equal(t, "vec3<u32>", functions[0].Params[0].TypeInfo.Type)
	}
}

func TestConstExtraction(t *testing.T) {
	code := `
const COLOR_MATERIAL_FLAGS_TEXTURE_BIT: u32              = 1u;
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_RESERVED_BITS: u32 = 3221225472u; // (0b11u32 << 30)
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_OPAQUE: u32        = 0u;          // (0u32 << 30)
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_MASK: u32          = 1073741824u; // (1u32 << 30)
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_BLEND: u32         = 2147483648u; // (2u32 << 30)
  `

	declarations, err := extract.Parse(code, code, map[int]string{}, nil)
	if !assert.NoError(t, err) {
		return
	}
	consts := declarations.Consts

	expectedConsts := []Const{
		{
			LineNumber: 2,
			Name:       "COLOR_MATERIAL_FLAGS_TEXTURE_BIT",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "1u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 3,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_RESERVED_BITS",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "3221225472u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 4,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_OPAQUE",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "0u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 5,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_MASK",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "1073741824u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 6,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_BLEND",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "2147483648u",
			HasShaderDefs: false,
		},
	}

	for i := range expectedConsts {
		expectedConsts[i].TypeInfo.FullTypePath = "u32"
	}
	for i := range consts {
		assert.Equal(t, expectedConsts[i], consts[i])
	}
}

func TestOfficialGrammarParsesBevyDirectivesAfterMasking(t *testing.T) {
	code := `#import bevy_pbr::{
    mesh_functions,
}
#ifdef PREPASS
@group(0) @binding(0)
var<uniform> settings: Settings;
#else
const VALUE: u32 = 1u;
#endif
`

	declarations, err := extract.Parse(code, bevy.MaskDirectives(code), map[int]string{}, nil)
	if !assert.NoError(t, err) {
		return
	}
	if assert.Len(t, declarations.Bindings, 1) {
		assert.Equal(t, 5, declarations.Bindings[0].LineNumber)
		assert.Equal(t, "settings", declarations.Bindings[0].Name)
	}
	if assert.Len(t, declarations.Consts, 1) {
		assert.Equal(t, 8, declarations.Consts[0].LineNumber)
		assert.Equal(t, "VALUE", declarations.Consts[0].Name)
	}
}

func TestWESLGrammarParsesWGSLDeclarationsWithExtensions(t *testing.T) {
	code := `import package::colors::chartreuse;

@if(DEBUG)
const DEBUG_COLOR: vec3f = chartreuse;

struct Vertex {
    position: vec3f,
}

fn shade(value: vec3f) -> vec3f {
    return value;
}`

	declarations, err := extract.ParseWESL(code, code, map[int]string{}, nil)
	if !assert.NoError(t, err) || !assert.Len(t, declarations.Consts, 1) || !assert.Len(t, declarations.Structures, 1) || !assert.Len(t, declarations.Functions, 1) {
		return
	}
	assert.Equal(t, "DEBUG_COLOR", declarations.Consts[0].Name)
	assert.Equal(t, "Vertex", declarations.Structures[0].Name)
	assert.Equal(t, "shade", declarations.Functions[0].Name)
}

func TestExtractionRegressionForAllItemKinds(t *testing.T) {
	code := allItemsFixture

	declarations, err := extract.Parse(code, code, extractComments(strings.Split(code, "\n")), nil)
	if !assert.NoError(t, err) {
		return
	}

	if assert.Len(t, declarations.Structures, 1) {
		structure := declarations.Structures[0]
		assert.Equal(t, "Settings", structure.Name)
		assert.Equal(t, "Material settings used by the shader.", structure.Comment)
		if assert.Len(t, structure.Fields, 2) {
			assert.Equal(t, "tint", structure.Fields[0].Name)
			assert.Equal(t, "vec4<f32>", structure.Fields[0].TypeInfo.Type)
			assert.Equal(t, []Annotation{{Name: "align", Value: "16"}}, structure.Fields[0].Annotations)
			assert.Equal(t, "enabled", structure.Fields[1].Name)
			assert.Equal(t, "u32", structure.Fields[1].TypeInfo.Type)
		}
	}

	if assert.Len(t, declarations.Consts, 3) {
		assert.Equal(t, "SCALE", declarations.Consts[0].Name)
		assert.Equal(t, "f32", declarations.Consts[0].TypeInfo.Type)
		assert.Equal(t, "2.0", declarations.Consts[0].Value)
		assert.Equal(t, "MAX_LIGHTS", declarations.Consts[1].Name)
		assert.Equal(t, "u32", declarations.Consts[1].TypeInfo.Type)
		assert.Equal(t, "QUALITY", declarations.Consts[2].Name)
	}

	if assert.Len(t, declarations.Bindings, 1) {
		binding := declarations.Bindings[0]
		assert.Equal(t, "settings", binding.Name)
		assert.Equal(t, "storage, read_write", binding.BindingType)
		assert.Equal(t, "Settings", binding.TypeInfo.Type)
		assert.Equal(t, []Annotation{{Name: "group", Value: "1"}, {Name: "binding", Value: "2"}}, binding.Annotations)
	}

	if assert.Len(t, declarations.Functions, 1) {
		function := declarations.Functions[0]
		assert.Equal(t, "shade", function.Name)
		assert.Equal(t, "compute", function.StageAttribute)
		assert.Equal(t, []string{"8", "4", "1"}, function.WorkgroupSize)
		assert.Equal(t, "Executes the material pass.", function.Comment)
		assert.Equal(t, "vec4<f32>", function.ReturnTypeInfo.Type)
		assert.Equal(t, []Annotation{{Name: "location", Value: "0"}}, function.ReturnTypeInfo.Annotations)
		if assert.Len(t, function.Params, 2) {
			assert.Equal(t, "invocation_id", function.Params[0].Name)
			assert.Equal(t, "vec3<u32>", function.Params[0].TypeInfo.Type)
			assert.Equal(t, []Annotation{{Name: "builtin", Value: "global_invocation_id"}}, function.Params[0].Annotations)
			assert.Equal(t, "factor", function.Params[1].Name)
			assert.Equal(t, "f32", function.Params[1].TypeInfo.Type)
		}
	}
}

func TestExtractionRegressionForShaderDefinitionBranches(t *testing.T) {
	code := shaderDefinitionsFixture

	definitions := bevy.DefinitionBlocks(code)
	shaderDefs := make([]extract.ShaderDefBlock, 0, len(definitions))
	for _, definition := range definitions {
		shaderDefs = append(shaderDefs, extract.ShaderDefBlock{DefName: definition.Name, IfdefLine: definition.IfLine, ElseLine: definition.ElseLine, EndifLine: definition.EndifLine})
	}

	declarations, err := extract.Parse(code, bevy.MaskDirectives(code), map[int]string{}, shaderDefs)
	if !assert.NoError(t, err) {
		return
	}

	if assert.Len(t, declarations.Consts, 1) {
		assert.Equal(t, []DefResult{{DefName: "FEATURE", Branch: "if", LineNumber: 1}}, declarations.Consts[0].ShaderDefs)
	}
	if assert.Len(t, declarations.Structures, 1) {
		assert.Equal(t, []DefResult{{DefName: "FEATURE", Branch: "if", LineNumber: 1}}, declarations.Structures[0].ShaderDefs)
		assert.Equal(t, []DefResult{{DefName: "FEATURE", Branch: "if", LineNumber: 1}}, declarations.Structures[0].Fields[0].ShaderDefs)
	}
	if assert.Len(t, declarations.Bindings, 1) {
		assert.Equal(t, []DefResult{{DefName: "FEATURE", Branch: "else", LineNumber: 6}}, declarations.Bindings[0].ShaderDefs)
	}
}

func TestExtractionRegressionForSyntaxVariants(t *testing.T) {
	code := edgeCasesFixture
	declarations, err := extract.Parse(code, code, extractComments(strings.Split(code, "\n")), nil)
	if !assert.NoError(t, err) {
		return
	}

	if assert.Len(t, declarations.Consts, 7) {
		assert.Equal(t, "INFERRED_INT", declarations.Consts[0].Name)
		assert.Equal(t, "AbstractInt", declarations.Consts[0].TypeInfo.Type)
		assert.Equal(t, "INFERRED_SIGNED", declarations.Consts[1].Name)
		assert.Equal(t, "AbstractInt", declarations.Consts[1].TypeInfo.Type)
		assert.Equal(t, "INFERRED_BOOL", declarations.Consts[2].Name)
		assert.Equal(t, "bool", declarations.Consts[2].TypeInfo.Type)
		assert.Equal(t, "HEX_VALUE", declarations.Consts[3].Name)
		assert.Equal(t, "u32", declarations.Consts[3].TypeInfo.Type)
		assert.Equal(t, "INFERRED_FLOAT", declarations.Consts[4].Name)
		assert.Equal(t, "AbstractFloat", declarations.Consts[4].TypeInfo.Type)
		assert.Equal(t, "INFERRED_VECTOR", declarations.Consts[5].Name)
		assert.Equal(t, "vec3<f32>", declarations.Consts[5].TypeInfo.Type)
		assert.Equal(t, "DERIVED", declarations.Consts[6].Name)
		assert.Equal(t, "AbstractInt", declarations.Consts[6].TypeInfo.Type)
	}

	if assert.Len(t, declarations.Structures, 1) {
		assert.Equal(t, "VertexData", declarations.Structures[0].Name)
		assert.Contains(t, declarations.Structures[0].Comment, "This comment spans multiple lines.")
		assert.Contains(t, declarations.Structures[0].Comment, "Position data from a previous pass.")
	}

	if assert.Len(t, declarations.Bindings, 3) {
		assert.Equal(t, "uniform", declarations.Bindings[0].BindingType)
		assert.Equal(t, "storage, read", declarations.Bindings[1].BindingType)
		assert.Equal(t, "uniform", declarations.Bindings[2].BindingType)
		assert.Equal(t, "array<mat4x4<f32>, 4>", declarations.Bindings[2].TypeInfo.Type)
	}

	if assert.Len(t, declarations.Functions, 3) {
		assert.Equal(t, "vertex", declarations.Functions[0].StageAttribute)
		if assert.Len(t, declarations.Functions[0].Params, 1) {
			assert.Equal(t, "position", declarations.Functions[0].Params[0].Name)
			assert.Equal(t, "vec3<f32>", declarations.Functions[0].Params[0].TypeInfo.Type)
			assert.Equal(t, []Annotation{{Name: "location", Value: "0"}}, declarations.Functions[0].Params[0].Annotations)
		}
		assert.Equal(t, "vec4<f32>", declarations.Functions[0].ReturnTypeInfo.Type)
		assert.Equal(t, "fragment", declarations.Functions[1].StageAttribute)
		assert.Empty(t, declarations.Functions[1].Params)
		assert.Empty(t, declarations.Functions[2].StageAttribute)
		assert.Equal(t, "helper", declarations.Functions[2].Name)
	}
}

func TestImportExtractionRegression(t *testing.T) {
	source := `#import bevy_pbr::{
    mesh_functions,
    view::View,
}
#import bevy_render::RenderDevice as Device;
`

	imports, err := bevy.ExtractAllImports(source)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"bevy_pbr::mesh_functions"}, imports["mesh_functions"])
	assert.Equal(t, []string{"bevy_pbr::view::View"}, imports["View"])
	assert.Equal(t, []string{"bevy_render::RenderDevice"}, imports["Device"])

	_, err = bevy.ExtractAllImports("#import bevy_pbr::{mesh_functions\n")
	assert.Error(t, err)
}

func TestDefinitionExtractionRegressionForNestedBranches(t *testing.T) {
	source := `#ifdef OUTER
const outer: u32 = 1u;
#ifdef INNER
const inner: u32 = 2u;
#else
const alternate: u32 = 3u;
#endif
#endif
`

	definitions := bevy.DefinitionBlocks(source)
	if !assert.Len(t, definitions, 2) {
		return
	}
	assert.Equal(t, bevy.DefinitionBlock{Name: "OUTER", IfLine: 1, EndifLine: 8}, definitions[0])
	assert.Equal(t, bevy.DefinitionBlock{Name: "INNER", IfLine: 3, ElseLine: intPointer(5), EndifLine: 7}, definitions[1])
}

func intPointer(value int) *int { return &value }

func TestExtractionIgnoresVariablesWithoutCompleteBindingAnnotations(t *testing.T) {
	code := `@group(0)
var<uniform> only_group: Settings;
@binding(1)
var<uniform> only_binding: Settings;
var<uniform> no_annotations: Settings;
`

	declarations, err := extract.Parse(code, code, map[int]string{}, nil)
	if !assert.NoError(t, err) {
		return
	}
	assert.Empty(t, declarations.Bindings)
}

func TestNestedDefinitionsWithoutTrailingNewline(t *testing.T) {
	code := "#ifdef OUTER\nconst outer: u32 = 1u;\n#ifdef INNER\nconst inner: u32 = 2u;\n#else\nconst alternate: u32 = 3u;\n#endif\n#else\nconst fallback: u32 = 4u;\n#endif"
	definitions := bevy.DefinitionBlocks(code)
	shaderDefs := make([]extract.ShaderDefBlock, 0, len(definitions))
	for _, definition := range definitions {
		shaderDefs = append(shaderDefs, extract.ShaderDefBlock{DefName: definition.Name, IfdefLine: definition.IfLine, ElseLine: definition.ElseLine, EndifLine: definition.EndifLine})
	}

	masked := bevy.MaskDirectives(code)
	assert.Equal(t, len(code), len(masked))
	assert.Equal(t, strings.Count(code, "\n"), strings.Count(masked, "\n"))
	declarations, err := extract.Parse(code, masked, map[int]string{}, shaderDefs)
	if !assert.NoError(t, err) || !assert.Len(t, declarations.Consts, 4) {
		return
	}
	assert.Len(t, declarations.Consts[1].ShaderDefs, 2)
	assert.Equal(t, "INNER", declarations.Consts[1].ShaderDefs[1].DefName)
	assert.Equal(t, "alternate", declarations.Consts[2].Name)
	assert.Equal(t, "else", declarations.Consts[2].ShaderDefs[1].Branch)
}
