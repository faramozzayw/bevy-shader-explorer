package bevy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskDirectivesPreservesSourcePositions(t *testing.T) {
	source := "#import bevy_pbr::{\n    mesh_functions,\n}\n#ifdef PREPASS\n@group(0) @binding(0)\nvar<uniform> settings: Settings;\n#else\nconst VALUE: u32 = 1u;\n#endif\n"

	masked := MaskDirectives(source)
	assert.Equal(t, len(source), len(masked))
	assert.Equal(t, strings.Count(source, "\n"), strings.Count(masked, "\n"))
	assert.Contains(t, masked, "var<uniform> settings: Settings;")
	assert.Contains(t, masked, "const VALUE: u32 = 1u;")

	for index := range source {
		if source[index] != '\n' && masked[index] != source[index] {
			assert.Equal(t, byte(' '), masked[index])
		}
	}
}
