package generation

import (
	"path/filepath"

	"github.com/faramozzayw/bevy-shader-explorer/config"
)

// GenerationContext is the immutable, build-scoped state shared by generation
// stages. Config remains the public input; this type contains the normalized
// config and paths derived from it once at the composition root.
type GenerationContext struct {
	Config         config.Config
	OutputDir      string
	PublicDir      string
	ProjectName    string
	ProjectVersion string
	SiteURL        string
}

func NewGenerationContext(cfg config.Config) GenerationContext {
	if outputDir, err := filepath.Abs(cfg.OutputDir); err == nil {
		cfg.OutputDir = outputDir
	}
	if sourcePath, err := filepath.Abs(cfg.SourcePath); err == nil {
		cfg.SourcePath = sourcePath
	}
	if cfg.SourceGithubRoot != "" {
		if root, err := filepath.Abs(cfg.SourceGithubRoot); err == nil {
			cfg.SourceGithubRoot = root
		}
	}
	return GenerationContext{
		Config:         cfg,
		OutputDir:      cfg.OutputDir,
		PublicDir:      filepath.Join(cfg.OutputDir, "public"),
		ProjectName:    cfg.Name,
		ProjectVersion: cfg.ProjectVersion,
		SiteURL:        cfg.SiteURL,
	}
}
