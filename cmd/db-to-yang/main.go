package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/stoewer/go-strcase"
)

var mssqlArg = flag.String("mssql", "", "DB connection string for SQL Server")
var mysqlArg = flag.String("mysql", "", "DB connection string for MySQL")

var tbl = flag.String("table", "", "Table name to generate YANG for.")

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
		IS_NULLABLE as RawIsNullable
		from INFORMATION_SCHEMA.COLUMNS where TABLE_NAME = '%s'`, table)
	return d.pool.Queryx(sqlstr)
}

func WriteYang(data tableData, out *os.File) error {
	var err error
	tmpl := template.New("yang")
	tmpl.Funcs(template.FuncMap{
		"camel": strcase.LowerCamelCase,
	})
	tmpl, err = tmpl.Parse(`
		container {{.Name | camel }} {
			description "Auto-generated from table {{.Name}}";

			{{range .Columns}}
			leaf {{.Name | camel }} {
				type {{.YangType}};
				{{if .IsNullable}}x:nullable;{{end}}        
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
		Name: table,
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
	Name          string `db:"Name"`
	DataType      string `db:"DataType"`
	RawIsNullable string `db:"RawIsNullable"`
}

func (c *column) IsNullable() bool {
	return c.RawIsNullable == "YES"
}

type tableData struct {
	Name    string
	Columns []column
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
	case "datetime", "datetime2", "smalldatetime", "date", "time":
		ytype = "dateTime"
	case "float", "real", "decimal", "numeric", "money", "smallmoney":
		ytype = "decimal64"
	default:
		ytype = "string"
	}
	return ytype
}

func chkerr(err error) {
	if err != nil {
		panic(err)
	}
}
