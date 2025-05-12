package wgsl

import (
	// . "main/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstExtraction(t *testing.T) {
	code := `
const COLOR_MATERIAL_FLAGS_TEXTURE_BIT: u32              = 1u;
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_RESERVED_BITS: u32 = 3221225472u; // (0b11u32 << 30)
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_OPAQUE: u32        = 0u;          // (0u32 << 30)
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_MASK: u32          = 1073741824u; // (1u32 << 30)
const COLOR_MATERIAL_FLAGS_ALPHA_MODE_BLEND: u32         = 2147483648u; // (2u32 << 30)
  `

	consts := extractConsts(code, map[int]string{}, []ShaderDefBlock{})

	expectedConsts := []Const{
		{
			LineNumber: 0,
			Name:       "COLOR_MATERIAL_FLAGS_TEXTURE_BIT",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "1u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 0,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_RESERVED_BITS",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "3221225472u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 0,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_OPAQUE",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "0u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 0,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_MASK",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "1073741824u",
			HasShaderDefs: false,
		},
		{
			LineNumber: 0,
			Name:       "COLOR_MATERIAL_FLAGS_ALPHA_MODE_BLEND",
			TypeInfo: TypeInfo{
				Type:          "u32",
				TypeLinkBlank: false,
			},
			Value:         "2147483648u",
			HasShaderDefs: false,
		},
	}

	for i := range consts {
		assert.Equal(t, expectedConsts[i], consts[i])
	}
}

func TestStructuresExtraction(t *testing.T) {
	code := `
struct BoxShadowVertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) point: vec2<f32>,
    @location(1) color: vec4<f32>,
    @location(2) @interpolate(flat) size: vec2<f32>,
    @location(3) @interpolate(flat) radius: vec4<f32>,
    @location(4) @interpolate(flat) blur: f32,
}
  `

	structure := extractStructures(code, map[int]string{}, []ShaderDefBlock{})[0]

	assert.Equal(t, Structure{
		Name: "BoxShadowVertexOutput",
		Fields: []NamedType{
			{
				Annotations: []Annotation{
					{Name: "builtin", Value: "position"},
				},
				Name: "position",
				TypeInfo: TypeInfo{
					Annotations:   nil,
					Type:          "vec4<f32>",
					FullTypePath:  "vec4<f32>",
					TypeLink:      "",
					TypeLinkBlank: false,
				},
				HasShaderDefs: false,
				ShaderDefs:    nil,
			},
			{
				Annotations: []Annotation{
					{Name: "location", Value: "0"},
				},
				Name: "point",
				TypeInfo: TypeInfo{
					Annotations:   nil,
					Type:          "vec2<f32>",
					FullTypePath:  "vec2<f32>",
					TypeLink:      "",
					TypeLinkBlank: false,
				},
				HasShaderDefs: false,
				ShaderDefs:    nil,
			},
			{
				Annotations: []Annotation{
					{Name: "location", Value: "1"},
				},
				Name: "color",
				TypeInfo: TypeInfo{
					Annotations:   nil,
					Type:          "vec4<f32>",
					FullTypePath:  "vec4<f32>",
					TypeLink:      "",
					TypeLinkBlank: false,
				},
				HasShaderDefs: false,
				ShaderDefs:    nil,
			},
			{
				Annotations: []Annotation{
					{Name: "location", Value: "2"},
					{Name: "interpolate", Value: "flat"},
				},
				Name: "size",
				TypeInfo: TypeInfo{
					Annotations:   nil,
					Type:          "vec2<f32>",
					FullTypePath:  "vec2<f32>",
					TypeLink:      "",
					TypeLinkBlank: false,
				},
				HasShaderDefs: false,
				ShaderDefs:    nil,
			},
			{
				Annotations: []Annotation{
					{Name: "location", Value: "3"},
					{Name: "interpolate", Value: "flat"},
				},
				Name: "radius",
				TypeInfo: TypeInfo{
					Annotations:   nil,
					Type:          "vec4<f32>",
					FullTypePath:  "vec4<f32>",
					TypeLink:      "",
					TypeLinkBlank: false,
				},
				HasShaderDefs: false,
				ShaderDefs:    nil,
			},
			{
				Annotations: []Annotation{
					{Name: "location", Value: "4"},
					{Name: "interpolate", Value: "flat"},
				},
				Name: "blur",
				TypeInfo: TypeInfo{
					Annotations:   nil,
					Type:          "f32",
					FullTypePath:  "f32",
					TypeLink:      "",
					TypeLinkBlank: false,
				},
				HasShaderDefs: false,
				ShaderDefs:    nil,
			},
		},
		LineNumber:       2,
		Comment:          "",
		HasShaderDefs:    false,
		ShaderDefs:       nil,
		HasFields:        true,
		FieldsShaderDefs: false,
	}, structure)
}

