package codegen

import (
	"testing"

	"github.com/freeconf/yang/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrudItem(t *testing.T) {
	mstr := `module x {
		typedef t {
			type string;
		}
		list x {
			leaf y {
				type t;
			}
			leaf-list z {
				type int32;
			}
		}
	}
	`
	m, err := parser.LoadModuleFromString(nil, mstr)
	require.NoError(t, err)
	opts := CrudOptions{
		Entries: []CrudOptionsEntry{
			{
				Table: "x",
				Ydef:  "x",
			},
		},
	}
	c := NewCruder(opts)
	require.NoError(t, c.read(m))
	x := c.Entries[0].fields[0]
	assert.Equal(t, "string", x.GoType())
	assert.Equal(t, "required", x.BindingTags("create"))
	z := c.Entries[0].fields[1]
	assert.Equal(t, "[]int", z.GoType())
	assert.Equal(t, "omitempty", z.BindingTags("create"))
}
