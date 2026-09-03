// Material settings used by the shader.
struct Settings {
    @align(16) tint: vec4<f32>,
    enabled: u32,
}

const SCALE: f32 = 2.0;
const MAX_LIGHTS = 4u;
@id(7) override QUALITY: u32 = 1u;

@group(1) @binding(2)
var<storage, read_write> settings: Settings;

// Executes the material pass.
@compute @workgroup_size(8, 4, 1)
fn shade(
    @builtin(global_invocation_id) invocation_id: vec3<u32>,
    factor: f32,
) -> @location(0) vec4<f32> {
}
