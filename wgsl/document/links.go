package document

import (
	"slices"
	"strings"

	"main/utils"
)

// ResolveTypeLink adds a documentation link after extraction is complete.
func (typeInfo *TypeInfo) ResolveTypeLink(imports map[string]string, definedStructures []string) {
	if typeInfo.TypeLink = utils.GetTypeLink(typeInfo.Type); typeInfo.TypeLink != "" {
		return
	}
	if typeInfo.FullTypePath == "" {
		typeInfo.FullTypePath = typeInfo.Type
	}
	if link, ok := imports[strings.Split(typeInfo.FullTypePath, "::")[0]]; ok {
		typeInfo.TypeLink, typeInfo.TypeLinkBlank = link+"#"+typeInfo.Type, true
		return
	}
	if slices.Contains(definedStructures, typeInfo.Type) {
		typeInfo.TypeLink = "#" + typeInfo.Type
	}
}
