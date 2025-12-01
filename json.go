// This uses the experimental json/v2 package that allows for field-level custom marshalling
// outside the struct definition.
//
// Required Environment Variable to Compile:
//
//	GOEXPERIMENT=jsonv2
package ncservice

import (
	"bytes"
	"encoding/json/v2"
	"reflect"
	"slices"
	"strings"
)

const GROUP_DEFAULT = "default" // When in field list, this special group it is always included
const GROUP_ALL = "all"         // When in target list, all field groups are included

// MarshalJsonList marshals struct to JSON including only fields tagged with specified groups
// useful for limiting output in APIs
func MarshalJsonList[T any](x []T, groups ...string) ([]byte, error) {
	return json.Marshal(x, json.WithMarshalers(
		json.MarshalFunc(filterJsonByGroups[T](groups)),
	))
}

// MarshalJSON marshals struct to JSON including fields tagged with specified groups
// useful for limiting output in APIs
func MarshalJSON[T any](x T, groups ...string) ([]byte, error) {
	return json.Marshal(x, json.WithMarshalers(
		json.MarshalFunc(filterJsonByGroups[T](groups)),
	))
}

func filterJsonByGroups[T any](target []string) func(T) ([]byte, error) {
	return func(x T) ([]byte, error) {
		vals := JsonValues(x, func(v Value, fld reflect.StructField) bool {
			if slices.Contains(target, GROUP_ALL) {
				return true
			}
			grps := strings.SplitSeq(fld.Tag.Get("groups"), ",")
			for g := range grps {
				if slices.Contains(target, g) || g == GROUP_DEFAULT {
					return true
				}
			}
			return false
		})
		return json.Marshal(marshalValues(vals))
	}
}

type marshalValues []Value

// preserves the order of the fields while not strictly necessary for JSON
// it makes the output more predictable and easier to read
func (vals marshalValues) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("{")
	for i, kv := range vals {
		if i != 0 {
			buf.WriteString(",")
		}
		// marshal key
		key, err := json.Marshal(kv.Col)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(":")

		// marshal value
		val, err := json.Marshal(kv.Val)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}
