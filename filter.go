package ncservice

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

func FilterOnlyKeys(x HasBasicCrudSupport) ValueFilter {
	return filterKeys(x, true)
}

func FilterNoKeys(x HasBasicCrudSupport) ValueFilter {
	return filterKeys(x, false)
}

func filterKeys(x HasBasicCrudSupport, isKey bool) ValueFilter {
	keys := x.CrudInfo().Keys
	return func(p Value) bool {
		for _, k := range keys {
			if k == p.Col {
				return isKey
			}
		}
		return !isKey
	}
}

func AppendFiltered(args []Value, p Value, f ValueFilter) []Value {
	if f(p) {
		return append(args, p)
	}
	return args
}
