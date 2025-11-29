package ncservice

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

type X struct {
	A string `json:"a" groups:"group1,group2"`
	B string `json:"b" groups:"group2"`
	C string `json:"c" groups:"group3"`
	D string `json:"d" `                 // no groups
	E string `json:"e" groups:"default"` // always included
}

func TestJsonFiltering(t *testing.T) {
	x := X{
		A: "test",
		B: "test2",
		C: "test3",
		D: "test4",
		E: "test5",
	}

	tests := []struct {
		target   []string
		expected string
	}{
		{
			target:   []string{"group1"},
			expected: `{"a":"test","e":"test5"}`,
		},
		{
			target:   []string{"group2"},
			expected: `{"a":"test","b":"test2","e":"test5"}`,
		},
		{
			target:   []string{"group3"},
			expected: `{"c":"test3","e":"test5"}`,
		},
		{
			target:   []string{"group1", "group3"},
			expected: `{"a":"test","c":"test3","e":"test5"}`,
		},
		{
			target:   []string{"x"},
			expected: `{"e":"test5"}`,
		},
		{
			target:   nil,
			expected: `{"e":"test5"}`,
		},
		{
			target:   []string{},
			expected: `{"e":"test5"}`,
		},
		{
			target:   []string{"group1", "group2", "group3"},
			expected: `{"a":"test","b":"test2","c":"test3","e":"test5"}`,
		},
		{
			target:   []string{"all"},
			expected: `{"a":"test","b":"test2","c":"test3","d":"test4","e":"test5"}`,
		},
		{
			target:   []string{"all", "group1"}, // group1 is redundant
			expected: `{"a":"test","b":"test2","c":"test3","d":"test4","e":"test5"}`,
		},
	}
	for _, test := range tests {
		actual, err := json.Marshal(&x, json.WithMarshalers(
			json.MarshalFunc(filterJsonByGroups[X](test.target)),
		))
		assert.NoError(t, err)
		assert.Equal(t, test.expected, string(actual))
	}
}

func TestJsonFilteringLists(t *testing.T) {
	xs := []X{
		{
			A: "test1",
			B: "test2",
			C: "test3",
			D: "test4",
			E: "test5",
		},
		{
			A: "test5",
			B: "test6",
			C: "test7",
			D: "test8",
			E: "test9",
		},
	}

	actual, err := MarshalJsonList(xs, "group2")
	assert.NoError(t, err)
	expected := `[{"a":"test1","b":"test2","e":"test5"},{"a":"test5","b":"test6","e":"test9"}]`
	assert.Equal(t, expected, string(actual))
}
