package codegen

import (
	"fmt"
	"io"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/NetCarrier/ncservice"
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
	entries, err := c.read(m)
	if err != nil {
		return err
	}
	return c.write(out, entries)
}

func (c *Cruder) read(m *meta.Module) ([]crudItem, error) {
	var entries []crudItem
	for _, e := range c.opts.Entries {
		entry := crudItem{
			Parent: c,
		}
		var valid bool
		entry.Def, valid = meta.Find(m, e.Ydef).(*meta.List)
		if !valid {
			return nil, fmt.Errorf("invalid YANG definition: %s", e.Ydef)
		}
		for _, f := range entry.Def.DataDefinitions() {
			f := crudField{
				Parent: entry,
				Def:    f.(meta.Leafable),
			}
			entry.fields = append(entry.fields, f)
		}

		entries = append(entries, entry)
	}
	return entries, nil
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

func (f crudField) GoTypePtr() string {
	return "*" + f.GoRawType()
}

func (f crudField) GoRawType() string {
	if f.Def.Type().Format() == val.FmtEnum {
		return f.getEnumType()
	}

	typeIdent := f.Def.Type().Ident()
	switch typeIdent {
	case "int32":
		return "int"
	case "dateTime":
		return "time.Time"
	case "boolean":
		return "bool"
	}
	return typeIdent
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
