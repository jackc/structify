package structify

import (
	"fmt"
	"reflect"
)

const structTagKey = "structify"

// Map
func Map(m map[string]any, dest any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("structify.Map: dest is not a pointer to struct")
	}

	if destValue.IsNil() {
		return fmt.Errorf("structify.Map: dest cannot be nil")
	}

	destElemValue := destValue.Elem()
	destElemType := destElemValue.Type()

	for i := 0; i < destElemType.NumField(); i++ {
		structField := destElemType.Field(i)
		keyName := structField.Name
		if value, ok := m[keyName]; ok {
			destElemValue.Field(i).Set(reflect.ValueOf(value))
		} else {
			return fmt.Errorf("structify.Map: m is missing %s", keyName)
		}
	}

	return nil
}
