package generation

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	wgsl "github.com/faramozzayw/bevy-shader-explorer/shader"
)

type shaderParseCache interface {
	path(shaderInput, []byte) string
	load(string) (wgsl.ShaderFile, bool)
	store(string, wgsl.ShaderFile)
}

type filesystemShaderParseCache struct {
	directory string
}

func newShaderParseCache() shaderParseCache {
	return filesystemShaderParseCache{directory: filepath.Join(os.TempDir(), "bevy-shader-explorer-cache")}
}

func (cache filesystemShaderParseCache) path(input shaderInput, content []byte) string {
	hash := sha256.New()
	hash.Write(content)
	for _, value := range []string{
		input.Config.SourcePath, input.Config.SourceGithubRoot, input.Config.SourceGithubSubpath,
		input.Config.SourceGithubURL, input.Config.SourceGithubRef, input.Config.Version,
		input.Config.FileFilter, input.Config.ProjectVersion, input.Path,
	} {
		hash.Write([]byte("\x00"))
		hash.Write([]byte(value))
	}
	// Keep parser-model changes from reusing serialized documents produced by
	// an older generator. This marker is deliberately part of the key rather
	// than a user-facing cache-version setting.
	hash.Write([]byte("\x00shader-def-source-ranges-wesl-fields-declaration-source-defs-wesl-node-ranges-mask-function-attrs"))
	return filepath.Join(cache.directory, fmt.Sprintf("%x.json", hash.Sum(nil)))
}

func (cache filesystemShaderParseCache) load(path string) (wgsl.ShaderFile, bool) {
	data, err := os.ReadFile(path)
	if err != nil || !cacheModelIsCurrent(data) {
		return wgsl.ShaderFile{}, false
	}
	var file wgsl.ShaderFile
	if err := json.Unmarshal(data, &file); err != nil {
		return wgsl.ShaderFile{}, false
	}
	return file, true
}

func (cache filesystemShaderParseCache) store(path string, file wgsl.ShaderFile) {
	data, err := json.Marshal(file)
	if err != nil {
		return
	}
	if err := os.MkdirAll(cache.directory, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func cacheModelIsCurrent(data []byte) bool {
	// Cache entries created before source-level shader-definition ranges were
	// added cannot render variant-aware source pages. The field is serialized
	// even when empty, so this check invalidates older entries exactly once.
	return bytes.Contains(data, []byte(`"shaderDefBlocks"`))
}
