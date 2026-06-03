package ncservice

import (
	"fmt"
	"reflect"
	"strings"
)

const GROUP_ALL = "all"

type Value struct {
	Table string
	Col   string
	Val   any
}

func JsonValues(h any, f ValueFilter) []Value {
	return values(h, f, jsonColMapper)
}

// Read all fields in a given struct ready for DB i/o
func Values(h any, f ValueFilter) []Value {
	return values(h, f, dbColMapper)
}

func dbColMapper(fld reflect.StructField) (string, bool) {
	return getGormTag(fld.Tag.Get("gorm"), "column")
}

func jsonColMapper(fld reflect.StructField) (string, bool) {
	name := fld.Tag.Get("json")
	if name == "" || name == "-" {
		return "", false
	}
	segs := strings.Split(name, ",")
	return segs[0], true
}

type colMapper func(fld reflect.StructField) (string, bool)

func values(h any, f ValueFilter, getCol colMapper) []Value {
	ref := reflect.ValueOf(h)
	if ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	t := ref.Type()
	n := ref.NumField()
	var values []Value
	for i := range n {
		fld := t.Field(i)
		col, exists := getCol(fld)
		if !exists {
			continue
		}
		v := Value{
			Col:   col,
			Table: fld.Tag.Get("table"),
		}
		refval := ref.Field(i)
		if !(refval.Kind() == reflect.Ptr && refval.IsNil()) {
			v.Val = refval.Interface()
		}
		values = appendFiltered(values, v, fld, f)
	}
	return values
}

func GetPrimaryKeyColumn(h any) []string {
	var cols []string
	ForEachGorm(reflect.ValueOf(h).Elem(), func(fld reflect.StructField, tag string, col string) bool {
		if _, exists := getGormTag(tag, "primaryKey"); exists {
			cols = append(cols, col)
		}
		return true
	})
	return cols
}

func ForEachGorm(ref reflect.Value, fn func(fld reflect.StructField, tag string, col string) bool) {
	t := ref.Type()
	for i := 0; i < ref.NumField(); i++ {
		fld := t.Field(i)
		tag := fld.Tag.Get("gorm")
		col, exists := getGormTag(tag, "column")
		if !exists {
			continue
		}
		if !fn(fld, tag, col) {
			break
		}
	}
}

func ForEachTag(h any, tagId string, fn func(f reflect.StructField, v reflect.Value, tag string) bool) {
	ref := reflect.ValueOf(h)
	if ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	t := ref.Type()
	for i := 0; i < ref.NumField(); i++ {
		fld := t.Field(i)
		t := fld.Tag.Get(tagId)
		if t == "" {
			continue
		}
		v := ref.FieldByName(fld.Name)
		if !fn(fld, v, t) {
			break
		}
	}
}

func getGormTag(tag string, target string) (string, bool) {
	for _, part := range strings.Split(tag, ";") {
		if part == target {
			return "", true
		}
		if strings.HasPrefix(part, target+":") {
			match := strings.Split(string(part), ":")
			return match[1], true
		}
	}
	return "", false
}

// FieldToColumn maps a JSON field name to its corresponding database column name
// using reflection to read the gorm:"column:xxx" tag from the provided struct.
func FieldToColumn[T any](jsonField string) (string, string, error) {
	var instance T
	t := reflect.TypeOf(instance)
	jsonFieldLc := strings.ToLower(jsonField)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get the JSON tag to match against the input
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag (format: "fieldName,omitempty" or just "fieldName")
		jsonName := strings.ToLower(strings.Split(jsonTag, ",")[0])
		if jsonName == jsonFieldLc {
			// Found matching field, now extract the column name from gorm tag
			gormTag := field.Tag.Get("gorm")
			if gormTag == "" {
				return "", "", fmt.Errorf("field %s has no gorm tag", jsonField)
			}

			// Parse gorm tag to find column name using getGormTag helper
			columnName, exists := getGormTag(gormTag, "column")
			if !exists {
				return "", "", fmt.Errorf("field %s has no column in gorm tag", jsonField)
			}
			table := field.Tag.Get("table")
			return columnName, table, nil
		}
	}

	return "", "", fmt.Errorf("unknown field: %s", jsonField)
}

// SetValues takes a list of values likely obtained from Values() and sets the corresponding
func SetValues(h any, values []Value) error {
	return setValues(h, values, dbColMapper)
}

// SetJsonValues takes a list of values likely obtained from JsonValues() and sets the corresponding
func SetJsonValues(h any, values []Value) error {
	return setValues(h, values, jsonColMapper)
}

// setValues takes a list of values likely obtained from Values() and sets the corresponding
func setValues(h any, values []Value, getCol colMapper) error {
	ref := reflect.ValueOf(h).Elem()
	t := ref.Type()
	for i := 0; i < ref.NumField(); i++ {
		fld := t.Field(i)
		col, exists := getCol(fld)
		if !exists {
			continue
		}
		for _, v := range values {
			if v.Col == col {
				field := ref.Field(i)
				if err := setValue(field, v); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

// setValue sets the value of a struct field using reflection, handling type conversion as needed.
func setValue(to reflect.Value, from Value) error {
	if !to.CanSet() {
		return fmt.Errorf("SetValues: cannot set field %s", to.Type().String())
	}

	fromVal := reflect.ValueOf(from.Val)
	if !fromVal.IsValid() {

	} else if fromVal.Type().ConvertibleTo(to.Type()) {
		to.Set(fromVal.Convert(to.Type()))
	} else if fromVal.Kind() == reflect.Ptr {
		eval := fromVal.Elem()
		if eval.Type().ConvertibleTo(to.Type()) {
			to.Set(eval.Convert(to.Type()))
		} else {
			return fmt.Errorf("SetValues: type not convertible for pointer: fieldType=%s, valueType=%s",
				to.Type().String(),
				fromVal.Type().String(),
			)
		}
	} else if to.Kind() == reflect.Ptr {
		pfield := reflect.New(to.Type().Elem())
		if fromVal.Type().ConvertibleTo(to.Type().Elem()) {
			pfield.Elem().Set(fromVal.Convert(to.Type().Elem()))
			to.Set(pfield)
		} else {
			return fmt.Errorf("SetValues: type not convertible for pointer field: fieldType=%s, valueType=%s",
				to.Type().String(),
				fromVal.Type().String(),
			)
		}
	}
	return nil
}
