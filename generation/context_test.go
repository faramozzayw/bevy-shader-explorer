package generation

import (
	"path/filepath"
	"testing"

	"github.com/faramozzayw/bevy-shader-explorer/config"
)

func TestNewGenerationContextNormalizesDerivedPaths(t *testing.T) {
	context := NewGenerationContext(config.Config{Name: "demo", ProjectVersion: "1.0.0", SourcePath: ".", OutputDir: "dist"})
	if context.ProjectName != "demo" || context.ProjectVersion != "1.0.0" {
		t.Fatalf("unexpected project identity: %#v", context)
	}
	if context.OutputDir == "dist" || context.PublicDir != filepath.Join(context.OutputDir, "public") {
		t.Fatalf("context paths were not normalized: %#v", context)
	}
}
