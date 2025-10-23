package ncservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrcase(t *testing.T) {
	assert.Equal(t, "my_field_name", SnakeCase("MyFieldName"))
	assert.Equal(t, "e9_id", SnakeCase("E9Id"))
}
