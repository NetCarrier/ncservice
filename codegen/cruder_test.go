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
	assert.Equal(t, "string", c.Entries[0].fields[0].GoType())
}
