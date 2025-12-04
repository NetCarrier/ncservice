package ncservice

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const GROUP_ALL = "all"

type Value struct {
	Col string
	Val any
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
			Col: col,
		}
		refval := ref.Field(i)
		if !(refval.Kind() == reflect.Ptr && refval.IsNil()) {
			v.Val = refval.Interface()
		}
		values = AppendFiltered(values, v, fld, f)
	}
	return values
}

func GetPrimaryKeyColumn(h any) []string {
	var cols []string
	forEachGorm(reflect.ValueOf(h).Elem(), func(fld reflect.StructField, tag string) bool {
		if _, exists := getGormTag(tag, "primaryKey"); exists {
			col, _ := getGormTag(tag, "column")
			cols = append(cols, col)
		}
		return true
	})
	return cols
}

func forEachGorm(ref reflect.Value, fn func(fld reflect.StructField, tag string) bool) {
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

// FieldToColumn maps a JSON field name to its corresponding database column name
// using reflection to read the gorm:"column:xxx" tag from the provided struct.
func FieldToColumn[T any](jsonField string) (string, error) {
	var instance T
	t := reflect.TypeOf(instance)
	jsonFieldLc := strings.ToLower(jsonField)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get the JSON tag to match against the input
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// Parse JSON tag (format: "fieldName,omitempty" or just "fieldName")
		jsonName := strings.ToLower(strings.Split(jsonTag, ",")[0])
		if jsonName == jsonFieldLc {
			// Found matching field, now extract the column name from gorm tag
			gormTag := field.Tag.Get("gorm")
			if gormTag == "" {
				return "", fmt.Errorf("field %s has no gorm tag", jsonField)
			}

			// Parse gorm tag to find column name
			for _, part := range strings.Split(gormTag, ";") {
				if strings.HasPrefix(part, "column:") {
					columnName := strings.TrimPrefix(part, "column:")
					return columnName, nil
				}
			}

			return "", fmt.Errorf("field %s has no column in gorm tag", jsonField)
		}
	}

	return "", fmt.Errorf("unknown field: %s", jsonField)
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

func SqlSelectColumns[T any](prefix string, target []string) string {
	var selected []string
	var t T
	ref := reflect.ValueOf(t)
	forEachGorm(ref, func(fld reflect.StructField, col string) bool {
		groupsStr := fld.Tag.Get("groups")
		if groupsStr != "" {
			groups := strings.Split(groupsStr, ",")
			for _, targetGroup := range target {
				if targetGroup == GROUP_ALL {
					goto selection
				}
				if slices.Contains(groups, targetGroup) {
					goto selection
				}
			}
			return true
		}
	selection:
		selected = append(selected, prefix+col)
		return true
	})
	return strings.Join(selected, ", ")
}
