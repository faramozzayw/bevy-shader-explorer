package generation

import (
	_ "embed"
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/aymerick/raymond"
	"github.com/gomarkdown/markdown"
)

//go:embed templates/wgsl-doc.hbs
var WGSL_DOC_TEMPLATE_SOURCE string

//go:embed templates/404.hbs
var NOT_FOUND_TEMPLATE_SOURCE string

//go:embed templates/home.hbs
var HOME_DOC_TEMPLATE_SOURCE string

//go:embed templates/package.hbs
var PACKAGE_DOC_TEMPLATE_SOURCE string

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

//go:embed templates/partials/footer.hbs
var FOOTER_TEMPLATE string

//go:embed templates/partials/project-header.hbs
var PROJECT_HEADER_TEMPLATE string

//go:embed templates/partials/version-selector.hbs
var VERSION_SELECTOR_TEMPLATE string

var handlebarsSetup sync.Once

func SetupHandlebars() {
	handlebarsSetup.Do(func() {
		raymond.RegisterHelper("eq", eq)
		raymond.RegisterHelper("neq", neq)
		raymond.RegisterHelper("parse-markdown", parseMarkdown)
		raymond.RegisterHelper("contains", contains)
		raymond.RegisterHelper("group-by-name", groupByName)
		raymond.RegisterHelper("json", jsonValue)

		raymond.RegisterPartial("shader-defs-list", SHADER_DEFS_LIST_TEMPLATE)
		raymond.RegisterPartial("type", TYPE_TEMPLATE)
		raymond.RegisterPartial("head", HEAD_TEMPLATE)
		raymond.RegisterPartial("gh-link", GH_LINK_TEMPLATE)
		raymond.RegisterPartial("annotations", ANNOTATIONS_TEMPLATE)
		raymond.RegisterPartial("header", HEADER_TEMPLATE)
		raymond.RegisterPartial("footer", FOOTER_TEMPLATE)
		raymond.RegisterPartial("project-header", PROJECT_HEADER_TEMPLATE)
		raymond.RegisterPartial("version-selector", VERSION_SELECTOR_TEMPLATE)
	})
}

func jsonValue(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
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

type namedTemplateGroup struct {
	Name  string
	First interface{}
	Items interface{}
}

func groupByName(values interface{}) []namedTemplateGroup {
	value := reflect.ValueOf(values)
	if !value.IsValid() || (value.Kind() != reflect.Slice && value.Kind() != reflect.Array) {
		return nil
	}

	type groupState struct {
		name  string
		items reflect.Value
	}
	groups := make([]groupState, 0, value.Len())
	byName := make(map[string]int, value.Len())
	for i := 0; i < value.Len(); i++ {
		item := value.Index(i)
		if item.Kind() == reflect.Pointer {
			if item.IsNil() {
				continue
			}
			item = item.Elem()
		}
		if item.Kind() != reflect.Struct {
			continue
		}
		nameField := item.FieldByName("Name")
		if !nameField.IsValid() || nameField.Kind() != reflect.String {
			continue
		}
		name := nameField.String()
		index, exists := byName[name]
		if !exists {
			index = len(groups)
			byName[name] = index
			groups = append(groups, groupState{name: name, items: reflect.MakeSlice(value.Type(), 0, 1)})
		}
		groups[index].items = reflect.Append(groups[index].items, value.Index(i))
	}

	result := make([]namedTemplateGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, namedTemplateGroup{Name: group.name, First: group.items.Index(0).Interface(), Items: group.items.Interface()})
	}
	return result
}
