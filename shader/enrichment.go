package shader

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	config "github.com/faramozzayw/bevy-shader-explorer/config"
	"github.com/faramozzayw/bevy-shader-explorer/shader/document"
	"github.com/faramozzayw/bevy-shader-explorer/utils"
)

// FileEnricher adds deployment and source metadata to a parsed document.
// Parsing stays independent from package configuration and GitHub links.
type FileEnricher interface {
	Enrich(document.File, FileMetadata) (ShaderFile, error)
}

type FileMetadata struct {
	Config *config.Config
	Path   string
}

type ShaderFileEnricher struct{}

func (ShaderFileEnricher) Enrich(file document.File, metadata FileMetadata) (ShaderFile, error) {
	if metadata.Config == nil {
		return ShaderFile{}, fmt.Errorf("file metadata requires config")
	}
	basename := filepath.Base(metadata.Path)
	extension := filepath.Ext(basename)
	filename := strings.TrimSuffix(basename, extension)
	originalDir := filepath.Dir(metadata.Path)
	innerPath, err := filepath.Rel(metadata.Config.SourcePath, originalDir)
	if err != nil {
		return ShaderFile{}, fmt.Errorf("resolve source path: %w", err)
	}
	innerPath = strings.ReplaceAll(innerPath, "src"+string(filepath.Separator), "")
	wgslPath := utils.DedupPathParts(filepath.Join(innerPath, filename)) + ".html"
	githubLink, err := GetGithubLink(metadata.Config, originalDir, basename)
	if err != nil {
		return ShaderFile{}, fmt.Errorf("build source link: %w", err)
	}

	file.Filename = basename
	result := ShaderFile{
		Version:    metadata.Config.Version,
		WgslPath:   wgslPath,
		GithubLink: githubLink,
		Link:       fmt.Sprintf("%s/%s", metadata.Config.Version, wgslPath),
		File:       file,
	}
	result.ShaderDefs = result.CollectedShaderDefs()
	result.DeclarationShaderDefs = result.CollectedDeclarationShaderDefs()
	return result, nil
}

func GetGithubLink(config *config.Config, dir, basename string) (string, error) {
	if config.SourceGithubURL == "" {
		return "", nil
	}
	repositoryRoot := config.SourceGithubRoot
	if repositoryRoot == "" {
		repositoryRoot = config.SourcePath
	}
	innerPath, err := filepath.Rel(repositoryRoot, dir)
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}
	joinedPath := filepath.Join(config.SourceGithubSubpath, innerPath, basename)
	if config.SourceGithubRef == "" {
		return "", nil
	}
	baseURL, err := url.Parse(strings.TrimRight(config.SourceGithubURL, "/") + "/blob/" + url.PathEscape(config.SourceGithubRef) + "/")
	if err != nil {
		return "", fmt.Errorf("parse repository URL: %w", err)
	}
	return baseURL.ResolveReference(&url.URL{Path: joinedPath}).String(), nil
}
