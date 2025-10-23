package codegen

import "strings"

// Supports SQL Servier and MySql types
func GoType(dbType string) string {
	switch dbType {
	case "INT", "BIGINT", "SMALLINT", "TINYINT":
		return "int"
	case "BIT":
		return "bool"
	case "VARCHAR", "NVARCHAR", "TEXT", "NTEXT", "CHAR", "NCHAR":
		return "string"
	case "DATETIME", "DATETIME2", "SMALLDATETIME", "DATE", "TIME":
		return "dateTime"
	case "FLOAT", "REAL", "DECIMAL", "NUMERIC", "MONEY", "SMALLMONEY":
		return "float64"
	default:
		return strings.ToLower(dbType)
	}
}
