package ncservice

import (
	"reflect"
	"slices"
)

type ValueFilter func(Value, reflect.StructField) bool

func FilterAll(param Value, fld reflect.StructField) bool {
	return true
}

func FilterInclude(includedCols []string) ValueFilter {
	return func(p Value, fld reflect.StructField) bool {
		return slices.Contains(includedCols, p.Col)
	}
}

func FilterExclude(excludedCols []string) ValueFilter {
	return func(p Value, fld reflect.StructField) bool {
		return !slices.Contains(excludedCols, p.Col)
	}
}

func FilterNot(f ValueFilter) ValueFilter {
	return func(p Value, fld reflect.StructField) bool {
		return !f(p, fld)
	}
}

func FilterNotNil(param Value, fld reflect.StructField) bool {
	return param.Val != nil
}

// FilterNotEqual is like a diff useful to tell the difference of an object for before and after
func DiffVals(origVals []Value, updatedVals []Value) []DiffVal {
	var diff []DiffVal

outer1:
	for _, orig := range origVals {
		found := DiffVal{Col: orig.Col, Orig: orig.Val}
		for _, updated := range updatedVals {
			if orig.Col == updated.Col {
				if reflect.DeepEqual(orig.Val, updated.Val) {
					continue outer1
				} else {
					found.Updated = updated.Val
					break
				}
			}
		}
		diff = append(diff, found)
	}

outer2:
	for _, updated := range updatedVals {
		found := DiffVal{Col: updated.Col, Updated: updated.Val}
		for _, orig := range origVals {
			if orig.Col == updated.Col {
				continue outer2
			}
		}
		diff = append(diff, found)
	}

	return diff
}

type DiffVal struct {
	Col     string
	Orig    any
	Updated any
}

func FilterAnd(f ...ValueFilter) ValueFilter {
	return func(p Value, fld reflect.StructField) bool {
		for _, f1 := range f {
			if !f1(p, fld) {
				return false
			}
		}
		return true
	}
}

func FilterOnlyKeys(x any) ValueFilter {
	return filterKeys(x, true)
}

func FilterNoKeys(x any) ValueFilter {
	return filterKeys(x, false)
}

func filterKeys(x any, isKey bool) ValueFilter {
	ref := reflect.ValueOf(x)
	if ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	t := ref.Type()
	n := ref.NumField()
	var keys []string
	for i := range n {
		fld := t.Field(i)
		tag := fld.Tag.Get("gorm")
		_, exists := getGormTag(tag, "primaryKey")
		if !exists {
			continue
		}
		col, exists := getGormTag(tag, "column")
		if !exists {
			continue
		}
		keys = append(keys, col)
	}
	return func(p Value, fld reflect.StructField) bool {
		if slices.Contains(keys, p.Col) {
			return isKey
		}
		return !isKey
	}
}

func AppendFiltered(args []Value, p Value, fld reflect.StructField, f ValueFilter) []Value {
	if f == nil || f(p, fld) {
		return append(args, p)
	}
	return args
}
