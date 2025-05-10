package main

import (
	_ "embed"
	"strings"

	"github.com/aymerick/raymond"
	"github.com/gomarkdown/markdown"
)

//go:embed templates/wgsl-doc.hbs
var WGSL_DOC_TEMPLATE_SOURCE string

//go:embed templates/404.hbs
var NOT_FOUND_TEMPLATE_SOURCE string

//go:embed templates/home.hbs
var HOME_DOC_TEMPLATE_SOURCE string

//go:embed templates/partials/shader-defs-list.hbs
var SHADER_DEFS_LIST_TEMPLATE string

//go:embed templates/partials/type.hbs
var TYPE_TEMPLATE string

//go:embed templates/partials/head.hbs
var HEAD_TEMPLATE string

//go:embed templates/partials/gh-link.hbs
var GH_LINK_TEMPLATE string

//go:embed templates/partials/annotations.hbs
var ANNOTATIONS_TEMPLATE string

//go:embed templates/partials/header.hbs
var HEADER_TEMPLATE string

//go:embed templates/partials/version-selector.hbs
var VERSION_SELECTOR_TEMPLATE string

func SetupHandlebars() {
	raymond.RegisterHelper("eq", eq)
	raymond.RegisterHelper("neq", neq)
	raymond.RegisterHelper("parse-markdown", parseMarkdown)
	raymond.RegisterHelper("contains", contains)

	raymond.RegisterPartial("shader-defs-list", SHADER_DEFS_LIST_TEMPLATE)
	raymond.RegisterPartial("type", TYPE_TEMPLATE)
	raymond.RegisterPartial("head", HEAD_TEMPLATE)
	raymond.RegisterPartial("gh-link", GH_LINK_TEMPLATE)
	raymond.RegisterPartial("annotations", ANNOTATIONS_TEMPLATE)
	raymond.RegisterPartial("header", HEADER_TEMPLATE)
	raymond.RegisterPartial("version-selector", VERSION_SELECTOR_TEMPLATE)
}

func eq(a, b interface{}) bool {
	return a == b
}

func neq(a, b interface{}) bool {
	return a != b
}

func parseMarkdown(text string) string {
	maybeUnsafeHTML := markdown.ToHTML([]byte(text), nil, nil)
	return strings.TrimSpace(string(maybeUnsafeHTML))
}

func contains(needle, haystack string) bool {
	return strings.Contains(haystack, needle)
}
