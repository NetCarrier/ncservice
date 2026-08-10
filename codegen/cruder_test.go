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
		typedef email {
		     type string;
		}
		list x {
			leaf y {
				type t;
			}
			leaf-list z {
				type int32;
			}
			leaf og {
			   description "Original Gangster";
			   type string {
			      pattern "[A-Z+]";
				  length "3..5";
			   }
			}
			leaf n {
			  description "Number";
			  type int32 {
				  range "10..500";
			  }
			}
			leaf-list f {
			    type email;
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
	og := c.Entries[0].fields[2]
	assert.Equal(t, "Original Gangster. Supported regular expressions: [A-Z+]. Allowed string length: 3..5", og.Description())
	n := c.Entries[0].fields[3]
	assert.Equal(t, "Number. Allowed number ranges: 10..500", n.Description())
	f := c.Entries[0].fields[4]
	assert.Equal(t, "[]string", f.GoType())
}

func TestBsonTagsAndCollection(t *testing.T) {
	mstr := `module x {
		prefix "x";

		extension collection {
			argument "name";
		}

		extension scopes {
			argument "value";
		}

		extension col {
			argument "name";
		}

		list x {
			leaf y {
				type string;
			}
			leaf-list z {
				type int32;
			}
			leaf hidden {
				type string;
				x:scopes "create,update";
			}
			leaf renamed {
				type string;
				x:col "legacy_name";
			}
		}
		list withCollection {
			x:collection "mycoll";
			leaf id {
				type string;
			}
		}
	}
	`
	m, err := parser.LoadModuleFromString(nil, mstr)
	require.NoError(t, err)
	opts := CrudOptions{
		Entries: []CrudOptionsEntry{
			{Table: "x", Ydef: "x"},
			{Table: "withCollection", Ydef: "withCollection"},
		},
	}
	c := NewCruder(opts)
	require.NoError(t, c.read(m))

	y := c.Entries[0].fields[0]
	assert.Equal(t, y.JsonTags("read"), y.BsonTags("read"))
	assert.Equal(t, "y", y.BsonTags("read"))

	z := c.Entries[0].fields[1]
	assert.Equal(t, "z,omitempty", z.BsonTags("create"))

	hidden := c.Entries[0].fields[2]
	assert.Equal(t, "-", hidden.BsonTags("read"))

	renamed := c.Entries[0].fields[3]
	assert.Equal(t, "legacy_name", renamed.BsonTags("read"), "x:col override should apply to the bson tag too, not just gorm's")
	assert.Equal(t, "renamed", renamed.JsonTags("read"), "the json tag should be unaffected by x:col")

	assert.False(t, c.Entries[0].HasCollection())
	assert.Equal(t, "", c.Entries[0].Collection())

	assert.True(t, c.Entries[1].HasCollection())
	assert.Equal(t, "mycoll", c.Entries[1].Collection())
}
