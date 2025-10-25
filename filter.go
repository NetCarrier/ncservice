package ncservice

import (
	"reflect"
	"slices"
)

type ValueFilter func(Value) bool

func FilterAll(param Value) bool {
	return true
}

func FilterNotNil(param Value) bool {
	return param.Val != nil
}

func FilterAnd(f ...ValueFilter) ValueFilter {
	return func(p Value) bool {
		for _, f1 := range f {
			if !f1(p) {
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
	return func(p Value) bool {
		if slices.Contains(keys, p.Col) {
			return isKey
		}
		return !isKey
	}
}

func AppendFiltered(args []Value, p Value, f ValueFilter) []Value {
	if f == nil || f(p) {
		return append(args, p)
	}
	return args
}
