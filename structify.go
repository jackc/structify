package structify

import (
	"fmt"
	"reflect"
	"strconv"
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
		var mapKey string
		if tag, ok := structField.Tag.Lookup(structTagKey); ok {
			if tag == "-" {
				continue // Skip ignored fields
			}
			mapKey = tag
		} else {
			normalizedName := normalizeFieldName(structField.Name)
			var found bool
			mapKey, found = normalizedNameToMapKey[normalizedName]
			if !found {
				return fmt.Errorf("structify.Map: m is missing key for %s", structField.Name)
			}
		}

		mapValue, ok := m[mapKey]
		if !ok {
			return fmt.Errorf("structify.Map: m is missing key for %s", structField.Name)
		}

		err := setAny(destElemValue.Field(i), mapValue)
		if err != nil {
			return fmt.Errorf("structify.Map: unable to set value for %s: %v", structField.Name, err)
		}
	}

	return nil
}

func setAny(dst reflect.Value, src any) error {
	switch src := src.(type) {
	case string:
		return setString(dst, src)
	case int:
		return setInt64(dst, int64(src))
	case int8:
		return setInt64(dst, int64(src))
	case int16:
		return setInt64(dst, int64(src))
	case int32:
		return setInt64(dst, int64(src))
	case int64:
		return setInt64(dst, int64(src))

	// Not supporting unsigned int inputs to avoid having to deal with overflow for uint and uint64.

	case float32:
		return setFloat64(dst, float64(src), 32)
	case float64:
		return setFloat64(dst, float64(src), 64)

	default:
		return fmt.Errorf("unsupported input type: %T", src)
	}
}

func setString(dst reflect.Value, src string) error {
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseInt(src, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", src, dst.Type())
		}
		if dst.OverflowInt(n) {
			return fmt.Errorf("%v overflows %v", n, dst.Type())
		}
		dst.SetInt(n)

	case reflect.String:
		dst.SetString(src)
	default:
		return fmt.Errorf("cannot assign %T to %v", src, dst.Type())
	}

	return nil
}

func setInt64(dst reflect.Value, src int64) error {
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if dst.OverflowInt(src) {
			return fmt.Errorf("%v overflows %v", src, dst.Type())
		}
		dst.SetInt(src)

	case reflect.String:
		dst.SetString(strconv.FormatInt(src, 10))
	case reflect.Float32, reflect.Float64:
		dst.SetFloat(float64(src))
	default:
		return fmt.Errorf("cannot assign %T to %s", src, dst.Type())
	}

	return nil
}

func setFloat64(dst reflect.Value, src float64, bitSize int) error {
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i64Src := int64(src)
		if src != float64(i64Src) {
			return fmt.Errorf("%v is not an integer", src)
		}

		if dst.OverflowInt(i64Src) {
			return fmt.Errorf("%v overflows %v", src, dst.Type())
		}
		dst.SetInt(i64Src)
	case reflect.String:
		dst.SetString(strconv.FormatFloat(src, 'f', -1, bitSize))
	case reflect.Float32, reflect.Float64:
		dst.SetFloat(src)
	default:
		return fmt.Errorf("cannot assign %T to %v", src, dst.Type())
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
