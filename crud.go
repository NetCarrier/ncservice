package ncservice

type Value struct {
	Col string
	Val any
}

type CrudInfo struct {
	Table string
	Keys  []string
}

type HasBasicCrudSupport interface {
	CrudInfo() CrudInfo
	SetValues(values []Value)
	Values(f ValueFilter) []Value
}

type HasValuesSupport interface {
	Values(f ValueFilter) []Value
}
