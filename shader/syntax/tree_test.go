package syntax

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWESLConditionalBlocksExtractsDeclarationBranches(t *testing.T) {
	source := `@if(FEATURE)
const enabled: u32 = 1u;
@else
const disabled: u32 = 0u;
`

	blocks, err := WESLConditionalBlocks(source)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, ConditionalBlock{Name: "FEATURE", IfLine: 1, ElseLine: intPointer(3), EndifLine: 6}, blocks[0])
}

func TestWESLConditionalBlocksExtractsBracedBranches(t *testing.T) {
	source := `fn main() {
@if(FEATURE) {
    let enabled = 1u;
} @else {
    let disabled = 0u;
}`

	blocks, err := WESLConditionalBlocks(source)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, "FEATURE", blocks[0].Name)
	assert.Equal(t, 2, blocks[0].IfLine)
	assert.NotNil(t, blocks[0].ElseLine)
	assert.Equal(t, 4, *blocks[0].ElseLine)
}

func TestFunctionRangesExcludeOnlyFunctionDeclarations(t *testing.T) {
	source := `struct Settings {
    value: u32,
}
fn main() {
    let value = 1u;
}`

	ranges, err := FunctionRanges(source, source)
	require.NoError(t, err)
	require.Len(t, ranges, 1)
	assert.Equal(t, LineRange{Start: 4, End: 6}, ranges[0])
}

func intPointer(value int) *int { return &value }
