package generation

import (
	"fmt"

	"github.com/faramozzayw/bevy-shader-explorer/config"
	"github.com/faramozzayw/bevy-shader-explorer/shader"
	"github.com/faramozzayw/bevy-shader-explorer/shader/highlight"
)

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

type sourceLineModel struct {
	Number int
	Html   string
}

type sourceTemplateModel struct {
	Filename         string
	Lines            []sourceLineModel
	LineCount        int
	GithubLink       string
	BackLink         string
	ProjectName      string
	ProjectVersion   string
	ProjectURLPrefix string
	PackageURLPrefix string
	Version          string
	VersionOptions   []map[string]string
	SearchIndex      string
	SeoTitle         string
	SeoDescription   string
	CanonicalURL     string
	OgImageURL       string
	StructuredData   string
}

func newSourceTemplateModel(file shader.ShaderFile) sourceTemplateModel {
	lines := highlight.Lines(file.SourceText, file.Filename)
	modelLines := make([]sourceLineModel, len(lines))
	for i, line := range lines {
		modelLines[i] = sourceLineModel{Number: line.Number, Html: line.HTML}
	}
	return sourceTemplateModel{
		Filename: file.Filename, Lines: modelLines, LineCount: len(modelLines),
		GithubLink: file.GithubLink, BackLink: "/" + file.Link,
		ProjectName: file.ProjectName, ProjectVersion: file.ProjectVersion,
		ProjectURLPrefix: file.ProjectURLPrefix, PackageURLPrefix: file.PackageURLPrefix,
		Version: file.Version, VersionOptions: file.VersionOptions, SearchIndex: file.SearchIndex,
		SeoTitle:       fmt.Sprintf("%s source — %s", file.Filename, file.ProjectName),
		SeoDescription: fmt.Sprintf("Source for the WGSL shader module %s from %s %s.", file.Filename, file.ProjectName, file.ProjectVersion),
		CanonicalURL:   file.SourceCanonicalURL, OgImageURL: file.OgImageURL,
		StructuredData: jsonValue(map[string]interface{}{
			"@context": "https://schema.org", "@type": "TechArticle",
			"name":        file.Filename + " source",
			"headline":    file.Filename + " source",
			"description": fmt.Sprintf("Source for the WGSL shader module %s from %s %s.", file.Filename, file.ProjectName, file.ProjectVersion),
			"url":         file.SourceCanonicalURL,
		}),
	}
}

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
