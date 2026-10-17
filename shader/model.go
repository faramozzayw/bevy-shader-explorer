package shader

import "github.com/faramozzayw/bevy-shader-explorer/shader/document"

type DefResult = document.DefResult
type Annotation = document.Annotation
type TypeInfo = document.TypeInfo
type NamedType = document.NamedType
type Const = document.Const
type Structure = document.Structure
type Function = document.Function
type Binding = document.Binding

type DeclaredImports = document.DeclaredImports

type ShaderFile struct {
	document.File
	Version            string              `json:"version"`
	WgslPath           string              `json:"shaderFile"`
	GithubLink         string              `json:"githubLink"`
	Link               string              `json:"link"`
	ProjectName        string              `json:"projectName"`
	ProjectDescription string              `json:"projectDescription"`
	ProjectVersion     string              `json:"projectVersion"`
	ProjectURLPrefix   string              `json:"projectURLPrefix"`
	PackageURLPrefix   string              `json:"packageURLPrefix"`
	VersionOptions     []map[string]string `json:"versionOptions"`
	SearchIndex        string              `json:"-"`
	Dependency         bool                `json:"dependency"`
	SourcePath         string              `json:"-"`
	SourceRoot         string              `json:"-"`
	OutputPrefix       string              `json:"-"`
	SourceText         string              `json:"-"`
	SourceLink         string              `json:"sourceLink,omitempty"`
	SourceCanonicalURL string              `json:"-"`
	SeoDescription     string              `json:"-"`
	SeoTitle           string              `json:"-"`
	CanonicalURL       string              `json:"-"`
	StructuredData     string              `json:"-"`
	OgImageURL         string              `json:"-"`
}
