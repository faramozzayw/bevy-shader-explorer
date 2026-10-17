package generation

import (
	"testing"

	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
)

func TestSourceLineShaderDefsTracksBranches(t *testing.T) {
	elseLine := 5
	blocks := []document.ShaderDefBlock{{
		Condition: document.NewShaderCondition("FEATURE"),
		IfLine:    3, ElseLine: &elseLine, EndifLine: 7,
	}}

	if got := sourceLineShaderDefs(blocks, 4); len(got) != 1 || got[0].Branch != "if" {
		t.Fatalf("expected if branch, got %#v", got)
	}
	if got := sourceLineShaderDefs(blocks, 6); len(got) != 1 || got[0].Branch != "else" {
		t.Fatalf("expected else branch, got %#v", got)
	}
	if got := sourceLineShaderDefs(blocks, 3); len(got) != 0 {
		t.Fatalf("expected directive line to be unclassified, got %#v", got)
	}
}
