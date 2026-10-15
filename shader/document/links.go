package document

import (
	"slices"
	"strings"

	"github.com/faramozzayw/bevy-shader-explorer/utils"
)

// ResolveTypeLink adds a documentation link after extraction is complete.
func (typeInfo *TypeInfo) ResolveTypeLink(imports map[string]string, definedStructures []string) {
	typeInfo.TypeLink = utils.GetTypeLink(typeInfo.Type)
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
	typeInfo.TypeParts = resolveTypeParts(typeInfo.Type, imports, definedStructures)
}

func resolveTypeParts(expression string, imports map[string]string, definedStructures []string) []TypePart {
	parts := make([]TypePart, 0, 4)
	for index := 0; index < len(expression); {
		start := index
		if isTypeIdentifierStart(expression[index]) {
			index++
			for index < len(expression) && isTypeIdentifierPart(expression[index]) {
				index++
			}
			for index+2 < len(expression) && expression[index:index+2] == "::" && isTypeIdentifierStart(expression[index+2]) {
				index += 2
				for index < len(expression) && isTypeIdentifierPart(expression[index]) {
					index++
				}
			}
			name := expression[start:index]
			link, blank := resolveNestedTypeLink(name, imports, definedStructures)
			parts = append(parts, TypePart{Text: name, TypeLink: link, TypeLinkBlank: blank})
			continue
		}
		index++
		parts = append(parts, TypePart{Text: expression[start:index]})
	}
	return parts
}

func resolveNestedTypeLink(name string, imports map[string]string, definedStructures []string) (string, bool) {
	if link := utils.GetTypeLink(name); link != "" {
		return link, false
	}
	if link, ok := imports[strings.Split(name, "::")[0]]; ok {
		return link + "#" + utils.RemovePath(name), true
	}
	if slices.Contains(definedStructures, name) {
		return "#" + name, false
	}
	return "", false
}

func isTypeIdentifierStart(char byte) bool {
	return char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z'
}

func isTypeIdentifierPart(char byte) bool {
	return isTypeIdentifierStart(char) || char >= '0' && char <= '9'
}
