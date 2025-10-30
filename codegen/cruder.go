package codegen

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/NetCarrier/ncservice"
	"github.com/NetCarrier/telapia/utils"
	"github.com/freeconf/yang/meta"
	"github.com/freeconf/yang/parser"
	"github.com/freeconf/yang/source"
	"github.com/freeconf/yang/val"
	"github.com/stoewer/go-strcase"
)

// cruder helps generate CRUD code based on YANG definitions
type Cruder struct {
	opts      CrudOptions
	enumTypes map[string]Enum
	items     []crudItem
	rootDef   *meta.Module
}

func NewCruder(opts CrudOptions) *Cruder {
	return &Cruder{
		opts:      opts,
		enumTypes: make(map[string]Enum),
	}
}

type CrudOptions struct {
	Package    string
	YangPath   string
	YangModule string
	Entries    []CrudOptionsEntry
	SnakeCase  bool
}

type CrudOptionsEntry struct {
	Table string
	Ydef  string
}

type crudItem struct {
	Parent *Cruder
	Def    *meta.List
	fields []crudField
}

func (c *Cruder) Run(out io.Writer) error {
	ypath := source.Path(c.opts.YangPath)
	m, err := parser.LoadModule(ypath, c.opts.YangModule)
	if err != nil {
		return err
	}
	if err = c.read(m); err != nil {
		return err
	}
	return c.write(out, c.items)
}

// TargetField finds the field definition for the given path. Useful for
// leafRefs that point to another definition.
func (c *Cruder) resolveFieldPath(target meta.Leafable) crudField {
	for _, i := range c.items {
		for _, f := range i.fields {
			if f.Def == target {
				return f
			}
		}
	}
	panic(fmt.Sprintf("crud item not found: %s", target))
}

func (c *Cruder) read(m *meta.Module) error {
	c.rootDef = m
	for _, e := range c.opts.Entries {
		entry := crudItem{
			Parent: c,
		}
		var valid bool
		entry.Def, valid = meta.Find(m, e.Ydef).(*meta.List)
		if !valid {
			return fmt.Errorf("invalid YANG definition: %s", e.Ydef)
		}
		for _, f := range entry.Def.DataDefinitions() {
			f := crudField{
				Parent: entry,
				Def:    f.(meta.Leafable),
			}
			entry.fields = append(entry.fields, f)
		}

		c.items = append(c.items, entry)
	}
	return nil
}

func (c *Cruder) write(out io.Writer, entries []crudItem) error {
	funcs := sprig.FuncMap()
	funcs["toLowerCamel"] = strcase.LowerCamelCase

	tmpl, err := internal.ReadFile("crud.go.tpl")
	if err != nil {
		return err
	}
	t, err := template.New("main").Funcs(funcs).Parse(string(tmpl))
	if err != nil {
		return err
	}
	return t.Execute(out, struct {
		Cruds   []crudItem
		Options CrudOptions
		Enums   map[string]Enum
	}{
		Cruds:   entries,
		Options: c.opts,
		Enums:   c.enumTypes,
	})
}

const (
	FieldCritNoKeys   = "nokeys"   // list def does not designate it as a key
	FieldCritKeys     = "keys"     // list def designates it as a key, only keys
	FieldCritEditable = "editable" // noedit is missing
)

func (f crudField) GormTags() string {
	tags := []string{
		"column:" + f.Col(),
	}
	if f.IsKey() {
		tags = append(tags, "primaryKey")
	}
	custom := getExtention(f.Def, "serializer", "")
	if custom != "" {
		tags = append(tags, "serializer:"+custom)
	}
	if f.DefaultValue() != nil {
		tags = append(tags, "default:"+*f.DefaultValue())
	}
	return strings.Join(tags, ";")
}

func (f crudField) BindingTags(typ string) string {
	tags := []string{}
	// TODO: Currently assuming all optional to test other type of validations.
	// will be fixed once binding validations tags will be implemented and tested properly
	tags = append(tags, "omitempty")
	// if f.IsNullable() || typ == "update" {
	// 	tags = append(tags, "omitempty")
	// } else {
	// 	tags = append(tags, "required")
	// }
	if f.isEnum() {
		// Add oneof tag for validation
		values := []string{}
		for _, val := range f.getEnumValues() {
			if val.Parent.GoType() == "string" {
				values = append(values, fmt.Sprintf("'%s'", val.Def.Ident()))
			} else {
				values = append(values, fmt.Sprintf("%d", val.Def.Value()))
			}
		}
		oneofValues := strings.Join(values, " ")
		tags = append(tags, fmt.Sprintf("oneof=%v", oneofValues))
	}
	return strings.Join(tags, ",")
}

func (v crudItem) Fields(crit ...string) []crudField {
	var out []crudField
	for _, f := range v.fields {
		for _, c := range crit {
			if c == FieldCritKeys && !f.IsKey() {
				goto skip
			}
			if c == FieldCritNoKeys && f.IsKey() {
				goto skip
			}
			if c == FieldCritEditable && !f.IsEditable() {
				goto skip
			}
		}
		out = append(out, f)
	skip:
	}

	return out
}

