#ifdef FEATURE
const ENABLED: bool = true;
struct FeatureSettings {
    value: f32,
}
#else
@group(0) @binding(0)
var<uniform> fallback: FallbackSettings;
#endif
