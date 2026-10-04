package codegen

import (
	"fmt"
	"strings"

	"github.com/freeconf/yang/meta"
	"github.com/stoewer/go-strcase"
)

// crudContainer is a YANG container inside a list. It generates its own Go
// struct, referenced from the parent struct by pointer, so a nested Mongo
// document (e.g. {"rfc6035": {"QualityEst": {"MOSLQ": 4.1}}}) decodes as-is.
//
// Containers are only supported on Mongo-backed (x:collection) lists, since a
// nested struct has no meaning for a SQL table, and only for reading for now:
// nested fields are not yet part of the generated Create/Update/Criteria structs.
type crudContainer struct {
	Parent     crudItem
	Def        *meta.Container
	structName string
	fields     []crudField
	containers []crudContainer
}

func newCrudContainer(entry crudItem, parentStruct string, def *meta.Container) (crudContainer, error) {
	where := fmt.Sprintf("container %q in list %q", def.Ident(), entry.Def.Ident())
	if !entry.HasCollection() {
		return crudContainer{}, fmt.Errorf("%s: containers are only supported in Mongo-backed lists (x:collection)", where)
	}
	if !isReadOnly(def) {
		// Create/Update/Criteria structs are built from the list's own leaves, so a
		// writable container would generate structs that silently drop its fields.
		// Lifting this needs nested Create/Update structs plus a Mongo write path that
		// sets nested fields by dotted path (e.g. "details.text") without replacing
		// the whole subdocument.
		return crudContainer{}, fmt.Errorf(`%s: create and update of nested fields are not supported yet, add x:scopes "read"`, where)
	}
	c := crudContainer{
		Parent:     entry,
		Def:        def,
		structName: getExtension(def, "struct", parentStruct+strcase.UpperCamelCase(def.Ident())),
	}
	for _, child := range def.DataDefinitions() {
		switch d := child.(type) {
		case meta.Leafable:
			c.fields = append(c.fields, crudField{Parent: entry, Def: d, nested: true})
		case *meta.Container:
			sub, err := newCrudContainer(entry, c.structName, d)
			if err != nil {
				return crudContainer{}, err
			}
			c.containers = append(c.containers, sub)
		default:
			return crudContainer{}, fmt.Errorf("%s: %q is not supported inside a container, only leaf, leaf-list and container", where, child.Ident())
		}
	}
	return c, nil
}

// isReadOnly reports whether x:scopes on the definition grants read and
// nothing else. No x:scopes means the default "create,read,update".
func isReadOnly(def meta.Definition) bool {
	scopes := getExtension(def, "scopes", "create,read,update")
	for _, s := range strings.Split(scopes, ",") {
		if strings.TrimSpace(s) != "read" {
			return false
		}
	}
	return true
}

// Name is the Go field name the parent struct uses for this container.
func (c crudContainer) Name() string {
	return strcase.UpperCamelCase(c.Def.Ident())
}

// Struct is the generated Go type name. Defaults to the parent struct name plus
// the container name; x:struct on the container overrides it.
func (c crudContainer) Struct() string {
	return c.structName
}

// GoType is a pointer so a section missing from the document decodes as nil.
func (c crudContainer) GoType() string {
	return "*" + c.structName
}

func (c crudContainer) JsonTags() string {
	return strcase.LowerCamelCase(c.Def.Ident()) + ",omitempty"
}

// BsonTags uses x:col for the document key when the YANG name differs from the
// stored one (e.g. qualityEst stored as "QualityEst").
func (c crudContainer) BsonTags() string {
	return getExtension(c.Def, "col", strcase.LowerCamelCase(c.Def.Ident())) + ",omitempty"
}

func (c crudContainer) Description() string {
	return c.Def.Description()
}

func (c crudContainer) JsonSchemaTag() string {
	if desc := c.Description(); desc != "" {
		return fmt.Sprintf(` jsonschema:"%s"`, desc)
	}
	return ""
}

// Fields returns the container's leaves and leaf-lists, in YANG order.
func (c crudContainer) Fields() []crudField {
	return c.fields
}

// Containers returns the containers directly inside this one, in YANG order.
func (c crudContainer) Containers() []crudContainer {
	return c.containers
}

// Containers returns the list's direct child containers, in YANG order.
func (v crudItem) Containers() []crudContainer {
	return v.containers
}

// NestedStructs returns every container under the list, depth first, so a
// template can declare one Go struct per container without recursion.
func (v crudItem) NestedStructs() []crudContainer {
	var out []crudContainer
	var walk func([]crudContainer)
	walk = func(cs []crudContainer) {
		for _, c := range cs {
			out = append(out, c)
			walk(c.containers)
		}
	}
	walk(v.containers)
	return out
}

// checkNestedStructNames fails when a generated nested struct name collides
// with a list struct or another nested struct, which would not compile.
func (c *Cruder) checkNestedStructNames() error {
	taken := make(map[string]bool, len(c.Entries))
	for _, e := range c.Entries {
		taken[e.Struct()] = true
	}
	for _, e := range c.Entries {
		for _, n := range e.NestedStructs() {
			if taken[n.Struct()] {
				return fmt.Errorf("container %q in list %q: struct name %q is already used, set x:struct on the container to pick another", n.Def.Ident(), e.Def.Ident(), n.Struct())
			}
			taken[n.Struct()] = true
		}
	}
	return nil
}
