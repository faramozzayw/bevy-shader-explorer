package extract

import (
	"testing"

	"github.com/faramozzayw/bevy-shader-explorer/shader/bevy"
	"github.com/faramozzayw/bevy-shader-explorer/shader/syntax"
)

func TestParsePreservesInlineInterpolationValue(t *testing.T) {
	code := "const SORTED_FRAGMENT_MAX_COUNT: u32 = #{SORTED_FRAGMENT_MAX_COUNT};\n"
	result, err := ParseWithParser(code, bevy.MaskDirectives(code), nil, nil, syntax.Parse)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Consts[0].Value; got != "#{SORTED_FRAGMENT_MAX_COUNT}" {
		t.Fatalf("value = %q, want complete interpolation", got)
	}
}
