package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewShaderConditionNormalizesExpression(t *testing.T) {
	condition := NewShaderCondition("COMBINE_BIND_GROUP || SPLIT_BIND_GROUP_FIRST")

	assert.Equal(t, "COMBINE_BIND_GROUP || SPLIT_BIND_GROUP_FIRST", condition.Expression)
	assert.Equal(t, []string{"COMBINE_BIND_GROUP", "SPLIT_BIND_GROUP_FIRST"}, condition.Atoms)
	assert.Equal(t, [][]ShaderTerm{
		{{Name: "COMBINE_BIND_GROUP"}},
		{{Name: "SPLIT_BIND_GROUP_FIRST"}},
	}, condition.Clauses)
}

func TestNewShaderConditionPreservesBooleanTerms(t *testing.T) {
	condition := NewShaderCondition("FOO && !BAR || BAZ")

	assert.Equal(t, []string{"FOO", "BAR", "BAZ"}, condition.Atoms)
	assert.Equal(t, [][]ShaderTerm{
		{{Name: "FOO"}, {Name: "BAR", Negated: true}},
		{{Name: "BAZ"}},
	}, condition.Clauses)
}
