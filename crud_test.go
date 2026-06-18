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
		NoGorm           string  `json:"noGorm"` // No gorm tag
		NoJson           string  `gorm:"column:no_json"`
		NoAnything       string
		IgnoredJson      string `json:"-"`
		OtherTable       string `json:"otherTable" gorm:"column:this_col" table:"another_table"`
	}

	tests := []struct {
		name          string
		jsonField     string
		expected      string
		expectedTable string
		expectError   bool
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
			name:        "case insensitive field matching",
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
		{
			name:        "Not a db field",
			jsonField:   "noGorm",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Not a json field",
			jsonField:   "NoJson",
			expected:    "",
			expectError: true,
		},
		{
			name:        "No anything",
			jsonField:   "NoAnything",
			expected:    "",
			expectError: true,
		},
		{
			name:        "ignored json",
			jsonField:   "IgnoredJson",
			expected:    "",
			expectError: true,
		},
		{
			name:          "table field",
			jsonField:     "OtherTable",
			expected:      "this_col",
			expectedTable: "another_table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, tbl, err := FieldToColumn[testStruct](tt.jsonField)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
				assert.Equal(t, tt.expectedTable, tbl)
			}
		})
	}
}

func TestValues(t *testing.T) {
	type testStruct struct {
		ID       int    `gorm:"column:id"`
		Name     string `gorm:"column:name"`
		Ignore   string
		Age      int      `gorm:"column:age"`
		FavColor *string  `gorm:"column:color"`
		FavSport *string  `gorm:"column:sport"`
		Teachers []string `gorm:"column:teachers"`
	}
	var red = "red"
	ts := testStruct{
		ID:       1,
		Name:     "Alice",
		Ignore:   "should be ignored",
		Age:      30,
		FavColor: &red,
	}

	values, err := Values(ts, func(v Value, _ reflect.StructField) bool {
		return v.Col != "age" // Filter out age
	})
	assert.NoError(t, err)

	assert.Equal(t, 5, len(values))

	assert.Equal(t, "id", values[0].Col)
	assert.EqualValues(t, 1, values[0].Val)
	assert.EqualValues(t, "Alice", values[1].Val)
	assert.EqualValues(t, &red, values[2].Val)
	assert.EqualValues(t, nil, values[3].Val)
	assert.EqualValues(t, []string(nil), values[4].Val)

	all, err := Values(&ts, nil)
	assert.NoError(t, err)
	assert.Equal(t, 6, len(all))
}

func TestSetValues(t *testing.T) {

	type testStruct struct {
		ID       int    `gorm:"column:id"`
		Name     string `gorm:"column:name"`
		Ignore   string
		Age      int      `gorm:"column:age"`
		FavColor *string  `gorm:"column:color"`
		FavSport *string  `gorm:"column:sport"`
		Strs     []string `gorm:"column:strs"`
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
		{Col: "strs"}, // testing we *don't set* anything here
	}

	ts2 := testStruct{
		Strs: []string{"a", "b"},
	}
	SetValues(&ts2, valuesToSet)

	assert.Equal(t, 2, ts2.ID)
	assert.Equal(t, "Bob", ts2.Name)
	assert.Equal(t, 25, ts2.Age)
	assert.Equal(t, "red", *ts2.FavColor)
	assert.NotNil(t, ts2.FavSport)
	assert.Equal(t, "soccer", *ts2.FavSport)
	assert.EqualValues(t, "", ts2.Ignore)
	assert.EqualValues(t, []string{"a", "b"}, ts2.Strs)
}
