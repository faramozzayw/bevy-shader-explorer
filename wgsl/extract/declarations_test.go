package extract

import (
	"testing"

	"github.com/faramozzayw/bevy-shader-explorer/wgsl/bevy"
)

func TestParsePreservesInlineInterpolationValue(t *testing.T) {
	code := "const SORTED_FRAGMENT_MAX_COUNT: u32 = #{SORTED_FRAGMENT_MAX_COUNT};\n"
	result, err := Parse(code, bevy.MaskDirectives(code), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Consts[0].Value; got != "#{SORTED_FRAGMENT_MAX_COUNT}" {
		t.Fatalf("value = %q, want complete interpolation", got)
	}
}
