package ncservice

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	val1 := Value{Col: "col1", Val: "value1"}
	val2 := Value{Col: "col2", Val: nil}
	val3 := Value{Col: "col3", Val: 42}

	// Test FilterAll
	var fld reflect.StructField
	assert.True(t, FilterAll(val1, fld))
	assert.True(t, FilterAll(val2, fld))

	// Test FilterNotNil
	assert.True(t, FilterNotNil(val1, fld))
	assert.False(t, FilterNotNil(val2, fld))

	// Test FilterAnd
	andFilter := FilterAnd(FilterNotNil, func(v Value, fld reflect.StructField) bool {
		return v.Col != "col3"
	})
	assert.True(t, andFilter(val1, fld))
	assert.False(t, andFilter(val2, fld))
	assert.False(t, andFilter(val3, fld))

	// Test AppendFiltered
	args := []Value{}
	args = AppendFiltered(args, val1, fld, FilterNotNil)
	assert.Len(t, args, 1)
	args = AppendFiltered(args, val2, fld, FilterNotNil)
	assert.Len(t, args, 1) // val2 should not be added
}
