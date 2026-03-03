package ncservice

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	val1 := Value{Col: "col1", Val: "value1"}
	val2 := Value{Col: "col2", Val: nil}
	val3 := Value{Col: "col3", Val: 42}

	var fld reflect.StructField

	t.Run("FilterAll", func(t *testing.T) {
		assert.True(t, FilterAll(val1, fld))
		assert.True(t, FilterAll(val2, fld))
	})

	t.Run("FilterNotNil", func(t *testing.T) {
		assert.True(t, FilterNotNil(val1, fld))
		assert.False(t, FilterNotNil(val2, fld))
	})

	t.Run("FilterAnd", func(t *testing.T) {
		andFilter := FilterAnd(FilterNotNil, func(v Value, fld reflect.StructField) bool {
			return v.Col != "col3"
		})
		assert.True(t, andFilter(val1, fld))
		assert.False(t, andFilter(val2, fld))
		assert.False(t, andFilter(val3, fld))
	})

	t.Run("AppendFiltered", func(t *testing.T) {
		args := []Value{}
		args = AppendFiltered(args, val1, fld, FilterNotNil)
		assert.Len(t, args, 1)
		args = AppendFiltered(args, val2, fld, FilterNotNil)
		assert.Len(t, args, 1) // val2 should not be added
	})
}

func TestDiffVal(t *testing.T) {
	a1 := Value{Col: "col1", Val: "value1"}
	a2 := Value{Col: "col2", Val: nil}
	a3 := Value{Col: "col3", Val: 42}
	a4 := Value{Col: "col4", Val: 99.9}

	b1 := a1
	b2 := Value{Col: "col2", Val: "value2"}
	b3 := Value{Col: "col3", Val: nil}
	b5 := Value{Col: "col5", Val: 99.9}

	diff := DiffVals([]Value{a1}, []Value{b1})
	assert.Equal(t, 0, len(diff))

	diff = DiffVals([]Value{a1}, []Value{b2})
	assert.Equal(t, 2, len(diff))
	assert.Equal(t, `[{col1 value1 <nil>} {col2 <nil> value2}]`, fmt.Sprintf("%v", diff))

	diff = DiffVals([]Value{a1, a2, a3, a4}, []Value{b1, b2, b3, b5})
	assert.Equal(t, 4, len(diff), diff)
	assert.Equal(t, `[{col2 <nil> value2} {col3 42 <nil>} {col4 99.9 <nil>} {col5 <nil> 99.9}]`, fmt.Sprintf("%v", diff))
}
