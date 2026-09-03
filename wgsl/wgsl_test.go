package wgsl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"main/wgsl/bevy"
	"main/wgsl/extract"
)

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