func (v crudItem) HasLastModified() bool {
	for _, f := range v.fields {
		if f.Def.Ident() == "lastModified" {
			return true
		}
	}
	return false
}

type crudField struct {
	Parent crudItem
	Def    meta.Leafable
}

func (f crudField) IsKey() bool {
	for _, k := range f.Parent.Def.KeyMeta() {
		if k.Ident() == f.Def.Ident() {
			return true
		}
	}
	return false
}

func (f crudField) Deref() string {
	if f.IsNullable() {
		return "*"
	}
	return ""
}

func (f crudField) Ref() string {
	if f.IsNullable() {
		return "&"
	}
	return ""
}

func (f crudField) Name() string {
	return strcase.UpperCamelCase(f.Def.Ident())
}

func (f crudField) Col() string {
	var col string
	if f.Parent.Parent.opts.SnakeCase {
		col = ncservice.SnakeCase(f.Def.Ident())
	} else {
		col = strcase.UpperCamelCase(f.Def.Ident())
	}

	return getExtention(f.Def, "col", col)
}

func getExtention(def meta.Definition, extName string, defaultValue string) string {
	x := meta.FindExtension(extName, def.Extensions())
	if x != nil {
		return x.Argument()
	}
	return defaultValue
}

func (f crudField) IsEditable() bool {
	return meta.FindExtension("noedit", f.Def.Extensions()) == nil
}

func (f crudField) IsNullable() bool {
	return meta.FindExtension("nullable", f.Def.Extensions()) != nil
}

func (f crudField) GoType() string {
	t := f.GoRawType()
	if f.IsNullable() {
		t = "*" + t
	}
	return t
}

func (f crudField) isEnum() bool {
	return f.Parent.Parent.enumTypes[f.GoRawType()].Name != ""
}

func (f crudField) getEnumValues() []EnumValue {
	return f.Parent.Parent.enumTypes[f.GoRawType()].Values()
}

func (f crudField) getEnumType() string {
	e := Enum{
		Def: f.Def.Type(),
	}
	if f.Def.Type().Ident() == "enumeration" {
		e.Name = f.Name() + "Enum"
		e.Prefix = f.Name()
	} else {
		e.Name = strcase.UpperCamelCase(f.Def.Type().Ident())
		e.Prefix = e.Name
	}
	f.Parent.Parent.enumTypes[e.Name] = e
	return e.Name
}

func (f crudField) DefaultValue() *string {
	for _, ext := range f.Def.Extensions() {
		if ext.Ident() == "default" {
			if f.GoType() == "string" || f.GoType() == "*string" {
				return utils.Ptr(fmt.Sprintf("'%s'", ext.Argument()))
			}
			return utils.Ptr(ext.Argument())
		}
	}
	return nil
}

func (f crudField) GoTypePtr() string {
	return "*" + f.GoRawType()
}

func (f crudField) GoRawType() string {
	t := f.Def.Type().Resolve()
	var goType string
	if t.Format() == val.FmtEnum {
		goType = f.getEnumType()
	} else {
		typeIdent := t.Ident()
		switch typeIdent {
		case "int32":
			goType = "int"
		case "dateTime":
			goType = "time.Time"
		case "boolean":
			goType = "bool"
		default:
			goType = typeIdent
		}
	}

	if _, ok := f.Def.(*meta.LeafList); ok {
		goType = "[]" + goType
	}
	return goType
}

func (f crudField) ForeignKeyTag() string {
	if path := f.Def.Type().Path(); path != "" {
		targetDef := meta.Find(f.Def, path).(meta.Leafable)
		targetField := f.Parent.Parent.resolveFieldPath(targetDef)
		return fmt.Sprintf(`fk:"%s.%s"`, targetField.Parent.Table(), targetField.Col())
	}
	return "" // not a foreign key
}

func (v crudItem) Table() string {
	return getExtention(v.Def, "table", strcase.UpperCamelCase(v.Def.Ident()))
}

func (v crudItem) Struct() string {
	return getExtention(v.Def, "struct", strcase.UpperCamelCase(v.Def.Ident()))
}

type Enum struct {
	Name   string
	Prefix string
	Def    *meta.Type
}

type EnumValue struct {
	Parent *Enum
	Def    *meta.Enum
}

func (ev EnumValue) Ident() string {
	return strcase.UpperCamelCase(ev.Def.Ident())
}

func (ev EnumValue) Value() string {
	if ev.Parent.GoType() == "string" {
		return fmt.Sprintf("%q", ev.Def.Ident())
	}
	return fmt.Sprintf("%d", ev.Def.Value())
}

func (e Enum) Values() []EnumValue {
	var out []EnumValue
	for _, v := range e.Def.Enums() {
		out = append(out, EnumValue{
			Parent: &e,
			Def:    v,
		})
	}
	return out
}

func (e Enum) GoType() string {
	x := meta.FindExtension("enumAsInt", e.Def.Extensions())
	if x != nil {
		return "int"
	}
	return "string"
}
