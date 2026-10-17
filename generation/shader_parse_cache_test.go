package generation

import "testing"

func TestCacheModelIsCurrentRequiresSourceDefinitionRanges(t *testing.T) {
	if cacheModelIsCurrent([]byte(`{"shaderDefs":[{"defName":"OLD"}]}`)) {
		t.Fatal("legacy definition cache should be invalidated")
	}
	if cacheModelIsCurrent([]byte(`{"shaderDefs":[],"consts":[]}`)) {
		t.Fatal("cache without source definition ranges should be invalidated")
	}
	if !cacheModelIsCurrent([]byte(`{"shaderDefs":[],"shaderDefBlocks":[]}`)) {
		t.Fatal("current empty source definition model should be accepted")
	}
}
