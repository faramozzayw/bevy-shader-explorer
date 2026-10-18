package document

import (
	"fmt"
	"slices"
	"strings"

	"github.com/faramozzayw/bevy-shader-explorer/utils"
)

// ResolveTypeLink adds a documentation link after extraction is complete.
func (typeInfo *TypeInfo) ResolveTypeLink(imports map[string]string, definedStructures []string) {
	typeInfo.Type = normalizeTypeExpression(typeInfo.Type)
	typeInfo.FullTypePath = normalizeTypeExpression(typeInfo.FullTypePath)
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

// normalizeTypeExpression keeps the source type readable when a parser emits
// optional whitespace around generic delimiters. Type expressions are rendered
// in several templates, so canonicalizing them here also prevents spacing
// differences between linked and plain types.
func normalizeTypeExpression(expression string) string {
	if expression == "" {
		return ""
	}

	// Collapse parser-introduced runs of whitespace first, then remove it only
	// where it is punctuation padding. Spaces after commas remain meaningful
	// and keep nested generic arguments readable (for example, array<T, 4>).
	expression = strings.Join(strings.Fields(expression), " ")
	for {
		normalized := strings.NewReplacer(
			" <", "<",
			"< ", "<",
			" >", ">",
			"> ", ">",
			" ,", ",",
		).Replace(expression)
		if normalized == expression {
			return normalized
		}
		expression = normalized
	}
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

func ResolveUsage(usedBy map[string][]UsageRef, keys ...string) ([]UsageRef, string) {
	var refs []UsageRef
	for _, key := range keys {
		refs = append(refs, usedBy[key]...)
	}
	if len(refs) == 0 {
		return nil, ""
	}

	seen := make(map[string]bool, len(refs))
	deduped := refs[:0]
	for _, ref := range refs {
		identity := ref.PackageName + "\x00" + ref.PackageVersion + "\x00" + ref.Filename
		if seen[identity] {
			continue
		}
		seen[identity] = true
		deduped = append(deduped, ref)
	}

	sorted := append([]UsageRef(nil), deduped...)
	slices.SortFunc(sorted, func(a, b UsageRef) int {
		if c := strings.Compare(a.PackageName, b.PackageName); c != 0 {
			return c
		}
		if c := strings.Compare(a.PackageVersion, b.PackageVersion); c != 0 {
			return c
		}
		return strings.Compare(a.Filename, b.Filename)
	})

	samePackage := true
	for _, ref := range sorted {
		if ref.PackageName != sorted[0].PackageName || ref.PackageVersion != sorted[0].PackageVersion {
			samePackage = false
			break
		}
	}

	noun := "shaders"
	if len(sorted) == 1 {
		noun = "shader"
	}
	summary := fmt.Sprintf("Used by %d %s", len(sorted), noun)
	if samePackage {
		summary = fmt.Sprintf("%s in %s %s", summary, sorted[0].PackageName, sorted[0].PackageVersion)
	}
	return sorted, summary
}

func ResolveModuleLink(fullPath string, declaredImportPaths map[string]string) (string, bool) {
	var longestMatch string
	for module := range declaredImportPaths {
		if strings.HasPrefix(fullPath, module) && len(module) > len(longestMatch) {
			longestMatch = module
		}
	}
	if longestMatch == "" {
		return "", false
	}
	return declaredImportPaths[longestMatch], true
}
