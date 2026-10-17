package highlight

import (
	"strings"
	"testing"
)

func TestLinesNumbersAndTrailingNewline(t *testing.T) {
	lines := Lines("fn main() {\n    return;\n}\n", "test.wgsl")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %#v", len(lines), lines)
	}
	if lines[0].Number != 1 || lines[2].Number != 3 {
		t.Fatalf("unexpected line numbers: %#v", lines)
	}
}

func TestLinesHighlightsKeywordAndType(t *testing.T) {
	lines := Lines("fn main() -> f32 {}", "test.wgsl")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	html := lines[0].HTML
	if !strings.Contains(html, `<span class="tok-kw">fn</span>`) {
		t.Fatalf("expected fn to be highlighted as keyword: %s", html)
	}
	if !strings.Contains(html, `<span class="tok-type">f32</span>`) {
		t.Fatalf("expected f32 to be highlighted as type: %s", html)
	}
}

func TestLinesHighlightsLineComment(t *testing.T) {
	lines := Lines("let x = 1; // hello", "test.wgsl")
	if !strings.Contains(lines[0].HTML, `<span class="tok-comment">// hello</span>`) {
		t.Fatalf("expected comment span: %s", lines[0].HTML)
	}
}

func TestLinesHighlightsMultilineBlockComment(t *testing.T) {
	lines := Lines("/* a\nb */\nfn f() {}", "test.wgsl")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %#v", len(lines), lines)
	}
	if !strings.Contains(lines[0].HTML, `tok-comment`) || !strings.Contains(lines[1].HTML, `tok-comment`) {
		t.Fatalf("expected comment class to continue across lines: %#v", lines)
	}
}

func TestLinesHighlightsDirectiveAtLineStart(t *testing.T) {
	lines := Lines("#import bevy_pbr::mesh_view_bindings::view\nfn f() {}", "test.wgsl")
	if !strings.Contains(lines[0].HTML, `<span class="tok-directive">`) {
		t.Fatalf("expected directive span: %s", lines[0].HTML)
	}
}

func TestLinesHighlightsAttribute(t *testing.T) {
	lines := Lines("@group(0) @binding(0) var<uniform> settings: Settings;", "test.wgsl")
	html := lines[0].HTML
	if !strings.Contains(html, `<span class="tok-attr">@</span><span class="tok-attr">group</span>`) {
		t.Fatalf("expected @group to be highlighted as an attribute: %s", html)
	}
	if !strings.Contains(html, `<span class="tok-number">0</span>`) {
		t.Fatalf("expected the attribute argument to be highlighted as a number, not an attribute: %s", html)
	}
}

func TestLinesHighlightsNumberLiterals(t *testing.T) {
	lines := Lines("const SCALE: f32 = 1.5e-3f;", "test.wgsl")
	if !strings.Contains(lines[0].HTML, `<span class="tok-number">1.5e-3f</span>`) {
		t.Fatalf("expected the float literal to be highlighted as a number: %s", lines[0].HTML)
	}
}

func TestLinesEscapesHTML(t *testing.T) {
	lines := Lines("let x = a < b;", "test.wgsl")
	if strings.Contains(lines[0].HTML, "<b") && !strings.Contains(lines[0].HTML, "&lt;") {
		t.Fatalf("expected escaped angle brackets: %s", lines[0].HTML)
	}
	if !strings.Contains(lines[0].HTML, "&lt;") {
		t.Fatalf("expected escaped '<': %s", lines[0].HTML)
	}
}

func TestLinesEmptySource(t *testing.T) {
	if lines := Lines("", "test.wgsl"); lines != nil {
		t.Fatalf("expected nil lines for empty source, got %#v", lines)
	}
}

func TestLinesSurvivesMalformedSource(t *testing.T) {
	// Tree-sitter is error-tolerant: malformed input still parses into a
	// best-effort tree (with ERROR nodes), so this exercises the renderer
	// against that tree rather than the plain-text fallback.
	lines := Lines("fn ((( broken", "test.wgsl")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %#v", len(lines), lines)
	}
	html := lines[0].HTML
	for _, want := range []string{"fn", "(((", "broken"} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected source text %q to still appear: %s", want, html)
		}
	}
}

func TestLinesWESLUsesNativeImportAndCommentGrammar(t *testing.T) {
	source := "// a comment\nimport bevy_pbr::mesh_view_bindings::view;\n\n@fragment\nfn main() -> @location(0) vec4<f32> {\n    return vec4<f32>(1.0, 0.0, 0.0, 1.0);\n}\n"
	lines := Lines(source, "test.wesl")
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines, got %d: %#v", len(lines), lines)
	}
	if !strings.Contains(lines[0].HTML, `<span class="tok-comment">// a comment</span>`) {
		t.Fatalf("expected comment span: %s", lines[0].HTML)
	}
	if !strings.Contains(lines[1].HTML, `<span class="tok-kw">import</span>`) {
		t.Fatalf("expected import to be highlighted as a keyword: %s", lines[1].HTML)
	}
	if !strings.Contains(lines[3].HTML, `<span class="tok-attr">@</span><span class="tok-attr">fragment</span>`) {
		t.Fatalf("expected @fragment to be highlighted as an attribute: %s", lines[3].HTML)
	}
}
