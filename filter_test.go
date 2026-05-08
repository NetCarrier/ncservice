package ncservice

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	s := struct {
		Field1 string `gorm:"column:col1"`
		Field2 *int   `gorm:"column:col2"`
		Field3 int    `gorm:"column:col3"`
		Field4 []int  `gorm:"column:col4"`
	}{
		Field1: "value1",
		Field2: nil,
		Field3: 42,
		Field4: []int{},
	}

	t.Run("FilterAll", func(t *testing.T) {
		vals := Values(s, FilterAll)
		assert.Len(t, vals, 4)
	})

	t.Run("FilterNotNil", func(t *testing.T) {
		vals := Values(s, FilterNotNil)
		assert.Len(t, vals, 2)
		assert.Equal(t, "col1", vals[0].Col)
		assert.Equal(t, "col3", vals[1].Col)
	})

	t.Run("FilterAnd", func(t *testing.T) {
		andFilter := FilterAnd(FilterNotNil, func(v Value, fld reflect.StructField) bool {
			return v.Col != "col3"
		})
		vals := Values(s, andFilter)
		assert.Len(t, vals, 1)
		assert.Equal(t, "col1", vals[0].Col)
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
