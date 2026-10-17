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

func TestWGSLParserIgnoresDefinitionsInsideFunctionBodies(t *testing.T) {
	source := `#ifdef TOP_LEVEL
fn main() {
#ifdef INSIDE_FUNCTION
    let value: u32 = 1u;
#endif
}
#endif
`

	parsed, err := parserFor("shader.wgsl").Parse(document.Source{Path: "shader.wgsl", Text: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.ShaderDefs) != 1 || parsed.ShaderDefs[0] != "TOP_LEVEL" {
		t.Fatalf("unexpected definitions: %#v", parsed.ShaderDefs)
	}
	if len(parsed.ShaderDefBlocks) != 1 || parsed.ShaderDefBlocks[0].IfLine != 1 || parsed.ShaderDefBlocks[0].EndifLine != 7 {
		t.Fatalf("unexpected definition source ranges: %#v", parsed.ShaderDefBlocks)
	}
}

func TestWESLParserIgnoresDefinitionsInsideFunctionBodies(t *testing.T) {
	source := `@if(TOP_LEVEL)
const value: u32 = 1u;

fn main() {
@if(INSIDE_FUNCTION) {
    let nested = 1u;
}
}`

	parsed, err := parserFor("shader.wesl").Parse(document.Source{Path: "shader.wesl", Text: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.ShaderDefs) != 1 || parsed.ShaderDefs[0] != "TOP_LEVEL" {
		t.Fatalf("unexpected definitions: %#v", parsed.ShaderDefs)
	}
}
