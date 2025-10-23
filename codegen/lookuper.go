package codegen

import (
	"fmt"
	"io"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/jmoiron/sqlx"
	"github.com/stoewer/go-strcase"
)

// lookuper helps generate lookup code based on DB tables
type Lookuper struct {
	db   *sqlx.DB
	opts LookupOptions
}

func NewLookuper(db *sqlx.DB, opts LookupOptions) *Lookuper {
	return &Lookuper{
		db:   db,
		opts: opts,
	}
}

type LookupOptions struct {
	Template string
	Lookups  []LookupEntryOptions
}

type lookup struct {
	Name    string
	Options LookupEntryOptions
	Entries []lookupEntry
	Fields  []lookupField
}

func (l lookup) GoType() string {
	return l.Id().GoType()
}

func (l lookup) Id() lookupField {
	for _, e := range l.Fields {
		if e.Name == l.Options.IdColumn {
			return e
		}
	}
	panic("id column not found")
}

func (l lookup) Value() lookupField {
	for _, e := range l.Fields {
		if e.Name == l.Options.ValueColumn {
			return e
		}
	}
	panic("value column not found")
}

type lookupEntry struct {
	parent lookup
	Field  lookupField
	Values []lookupFieldValue
}

func (le lookupEntry) Id() lookupFieldValue {
	idFields := le.parent.Id()
	for _, v := range le.Values {
		if v.Field.Name == idFields.Name {
			return v
		}
	}
	panic("id field not found")
}

func (le lookupEntry) GoLabel() string {
	n := le.parent.Name
	s := strcase.UpperCamelCase(fmt.Sprintf("%v", le.Label().Value))
	if le.parent.Options.Overrides != nil {
		id := fmt.Sprintf("%v", le.Id().Value)
		if ov, ok := le.parent.Options.Overrides[id]; ok {
			s = ov
		}
	}
	return n + s
}

func (le lookupEntry) Label() lookupFieldValue {
	idFields := le.parent.Value()
	for _, v := range le.Values {
		if v.Field.Name == idFields.Name {
			return v
		}
	}
	panic("value field not found")
}

type lookupFieldValue struct {
	Field lookupField
	Value any
}

func (lv lookupFieldValue) GoValue() string {
	switch lv.Field.GoType() {
	case "string":
		if lv.Value == nil {
			return `""`
		}
		return fmt.Sprintf("%q", lv.Value)
	default:
		return fmt.Sprintf("%v", lv.Value)
	}
}

type lookupField struct {
	Name string
	Type string
}

func (f lookupField) GoType() string {
	return GoType(f.Type)
}

type LookupEntryOptions struct {
	Description string
	Table       string
	IdColumn    string
	ValueColumn string
	Overrides   map[string]string
}

func (l *Lookuper) Run(out io.Writer) error {
	items, err := l.read()
	if err != nil {
		return err
	}
	return l.write(out, items)
}

func (l *Lookuper) read() ([]lookup, error) {
	var items []lookup
	for _, opt := range l.opts.Lookups {
		item := lookup{
			Name:    opt.Table,
			Options: opt,
		}
		sqlstr := fmt.Sprintf(`select * from %s`, opt.Table)
		rows, err := l.db.Queryx(sqlstr)
		if err != nil {
			return nil, fmt.Errorf("bad query '%s'. %v", sqlstr, err)
		}
		defer rows.Close()

		types, err := rows.ColumnTypes()
		if err != nil {
			return nil, fmt.Errorf("failure to get column types for table %s. %v", opt.Table, err)
		}
		for _, ct := range types {
			fld := lookupField{
				Name: ct.Name(),
				Type: ct.DatabaseTypeName(),
			}
			item.Fields = append(item.Fields, fld)
		}

		for rows.Next() {
			entry := lookupEntry{
				parent: item,
			}
			var row []any = make([]any, len(types))
			row, err := rows.SliceScan()
			if err != nil {
				return nil, fmt.Errorf("failure to read row . %v", err)
			}
			for col, val := range row {
				fldVal := lookupFieldValue{
					Field: item.Fields[col],
					Value: val,
				}
				entry.Values = append(entry.Values, fldVal)
			}
			item.Entries = append(item.Entries, entry)
		}
		items = append(items, item)
	}

	return items, nil
}

func (l *Lookuper) write(out io.Writer, entries []lookup) error {
	funcs := sprig.FuncMap()
	funcs["toLowerCamel"] = strcase.LowerCamelCase
	tmpl := template.New(filepath.Base(l.opts.Template))
	t, err := tmpl.Funcs(funcs).ParseFiles(l.opts.Template)
	if err != nil {
		return err
	}
	return t.Execute(out, struct{ Lookups []lookup }{Lookups: entries})
}
