package ncservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	val1 := Value{Col: "col1", Val: "value1"}
	val2 := Value{Col: "col2", Val: nil}
	val3 := Value{Col: "col3", Val: 42}

	// Test FilterAll
	assert.True(t, FilterAll(val1))
	assert.True(t, FilterAll(val2))

	// Test FilterNotNil
	assert.True(t, FilterNotNil(val1))
	assert.False(t, FilterNotNil(val2))

	// Test FilterAnd
	andFilter := FilterAnd(FilterNotNil, func(v Value) bool {
		return v.Col != "col3"
	})
	assert.True(t, andFilter(val1))
	assert.False(t, andFilter(val2))
	assert.False(t, andFilter(val3))

	// Test AppendFiltered
	args := []Value{}
	args = AppendFiltered(args, val1, FilterNotNil)
	assert.Len(t, args, 1)
	args = AppendFiltered(args, val2, FilterNotNil)
	assert.Len(t, args, 1) // val2 should not be added
}
