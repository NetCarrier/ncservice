package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"

	"github.com/NetCarrier/ncservice/codegen"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
)

var mssqlArg = flag.String("mssql", "", "SQL Server connection string.")
var mysqlArg = flag.String("mysql", "", "MySQL connection string.")
var outArg = flag.String("out", "", "Required output file.")
var lookupArg = flag.String("lookup", "", "Config to process lookup data.")
var crudArg = flag.String("crud", "", "Config file to process struct data.")

func main() {
	flag.Parse()

	if *outArg == "" {
		panic("out arg required")
	}

	var out bytes.Buffer

	if *lookupArg != "" {
		var err error
		var pool *sqlx.DB
		if *mssqlArg != "" {
			pool, err = sqlx.Open("sqlserver", *mssqlArg)
			chkerr(err)
		} else if *mysqlArg != "" {
			pool, err = sqlx.Open("mysql", *mysqlArg)
			chkerr(err)
		} else {
			log.Fatal("one of mssql or mysql arg required")
		}
		defer pool.Close()
		cfg := readConfig[codegen.LookupOptions](*lookupArg)
		chkerr(codegen.NewLookuper(pool, cfg).Run(&out))
	}
	if *crudArg != "" {
		cfg := readConfig[codegen.CrudOptions](*crudArg)
		chkerr(codegen.NewCruder(cfg).Run(&out))
	}

	content, err := format.Source(out.Bytes())
	if err != nil {
		os.WriteFile(*outArg, out.Bytes(), 0666)
		chkerr(fmt.Errorf("did not generate valid go code. %v", err))
	}
	chkerr(os.WriteFile(*outArg, content, 0666))
}

func readConfig[T any](path string) T {
	var cfg T
	f, err := os.ReadFile(path)
	chkerr(err)
	chkerr(json.Unmarshal(f, &cfg))
	return cfg
}

func chkerr(err error) {
	if err != nil {
		panic(err)
	}
}
