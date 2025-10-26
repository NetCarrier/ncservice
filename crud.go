package ncservice

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Value struct {
	Col string
	Val any
}

// Read all fields in a given struct ready for DB i/o
func Values(h any, f ValueFilter) []Value {
	ref := reflect.ValueOf(h)
	if ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	t := ref.Type()
	n := ref.NumField()
	var values []Value
	for i := range n {
		fld := t.Field(i)
		col, exists := getGormTag(fld.Tag.Get("gorm"), "column")
		if !exists {
			continue
		}
		v := Value{
			Col: col,
		}
		refval := ref.Field(i)
		if !(refval.Kind() == reflect.Ptr && refval.IsNil()) {
			v.Val = refval.Interface()
		}
		values = AppendFiltered(values, v, f)
	}
	return values
}

func GetPrimaryKeyColumn(h any) []string {
	var cols []string
	forEachGorm(h, func(fld reflect.StructField, tag string) bool {
		if _, exists := getGormTag(tag, "primaryKey"); exists {
			col, _ := getGormTag(tag, "column")
			cols = append(cols, col)
		}
		return true
	})
	return cols
}

func forEachGorm(h any, fn func(fld reflect.StructField, tag string) bool) {
	ref := reflect.ValueOf(h).Elem()
	t := ref.Type()
	for i := 0; i < ref.NumField(); i++ {
		fld := t.Field(i)
		col, exists := getGormTag(fld.Tag.Get("gorm"), "column")
		if !exists {
			continue
		}
		if !fn(fld, col) {
			break
		}
	}
}

func forEachTag(h any, tagId string, fn func(f reflect.StructField, v reflect.Value, tag string) bool) {
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

func ResolveForeignKeys(db *gorm.DB, x schema.Tabler) error {
	var err error
	forEachTag(x, "fk", func(fld reflect.StructField, raw reflect.Value, tag string) bool {
		v := raw
		// handle pointers to foreign keys as optional FKs: if they exist, they must be valid
		if raw.Kind() == reflect.Ptr {
			if raw.IsNil() {
				return true
			}
			v = raw.Elem()
		}
		parts := strings.Split(tag, ".")
		if len(parts) != 2 {
			panic("invalid fk tag format " + tag)
		}
		targetTable := parts[0]
		targetCol := parts[1]
		var exists bool
		id := v.Interface()
		sql := fmt.Sprintf("select exists(select 1 from %s where %s = ?)", targetTable, targetCol)
		err = db.Raw(sql, id).Scan(&exists).Error
		if err != nil {
			err = fmt.Errorf("erorr checking foreign key. %w", err)
			return false
		}
		if !exists {
			err = fmt.Errorf("foreign key constraint failed: %s.%s=%v does not exist", targetTable, targetCol, id)
			return false
		}
		return true
	})
	return err
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

func SetValues(h any, values []Value) error {
	ref := reflect.ValueOf(h).Elem()
	t := ref.Type()
	for i := 0; i < ref.NumField(); i++ {
		fld := t.Field(i)
		col, exists := getGormTag(fld.Tag.Get("gorm"), "column")
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
