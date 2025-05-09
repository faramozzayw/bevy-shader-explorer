package main

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/aymerick/raymond"
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
	raymond.RegisterHelper("linkify", linkify)
	raymond.RegisterHelper("code-highlight", codeHighlight)
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

// turns URLs into clickable links
func linkify(text string) string {
	re := regexp.MustCompile(`(?:https?|ftp):\/\/[\n\S]+`)
	return re.ReplaceAllStringFunc(text, func(url string) string {
		return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, url, url)
	})
}

// highlights code wrapped with backticks
func codeHighlight(text string) string {
	re := regexp.MustCompile("`(.*?)`")
	return re.ReplaceAllString(text, `<code>$1</code>`)
}

// checks if haystack contains needle
func contains(needle, haystack string) bool {
	return strings.Contains(haystack, needle)
}
