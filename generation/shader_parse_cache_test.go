package generation

import "testing"

func TestCacheModelIsCurrentRequiresNormalizedShaderDefinitions(t *testing.T) {
	if cacheModelIsCurrent([]byte(`{"shaderDefs":[{"defName":"OLD"}]}`)) {
		t.Fatal("legacy definition cache should be invalidated")
	}
	if !cacheModelIsCurrent([]byte(`{"shaderDefs":[],"consts":[]}`)) {
		t.Fatal("current empty definition model should be accepted")
	}
	if !cacheModelIsCurrent([]byte(`{"condition":{"clauses":[]}}`)) {
		t.Fatal("current normalized condition model should be accepted")
	}
}
