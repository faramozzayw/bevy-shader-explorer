package generation

import "github.com/faramozzayw/bevy-shader-explorer/config"

type packageTemplateModel struct {
	Name                      string
	Files                     []map[string]string
	ModuleGroups              []moduleGroup
	ShowModuleGroups          bool
	Count                     int
	Description               string
	SeoDescription            string
	Dependencies              []map[string]string
	HasDependencies           bool
	DependencyCount           int
	DirectDependencies        []map[string]string
	TransitiveDependencies    []map[string]string
	HasDirectDependencies     bool
	HasTransitiveDependencies bool
	DirectDependencyCount     int
	TransitiveDependencyCount int
	Authors                   []string
	License                   string
	Repository                string
	Homepage                  string
	PackageVersion            string
	SearchIndex               string
	SeoTitle                  string
	Version                   string
	ProjectVersion            string
	ProjectCount              int
	URLPrefix                 string
	PackageURLPrefix          string
	VersionOptions            []map[string]string
	CanonicalURL              string
	OgImageURL                string
	IssueURL                  string
	StructuredData            string
}

type homeTemplateModel struct {
	Sections         []homeSection
	PackageCount     int
	TotalShaderCount int
	ProjectCount     int
	DependencyCount  int
	Name             string
	Description      string
	Version          string
	ProjectVersion   string
	URLPrefix        string
	CanonicalURL     string
	OgImageURL       string
	IssueURL         string
	StructuredData   string
}

type notFoundTemplateModel struct{}

func newHomeTemplateModel(cfg config.Config, sections []homeSection, packageCount, shaderCount, projectCount, dependencyCount int, projectVersion string) homeTemplateModel {
	return homeTemplateModel{
		Sections: sections, PackageCount: packageCount, TotalShaderCount: shaderCount,
		ProjectCount: projectCount, DependencyCount: dependencyCount, Name: cfg.Name,
		Description: cfg.Description, Version: cfg.Version, ProjectVersion: projectVersion,
		URLPrefix: joinDocURL("project", ""), CanonicalURL: canonicalURL(cfg.SiteURL, ""),
		OgImageURL: ogImageURL(cfg, "site", projectVersion), IssueURL: cfg.IssueURL,
		StructuredData: jsonValue(map[string]interface{}{
			"@context": "https://schema.org", "@type": "WebSite", "name": "Bevy Shader Explorer",
			"alternateName": "Shader Explorer", "description": cfg.Description, "url": canonicalURL(cfg.SiteURL, ""),
			"publisher": map[string]interface{}{"@type": "Organization", "name": "Bevy Shader Explorer", "url": "https://github.com/faramozzayw/bevy-shader-explorer"},
		}),
	}
}
