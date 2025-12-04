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

func TestFieldToColumn(t *testing.T) {
	// Test struct mimicking Extension struct's tag structure
	type testStruct struct {
		EmId             int     `json:"emId" gorm:"column:em_id;primaryKey"`
		Name             string  `json:"name" gorm:"column:em_name"`
		Number           int     `json:"number" gorm:"column:em_number"`
		CallForwarding   bool    `json:"callForwarding" gorm:"column:em_call_forwarding"`
		Email            *string `json:"email" gorm:"column:em_email"`
		TenantId         int     `json:"tenantId" gorm:"column:tm_id"`
		Status           string  `json:"status" gorm:"column:em_status"`
		LastName         *string `json:"lastName" gorm:"column:em_last_name"`
		VoicemailService bool    `json:"voicemailService" gorm:"column:em_voicemail_service"`
	}

	tests := []struct {
		name        string
		jsonField   string
		expected    string
		expectError bool
	}{
		{
			name:        "primary key field",
			jsonField:   "emId",
			expected:    "em_id",
			expectError: false,
		},
		{
			name:        "string field",
			jsonField:   "name",
			expected:    "em_name",
			expectError: false,
		},
		{
			name:        "string field",
			jsonField:   "NamE",
			expected:    "em_name",
			expectError: false,
		},
		{
			name:        "integer field",
			jsonField:   "number",
			expected:    "em_number",
			expectError: false,
		},
		{
			name:        "boolean field",
			jsonField:   "callForwarding",
			expected:    "em_call_forwarding",
			expectError: false,
		},
		{
			name:        "nullable string field",
			jsonField:   "email",
			expected:    "em_email",
			expectError: false,
		},
		{
			name:        "tenant id foreign key",
			jsonField:   "tenantId",
			expected:    "tm_id",
			expectError: false,
		},
		{
			name:        "status enum field",
			jsonField:   "status",
			expected:    "em_status",
			expectError: false,
		},
		{
			name:        "last name field",
			jsonField:   "lastName",
			expected:    "em_last_name",
			expectError: false,
		},
		{
			name:        "voicemail service field",
			jsonField:   "voicemailService",
			expected:    "em_voicemail_service",
			expectError: false,
		},
		{
			name:        "invalid field name",
			jsonField:   "nonExistentField",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty field name",
			jsonField:   "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FieldToColumn[testStruct](tt.jsonField)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown field")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
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

func TestDbCols(t *testing.T) {
	type testStruct struct {
		A string `gorm:"column:aa" groups:"group1"`
		B string `gorm:"column:bb"`
		C string `gorm:"column:cc" groups:"group1,group2"`
		D string
	}
	tests := []struct {
		target   []string
		expected string
	}{
		{
			target:   []string{"group2"},
			expected: "prefix.bb, prefix.cc",
		},
		{
			target:   []string{"group1"},
			expected: "prefix.aa, prefix.bb, prefix.cc",
		},
		{
			target:   []string{"group3"},
			expected: "prefix.bb",
		},
		{
			target:   []string{"group1", "group2"},
			expected: "prefix.aa, prefix.bb, prefix.cc",
		},
		{
			target:   []string{"all"},
			expected: "prefix.aa, prefix.bb, prefix.cc",
		},
	}
	for _, test := range tests {
		cols := SqlSelectColumns[testStruct]("prefix.", test.target)
		assert.Equal(t, test.expected, cols, test.target)
	}
}
