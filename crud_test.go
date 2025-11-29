package ncservice

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGormTag(t *testing.T) {
	tests := []struct {
		tag    string
		part   string
		exists bool
		match  string
	}{
		{
			tag:    "column:id;primaryKey",
			part:   "column",
			exists: true,
			match:  "id",
		},
		{
			tag:    "",
			part:   "column",
			exists: false,
		},
		{
			tag:    "column:x",
			part:   "column",
			exists: true,
			match:  "x",
		},
		{
			tag:    "column",
			part:   "column",
			exists: true,
			match:  "",
		},
		{
			tag:    "",
			part:   "column",
			exists: false,
			match:  "",
		},
	}
	for _, test := range tests {
		value, exists := getGormTag(test.tag, test.part)
		assert.Equal(t, test.exists, exists, test.tag)
		assert.Equal(t, test.match, value)
	}
}

func TestValues(t *testing.T) {
	type testStruct struct {
		ID       int    `gorm:"column:id"`
		Name     string `gorm:"column:name"`
		Ignore   string
		Age      int     `gorm:"column:age"`
		FavColor *string `gorm:"column:color"`
		FavSport *string `gorm:"column:sport"`
	}
	var red = "red"
	ts := testStruct{
		ID:       1,
		Name:     "Alice",
		Ignore:   "should be ignored",
		Age:      30,
		FavColor: &red,
	}

	values := Values(ts, func(v Value, _ reflect.StructField) bool {
		return v.Col != "age" // Filter out age
	})

	assert.Equal(t, 4, len(values))

	assert.Equal(t, "id", values[0].Col)
	assert.EqualValues(t, 1, values[0].Val)
	assert.EqualValues(t, "Alice", values[1].Val)
	assert.EqualValues(t, &red, values[2].Val)
	assert.EqualValues(t, nil, values[3].Val)

	all := Values(&ts, nil)
	assert.Equal(t, 5, len(all))
}

func TestSetValues(t *testing.T) {

	type testStruct struct {
		ID       int    `gorm:"column:id"`
		Name     string `gorm:"column:name"`
		Ignore   string
		Age      int     `gorm:"column:age"`
		FavColor *string `gorm:"column:color"`
		FavSport *string `gorm:"column:sport"`
	}
	var red = "red"
	var bob = "Bob"

	valuesToSet := []Value{
		{Col: "id", Val: 2},
		{Col: "name", Val: &bob},
		{Col: "age", Val: 25},
		{Col: "ignore", Val: "bingo"},
		{Col: "color", Val: &red},
		{Col: "sport", Val: "soccer"},
	}

	var ts2 testStruct
	SetValues(&ts2, valuesToSet)

	assert.Equal(t, 2, ts2.ID)
	assert.Equal(t, "Bob", ts2.Name)
	assert.Equal(t, 25, ts2.Age)
	assert.Equal(t, "red", *ts2.FavColor)
	assert.NotNil(t, ts2.FavSport)
	assert.Equal(t, "soccer", *ts2.FavSport)
	assert.EqualValues(t, "", ts2.Ignore)
}
