package pgimpl

import (
	"reflect"
)

func normalizeNil(v any) any {
	if v := reflect.ValueOf(v); v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}
	return v
}
