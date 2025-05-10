package main

import (
	_ "embed"
	"strings"

	"github.com/aymerick/raymond"
	"github.com/gomarkdown/markdown"
)

//go:embed templates/partials/shader-defs-list.hbs
var SHADER_DEFS_LIST_TEMPLATE string

//go:embed templates/partials/type.hbs
var TYPE_TEMPLATE string

//go:embed templates/partials/gh-link.hbs
var GH_LINK_TEMPLATE string

//go:embed templates/partials/annotations.hbs
var ANNOTATIONS_TEMPLATE string

//go:embed templates/partials/header.hbs
var HEADER_TEMPLATE string

func RegisterHelpers() {
	// Register helpers
	raymond.RegisterHelper("eq", eq)
	raymond.RegisterHelper("neq", neq)
	raymond.RegisterHelper("parse-markdown", parseMarkdown)
	raymond.RegisterHelper("contains", contains)
}

func RegisterPartials() {
	raymond.RegisterPartial("shader-defs-list", SHADER_DEFS_LIST_TEMPLATE)
	raymond.RegisterPartial("type", TYPE_TEMPLATE)
	raymond.RegisterPartial("gh-link", GH_LINK_TEMPLATE)
	raymond.RegisterPartial("annotations", ANNOTATIONS_TEMPLATE)
	raymond.RegisterPartial("header", HEADER_TEMPLATE)
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

// checks if haystack contains needle
func contains(needle, haystack string) bool {
	return strings.Contains(haystack, needle)
}
