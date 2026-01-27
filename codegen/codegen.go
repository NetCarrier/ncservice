package codegen

import (
	"encoding/json"
	"io"
	"os"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/jmoiron/sqlx"
	"github.com/stoewer/go-strcase"
)

type Options struct {
	Package string
	Crud    *CrudOptions
	Lookup  *LookupOptions
}

func ReadOptions(fname string) (Options, error) {
	return readConfig[Options](fname)
}

type CodeGen struct {
	opts   Options
	crud   *Cruder
	lookup *Lookuper
}

func (c *CodeGen) Run(opts Options, db *sqlx.DB) error {
	c.opts = opts
	if c.opts.Crud != nil {
		c.crud = &Cruder{}
		if err := c.crud.Run(*c.opts.Crud); err != nil {
			return err
		}
	}
	if c.opts.Lookup != nil {
		c.lookup = &Lookuper{}
		if err := c.lookup.Run(*c.opts.Lookup, db); err != nil {
			return err
		}
	}
	return nil
}

func (c *CodeGen) Write(templateFname string, out io.Writer) error {
	funcs := sprig.FuncMap()
	funcs["toLowerCamel"] = strcase.LowerCamelCase
	funcs["yangRange"] = YangRange

	tmpl, err := os.ReadFile(templateFname)
	if err != nil {
		return err
	}
	t, err := template.New("main").Funcs(funcs).Parse(string(tmpl))
	if err != nil {
		return err
	}
	return t.Execute(out, struct {
		Options Options
		Crud    *Cruder
		Lookup  *Lookuper
	}{
		Options: c.opts,
		Crud:    c.crud,
		Lookup:  c.lookup,
	})
}

func readConfig[T any](path string) (T, error) {
	var cfg T
	f, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(f, &cfg)
	return cfg, err
}
