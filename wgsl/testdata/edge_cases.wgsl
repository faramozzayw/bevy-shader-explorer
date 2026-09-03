/*
 * This comment spans multiple lines.
 */
// Position data from a previous pass.
struct VertexData {
    position: vec4<f32>,
}

const INFERRED_INT = 42;
const INFERRED_SIGNED = -7;
const INFERRED_BOOL = true;
const HEX_VALUE: u32 = 0x2Au;
const INFERRED_FLOAT = 3.5;
const INFERRED_VECTOR = vec3<f32>(1.0, 0.0, 0.0);
const DERIVED = INFERRED_INT + 1;

@group(0) @binding(0)
var<uniform> uniforms: VertexData;
@group(0) @binding(1)
var<storage, read> positions: VertexData;
@group(0) @binding(2)
var<uniform> matrices: array<mat4x4<f32>, 4>;

@vertex
fn vertex_main(@location(0) position: vec3<f32>) -> @builtin(position) vec4<f32> {
}

@fragment
fn fragment_main() {
}

fn helper(value: f32) -> f32 {
}
