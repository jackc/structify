package structify

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

const structTagKey = "structify"

var DefaultParser *Parser

func init() {
	DefaultParser = &Parser{}
}

func Parse(m map[string]any, dest any) error {
	return DefaultParser.Parse(m, dest)
}

type Parser struct {
}

// Parse
func (p *Parser) Parse(src, dst any) error {
	src, err := normalizeSource(src)
	if err != nil {
		return fmt.Errorf("structify: %v", err)
	}

	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr {
		return fmt.Errorf("structify.Parse: dst is not a pointer, %v", dstVal.Kind())
	}
	if dstVal.IsNil() {
		return fmt.Errorf("structify.Parse: dst cannot be nil")
	}

	dstElemVal := dstVal.Elem()

	switch dstElemVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		err := p.setAnyInt(src, dstElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Float32, reflect.Float64:
		err := p.setAnyFloat(src, dstElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.String:
		err := p.setAnyString(src, dstElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Bool:
		err := p.setAnyBool(src, dstElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Struct:
		err := p.setAnyStruct(src, dstElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Slice:
		err := p.setAnySlice(src, dstElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}

	default:
		return fmt.Errorf("cannot assign %T to %v", src, dstVal.Type())
	}

	return nil
}

func normalizeSource(src any) (any, error) {
	switch src := src.(type) {
	case string:
		return src, nil

	case int:
		return int64(src), nil
	case int8:
		return int64(src), nil
	case int16:
		return int64(src), nil
	case int32:
		return int64(src), nil
	case int64:
		return int64(src), nil

	// Not supporting unsigned int inputs to avoid having to deal with overflow for uint and uint64.

	case float32:
		return float64(src), nil
	case float64:
		return float64(src), nil

	case bool:
		return src, nil

	case map[string]any:
		return src, nil

	case map[string]string:
		newMap := make(map[string]any, len(src))
		for k, v := range src {
			newMap[k] = v
		}
		return newMap, nil

	case []any:
		return src, nil
	}

	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Slice {
		newSlice := make([]any, srcVal.Len())
		for i := 0; i < srcVal.Len(); i++ {
			newSlice[i] = srcVal.Index(i).Interface()
		}
		return newSlice, nil
	}

	return nil, fmt.Errorf("unsupported source type: %T", src)
}

func (p *Parser) parseString(src string, dstVal reflect.Value) error {
	switch dstVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseInt(src, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
		}
		if dstVal.OverflowInt(n) {
			return fmt.Errorf("%v overflows %v", n, dstVal.Type())
		}
		dstVal.SetInt(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(src, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
		}
		dstVal.SetFloat(n)

	case reflect.String:
		dstVal.SetString(src)
	default:
		return fmt.Errorf("cannot assign %T to %v", src, dstVal.Type())
	}

	return nil
}

func (p *Parser) setAnyInt(src any, dstVal reflect.Value) error {
	var n int64
	switch src := src.(type) {
	case int64:
		n = src
	case float64:
		n = int64(src)
		if src != float64(n) {
			return fmt.Errorf("%v is not an integer", src)
		}
	case string:
		var err error
		n, err = strconv.ParseInt(src, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
		}
	default:
		return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
	}
	if dstVal.OverflowInt(n) {
		return fmt.Errorf("%v overflows %v", n, dstVal.Type())
	}
	dstVal.SetInt(n)

	return nil
}

func (p *Parser) setAnyFloat(src any, dstVal reflect.Value) error {
	var n float64
	switch src := src.(type) {
	case float64:
		n = src
	case int64:
		n = float64(src)
	case string:
		var err error
		n, err = strconv.ParseFloat(src, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
		}
	default:
		return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
	}
	dstVal.SetFloat(n)

	return nil
}

func (p *Parser) setAnyString(src any, dstVal reflect.Value) error {
	var s string
	switch src := src.(type) {
	case string:
		s = src
	case int64:
		s = strconv.FormatInt(src, 10)
	case float64:
		s = strconv.FormatFloat(src, 'f', -1, 64)
	default:
		return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
	}
	dstVal.SetString(s)

	return nil
}

func (p *Parser) setAnyBool(src any, dstVal reflect.Value) error {
	var b bool
	switch src := src.(type) {
	case bool:
		b = src
	default:
		return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
	}
	dstVal.SetBool(b)

	return nil
}

func (p *Parser) setAnyStruct(src any, dstVal reflect.Value) error {
	var srcMap map[string]any
	var ok bool
	if srcMap, ok = src.(map[string]any); !ok {
		return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
	}

	normalizedNameToMapKey := make(map[string]string, len(srcMap))
	for key := range srcMap {
		normalizedNameToMapKey[normalizeFieldName(key)] = key
	}

	destElemType := dstVal.Type()

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
				return fmt.Errorf("missing value for %s", structField.Name)
			}
		}

		mapValue, ok := srcMap[mapKey]
		if !ok {
			return fmt.Errorf("missing value for %s", structField.Name)
		}

		err := p.Parse(mapValue, dstVal.Field(i).Addr().Interface())
		if err != nil {
			return fmt.Errorf("unable to set value for %s: %v", structField.Name, err)
		}
	}

	return nil
}

func (p *Parser) setAnySlice(src any, dstVal reflect.Value) error {
	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() != reflect.Slice {
		return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
	}

	dstVal.Set(reflect.MakeSlice(dstVal.Type(), srcVal.Len(), srcVal.Cap()))

	for i := 0; i < srcVal.Len(); i++ {
		err := p.Parse(srcVal.Index(i).Interface(), dstVal.Index(i).Addr().Interface())
		if err != nil {
			return fmt.Errorf("cannot assign [%d]: %v", i, err)
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
