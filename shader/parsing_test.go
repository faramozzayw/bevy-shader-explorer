package shader

import "testing"

import "github.com/faramozzayw/bevy-shader-explorer/shader/document"

func TestLanguageForSelectsByExtension(t *testing.T) {
	if _, ok := parserFor("shader.wgsl").(interface {
		Parse(document.Source) (document.File, error)
	}); !ok {
		t.Fatalf("expected WGSL strategy")
	}
	if _, ok := parserFor("shader.wesl").(interface {
		Parse(document.Source) (document.File, error)
	}); !ok {
		t.Fatalf("expected WESL strategy")
	}
}

func TestWESLLanguageConvertsConditionalBlocks(t *testing.T) {
	parsed, err := parserFor("shader.wesl").Parse(document.Source{Path: "shader.wesl", Text: "@if(FOO)\nconst value: u32 = 1;\n"})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.ShaderDefs) != 1 || parsed.ShaderDefs[0] != "FOO" {
		t.Fatalf("unexpected definitions: %#v", parsed.ShaderDefs)
	}
}
