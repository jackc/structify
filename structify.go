package structify

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

const structTagKey = "structify"

// Map
func Map(m map[string]any, dest any) error {
	normalizedNameToMapKey := make(map[string]string, len(m))
	for key := range m {
		normalizedNameToMapKey[normalizeFieldName(key)] = key
	}

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
		normalizedName := normalizeFieldName(structField.Name)
		if mapKey, ok := normalizedNameToMapKey[normalizedName]; ok {
			value := reflect.ValueOf(m[mapKey])
			destElemValue.Field(i).Set(value)
		} else {
			return fmt.Errorf("structify.Map: m is missing key for %s", structField.Name)
		}
	}

	return nil
}

// normalizeFieldName removes all characters except letters and digits and lower cases the letters.
func normalizeFieldName(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return unicode.ToLower(r)
		} else if unicode.IsDigit(r) {
			return r
		} else {
			return -1
		}
	}, s)
}
