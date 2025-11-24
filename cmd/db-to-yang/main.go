package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/stoewer/go-strcase"
)

var mssqlArg = flag.String("mssql", "", "DB connection string for SQL Server")
var mysqlArg = flag.String("mysql", "", "DB connection string for MySQL")

var tbl = flag.String("table", "", "Table name to generate YANG for.")
var colsArg = flag.Bool("cols", false, "Add column annotations.")

func main() {
	flag.Parse()
	var rdr DbReader
	if *mssqlArg != "" {
		pool, err := sqlx.Open("sqlserver", *mssqlArg)
		chkerr(err)
		defer pool.Close()
		rdr = &SqlServer{pool: pool}
	} else if *mysqlArg != "" {
		pool, err := sqlx.Open("mysql", *mysqlArg)
		chkerr(err)
		defer pool.Close()
		rdr = &MySql{pool: pool}
	} else {
		log.Fatal("one of mssql or mysql arg required")
	}

	if *tbl == "" {
		log.Fatal("table arg required")
	}

	data, err := ReadDb(rdr, *tbl)
	chkerr(err)
	chkerr(WriteYang(data, os.Stdout))
}

type DbReader interface {
	Rows(table string) (*sqlx.Rows, error)
}

type SqlServer struct {
	pool *sqlx.DB
}

func (d *SqlServer) Rows(table string) (*sqlx.Rows, error) {
	sqlstr := fmt.Sprintf(`select 
		COLUMN_NAME as Name, 
		DATA_TYPE as DataType,
		IS_NULLABLE as RawIsNullable
		from INFORMATION_SCHEMA.COLUMNS where TABLE_NAME = '%s'`, table)
	return d.pool.Queryx(sqlstr)
}

type MySql struct {
	pool *sqlx.DB
}

func (d *MySql) Rows(table string) (*sqlx.Rows, error) {
	sqlstr := fmt.Sprintf(`select 
		COLUMN_NAME as Name, 
    	DATA_TYPE as DataType, 
    	IS_NULLABLE as RawIsNullable, 
    	COLUMN_TYPE as ColumnType,
    	COLUMN_DEFAULT as DefaultValue,
		COLUMN_COMMENT as Description,
		COLUMN_KEY as ColumnKey
    	from INFORMATION_SCHEMA.COLUMNS where TABLE_NAME = '%s'`, table)
	return d.pool.Queryx(sqlstr)
}

func WriteYang(data tableData, out *os.File) error {
	var err error
	tmpl := template.New("yang")
	tmpl.Funcs(template.FuncMap{
		"camel": strcase.LowerCamelCase,

		// supports nil and notnil as of now
		"is": func(typ string, val any) bool {
			rv := reflect.ValueOf(val)
			if rv.Kind() == reflect.Pointer && rv.IsNil() {
				return typ == "nil"
			}
			// Pull actual value out from pointer
			for rv.Kind() == reflect.Pointer {
				rv = rv.Elem()
			}

			switch rv.Kind() {
			case reflect.String:
				return (typ == "nil" && rv.String() == "") || (typ == "notnil" && rv.String() != "")
			default:
				return typ == "nil"
			}
		},
	})
	tmpl, err = tmpl.Parse(`
	list {{.Name | camel }} {
		description "Auto-generated from table {{.Name}}";
		
		x:table "{{.Name}}";
		{{- if is "notnil" .PrimaryKey }}
		key {{ .PrimaryKey }};{{ end }}
		{{range .Columns}}
		leaf {{.Name | camel }} {
			{{- if .IsEnum }}
			type enumeration {
				{{- range .EnumValues }}
				enum {{ . }};{{ end }}
			}
			{{- else }}
			type {{.YangType}};
			{{- end }}
			{{- if is "notnil" .Description }}
			description "{{ .Description }}";{{ end }}
			{{- if or .IsNullable (is "notnil" .DefaultValue) }}
			x:nullable;{{ end }}
			{{- if $.ShowCols }}
			{{- if is "notnil" .DefaultValue }}
			default {{ .DefaultValue }};{{ end }}
			x:col "{{.Name}}";{{ end }}
		}
		{{end}}
	}
`)
	if err != nil {
		return err
	}
	return tmpl.Execute(out, data)
}

func ReadDb(db DbReader, table string) (tableData, error) {
	data := tableData{
		Name:     table,
		ShowCols: *colsArg,
	}
	rows, err := db.Rows(table)
	if err != nil {
		return data, err
	}
	defer rows.Close()

	for rows.Next() {
		var col column
		if err := rows.StructScan(&col); err != nil {
			return data, err
		}
		data.Columns = append(data.Columns, col)
	}
	return data, nil
}

type column struct {
	Name            string  `db:"Name"`
	DataType        string  `db:"DataType"`
	RawIsNullable   string  `db:"RawIsNullable"`
	ColumnType      string  `db:"ColumnType"`
	DefaultValueRaw *string `db:"DefaultValue"`
	Description     *string `db:"Description"`
	ColumnKey       *string `db:"ColumnKey"`
}

func (c *column) IsNullable() bool {
	return c.RawIsNullable == "YES"
}

type tableData struct {
	Name     string
	Columns  []column
	ShowCols bool
}

func (c column) YangType() string {
	var ytype string
	switch c.DataType {
	case "int", "bigint", "smallint", "tinyint":
		ytype = "int32"
	case "bit":
		ytype = "boolean"
	case "varchar", "nvarchar", "text", "ntext", "char", "nchar":
		ytype = "string"
	case "datetime", "datetime2", "smalldatetime", "date":
		ytype = "dateTime"
	case "time":
		if mysqlArg != nil {
			ytype = "string"
		} else {
			ytype = "dateTime"
		}
	case "float", "real", "decimal", "numeric", "money", "smallmoney":
		ytype = "decimal64"
	default:
		ytype = "string"
	}
	return ytype
}

func (c *column) DefaultValue() string {
	if c.DefaultValueRaw == nil {
		return ""
	}
	switch c.YangType() {
	case "string", "dateTime":
		return fmt.Sprintf("\"%s\"", *c.DefaultValueRaw)
	default:
		return *c.DefaultValueRaw
	}
}

func (t tableData) PrimaryKey() string {
	if len(t.Columns) == 0 {
		return ""
	}
	for _, col := range t.Columns {
		if col.ColumnKey != nil && *col.ColumnKey == "PRI" {
			return strcase.LowerCamelCase(col.Name)
		}
	}
	return ""
}

func (c *column) IsEnum() bool {
	return c.DataType == "enum"
}

func (c *column) EnumValues() []string {
	if c.DataType != "enum" {
		return nil
	}
	var vals []string
	// COLUMN_TYPE is like: enum('value1','value2','value3')
	ct := c.ColumnType
	if len(ct) < 6 {
		return vals
	}
	ct = ct[5 : len(ct)-1] // strip "enum(" and ")"
	vals = append(vals, splitEnumValues(ct)...)
	return vals
}

func splitEnumValues(ct string) []string {
	var vals []string
	for part := range strings.SplitSeq(ct, ",") {
		if len(part) >= 2 && part[0] == '\'' && part[len(part)-1] == '\'' {
			vals = append(vals, part[1:len(part)-1])
		}
	}
	return vals
}

func chkerr(err error) {
	if err != nil {
		panic(err)
	}
}