func TestBindingExtractions(t *testing.T) {
	code := `
@group(0) @binding(7) var<storage, read_write> exposure: f32;
@group(1) @binding(0) var<storage> material_color: binding_array<Color, 4>;
@group(1) @binding(4) var material_color_texture: binding_array<texture_2d<f32>, 4>;
@group(1) @binding(2) var material_color_sampler: binding_array<sampler, 4>;
@group(2) @binding(3) var<uniform> material_color: Color;
  `

	expectedBindings := []Binding{
		{
			LineNumber:  2,
			Name:        "exposure",
			BindingType: "storage, read_write",
			Annotations: []Annotation{
				{Name: "group", Value: "0"},
				{Name: "binding", Value: "7"},
			},
			TypeInfo: TypeInfo{
				Annotations:   nil,
				Type:          "f32",
				FullTypePath:  "f32",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
		},
		{
			LineNumber:  3,
			Name:        "material_color",
			BindingType: "storage",
			Annotations: []Annotation{
				{Name: "group", Value: "1"},
				{Name: "binding", Value: "0"},
			},
			TypeInfo: TypeInfo{
				Annotations:   nil,
				Type:          "binding_array<Color, 4>",
				FullTypePath:  "binding_array<Color, 4>",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
		},
		{
			LineNumber:  4,
			Name:        "material_color_texture",
			BindingType: "",
			Annotations: []Annotation{
				{Name: "group", Value: "1"},
				{Name: "binding", Value: "4"},
			},
			TypeInfo: TypeInfo{
				Annotations:   nil,
				Type:          "binding_array<texture_2d<f32>, 4>",
				FullTypePath:  "binding_array<texture_2d<f32>, 4>",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
		},
		{
			LineNumber:  5,
			Name:        "material_color_sampler",
			BindingType: "",
			Annotations: []Annotation{
				{Name: "group", Value: "1"},
				{Name: "binding", Value: "2"},
			},
			TypeInfo: TypeInfo{
				Annotations:   nil,
				Type:          "binding_array<sampler, 4>",
				FullTypePath:  "binding_array<sampler, 4>",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
		},
		{
			LineNumber:  6,
			Name:        "material_color",
			BindingType: "uniform",
			Annotations: []Annotation{
				{Name: "group", Value: "2"},
				{Name: "binding", Value: "3"},
			},
			TypeInfo: TypeInfo{
				Annotations:   nil,
				Type:          "Color",
				FullTypePath:  "Color",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
		},
	}

	bindings := extractBindings(code, map[int]string{}, []ShaderDefBlock{})

	for i := range bindings {
		assert.Equal(t, expectedBindings[i], bindings[i])
	}
}

func TestFunctionsExtraction(t *testing.T) {
	code := `
fn selectCorner(p: vec2<f32>, c: vec4<f32>) -> f32 {
  // stuff
}

@vertex
fn vertex(
    @location(0) vertex_position: vec3<f32>,
) -> BoxShadowVertexOutput {
  // stuff
}
 
@fragment
fn fragment(
    in: BoxShadowVertexOutput,
) -> @location(0) vec4<f32> {
  // stuff
}
@compute
@workgroup_size(256, 1, 1)
fn downsample_depth_first(
    @builtin(num_workgroups) num_workgroups: vec3u,
    @builtin(workgroup_id) workgroup_id: vec3u,
    @builtin(local_invocation_index) local_invocation_index: u32,
) {
  // stuff
}
`

	functions := extractFunctions(code, map[int]string{}, []ShaderDefBlock{})

	expectedFunctions := []Function{
		{
			StageAttribute: "",
			Name:           "selectCorner",
			LineNumber:     2,
			Params: []NamedType{
				{
					Annotations: []Annotation{},
					Name:        "p",
					TypeInfo: TypeInfo{
						Annotations:   nil,
						Type:          "vec2<f32>",
						FullTypePath:  "vec2<f32>",
						TypeLink:      "",
						TypeLinkBlank: false,
					},
					HasShaderDefs: false,
					ShaderDefs:    nil,
				},
				{
					Annotations: []Annotation{},
					Name:        "c",
					TypeInfo: TypeInfo{
						Annotations:   nil,
						Type:          "vec4<f32>",
						FullTypePath:  "vec4<f32>",
						TypeLink:      "",
						TypeLinkBlank: false,
					},
					HasShaderDefs: false,
					ShaderDefs:    nil,
				},
			},
			ReturnTypeInfo: TypeInfo{
				Annotations:   []Annotation{},
				Type:          "f32",
				FullTypePath:  "",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
			Comment:       "",
			HasParams:     true,
		},
		{
			StageAttribute: "vertex",
			Name:           "vertex",
			LineNumber:     6,
			Params: []NamedType{
				{
					Annotations: []Annotation{
						{Name: "location", Value: "0"},
					},
					Name: "vertex_position",
					TypeInfo: TypeInfo{
						Annotations:   nil,
						Type:          "vec3<f32>",
						FullTypePath:  "vec3<f32>",
						TypeLink:      "",
						TypeLinkBlank: false,
					},
					HasShaderDefs: false,
					ShaderDefs:    nil,
				},
			},
			ReturnTypeInfo: TypeInfo{
				Annotations:   []Annotation{},
				Type:          "BoxShadowVertexOutput",
				FullTypePath:  "",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
			Comment:       "",
			HasParams:     true,
		},
		{
			StageAttribute: "fragment",
			Name:           "fragment",
			LineNumber:     13,
			Params: []NamedType{
				{
					Annotations: []Annotation{},
					Name:        "in",
					TypeInfo: TypeInfo{
						Annotations:   nil,
						Type:          "BoxShadowVertexOutput",
						FullTypePath:  "BoxShadowVertexOutput",
						TypeLink:      "",
						TypeLinkBlank: false,
					},
					HasShaderDefs: false,
					ShaderDefs:    nil,
				},
			},
			ReturnTypeInfo: TypeInfo{
				Annotations: []Annotation{
					{Name: "location", Value: "0"},
				},
				Type:          "vec4<f32>",
				FullTypePath:  "",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
			Comment:       "",
			HasParams:     true,
		},
		{
			StageAttribute: "compute",
			WorkgroupSize:  []string{"256", "1", "1"},
			Name:           "downsample_depth_first",
			LineNumber:     19,
			Params: []NamedType{
				{
					Annotations: []Annotation{
						{Name: "builtin", Value: "num_workgroups"},
					},
					Name: "num_workgroups",
					TypeInfo: TypeInfo{
						Annotations:   nil,
						Type:          "vec3u",
						FullTypePath:  "vec3u",
						TypeLink:      "",
						TypeLinkBlank: false,
					},
					HasShaderDefs: false,
					ShaderDefs:    nil,
				},
				{
					Annotations: []Annotation{
						{Name: "builtin", Value: "workgroup_id"},
					},
					Name: "workgroup_id",
					TypeInfo: TypeInfo{
						Annotations:   nil,
						Type:          "vec3u",
						FullTypePath:  "vec3u",
						TypeLink:      "",
						TypeLinkBlank: false,
					},
					HasShaderDefs: false,
					ShaderDefs:    nil,
				},
				{
					Annotations: []Annotation{
						{Name: "builtin", Value: "local_invocation_index"},
					},
					Name: "local_invocation_index",
					TypeInfo: TypeInfo{
						Annotations:   nil,
						Type:          "u32",
						FullTypePath:  "u32",
						TypeLink:      "",
						TypeLinkBlank: false,
					},
					HasShaderDefs: false,
					ShaderDefs:    nil,
				},
			},
			ReturnTypeInfo: TypeInfo{
				Annotations:   []Annotation{},
				Type:          "void",
				FullTypePath:  "",
				TypeLink:      "",
				TypeLinkBlank: false,
			},
			HasShaderDefs: false,
			ShaderDefs:    nil,
			Comment:       "",
			HasParams:     true,
		},
	}

	// PrintAsJson(functions)

	for i := range functions {
		assert.Equal(t, expectedFunctions[i], functions[i])
	}
}
