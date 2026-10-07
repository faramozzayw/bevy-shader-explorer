package generation

import (
	"fmt"
	"strings"
)

const packageDescriptionLimit = 155

func packageSEODescription(packageName, version, description string, shaderCount int) string {
	base := strings.Join(strings.Fields(description), " ")
	if base == "" {
		base = fmt.Sprintf("Searchable WGSL shader documentation for %s %s", packageName, version)
	}
	base = strings.TrimRight(base, ".!? ")
	suffix := fmt.Sprintf(". Browse %d shader modules from %s %s.", shaderCount, packageName, version)
	available := packageDescriptionLimit - len([]rune(suffix))
	if available < 1 {
		return strings.TrimSpace(string([]rune(suffix)[1:]))
	}
	if len([]rune(base)) > available {
		base = strings.TrimSpace(string([]rune(base)[:available-1])) + "…"
	}
	return base + suffix
}
