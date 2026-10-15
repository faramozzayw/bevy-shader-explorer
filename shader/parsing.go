package shader

import (
	"path/filepath"

	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
	"github.com/faramozzayw/bevy-shader-explorer/shader/wesl"
	"github.com/faramozzayw/bevy-shader-explorer/shader/wgsl"
)

// ShaderParser turns source into the parser-owned document model.
type ShaderParser interface {
	Parse(document.Source) (document.File, error)
}

type Source = document.Source

func parserFor(path string) ShaderParser {
	if filepath.Ext(path) == ".wesl" {
		return wesl.Parser{}
	}
	return wgsl.Parser{}
}
