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

// StructifyScanner allows a type to control how it is parsed.
type StructifyScanner interface {
	StructifyScan(parser *Parser, src any) error
}

// Scanner matches the database/sql.Scanner interface. It allows many database/sql types to be used without needing to
// implement any structify interfaces. If a type does need to implement custom scanning logic for structify prefer the
// StructifyScanner interface.
type Scanner interface {
	Scan(value any) error
}

// MissingFieldScanner allows a field to be missing from the source data.
type MissingFieldScanner interface {
	ScanMissingField()
}

func Parse(m map[string]any, dest any) error {
	return DefaultParser.Parse(m, dest)
}

type Parser struct {
	typeScannerFuncs map[reflect.Type]TypeScannerFunc
}

type TypeScannerFunc func(parser *Parser, src, dst any) error

// RegisterTypeScanner configures parser to call fn for any scan destination with the same type as value.
func (p *Parser) RegisterTypeScanner(value any, fn TypeScannerFunc) {
	if p.typeScannerFuncs == nil {
		p.typeScannerFuncs = make(map[reflect.Type]TypeScannerFunc)
	}

	p.typeScannerFuncs[reflect.TypeOf(value)] = fn
}

// Parse
func (p *Parser) Parse(src, dst any) error {
	src, err := normalizeSource(src)
	if err != nil {
		return fmt.Errorf("structify: %v", err)
	}

	return p.parseNormalizedSource(src, dst)
}

func (p *Parser) parseNormalizedSource(src, dst any) error {
	if p.typeScannerFuncs != nil {
		dstType := reflect.TypeOf(dst)
		if fn, ok := p.typeScannerFuncs[dstType]; ok {
			err := fn(p, src, dst)
			if err != nil {
				return fmt.Errorf("structify: %v", err)
			}
			return nil
		}
	}

	switch dst := dst.(type) {
	case StructifyScanner:
		err := dst.StructifyScan(p, src)
		if err != nil {
			return fmt.Errorf("structify: %v", err)
		}
		return nil
	case Scanner:
		err := dst.Scan(src)
		if err != nil {
			return fmt.Errorf("structify: %v", err)
		}
		return nil
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
	case reflect.Interface:
		err := p.setAnyInterface(src, dstElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Pointer:
		if src == nil {
			dstElemVal.Set(reflect.Zero(dstElemVal.Type()))
		} else {
			dstElemVal.Set(reflect.New(dstElemVal.Type().Elem()))
			err := p.parseNormalizedSource(src, dstElemVal.Interface())
			if err != nil {
				return fmt.Errorf("structify.Parse: %v", err)
			}
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
		normSrc := make(map[string]any, len(src))
		for k, v := range src {
			normV, err := normalizeSource(v)
			if err != nil {
				return nil, err
			}
			normSrc[k] = normV
		}
		return normSrc, nil

	case map[string]string:
		newMap := make(map[string]any, len(src))
		for k, v := range src {
			newMap[k] = v
		}
		return newMap, nil

	case []any:
		normSrc := make([]any, len(src))
		for i := range src {
			normV, err := normalizeSource(src[i])
			if err != nil {
				return nil, err
			}
			normSrc[i] = normV
		}
		return normSrc, nil
	}

	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Slice {
		newSlice := make([]any, srcVal.Len())
		for i := 0; i < srcVal.Len(); i++ {
			normSrcVal, err := normalizeSource(srcVal.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			newSlice[i] = normSrcVal
		}
		return newSlice, nil
	}

	// Normalize typed nils into untyped nils
	if src == nil || srcVal.IsNil() {
		return nil, nil
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
			mapKey = normalizedNameToMapKey[normalizedName]
		}

		mapValue, found := srcMap[mapKey]
		if found {
			err := p.parseNormalizedSource(mapValue, dstVal.Field(i).Addr().Interface())
			if err != nil {
				return fmt.Errorf("unable to set value for %s: %v", structField.Name, err)
			}
		} else {
			field := dstVal.Field(i).Addr().Interface()
			if mfc, ok := field.(MissingFieldScanner); ok {
				mfc.ScanMissingField()
			} else {
				return fmt.Errorf("missing value for %s", structField.Name)
			}
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
		err := p.parseNormalizedSource(srcVal.Index(i).Interface(), dstVal.Index(i).Addr().Interface())
		if err != nil {
			return fmt.Errorf("cannot assign [%d]: %v", i, err)
		}
	}

	return nil
}

func (p *Parser) setAnyInterface(src any, dstVal reflect.Value) error {
	srcVal := reflect.ValueOf(src)

	if !srcVal.CanConvert(dstVal.Type()) {
		return fmt.Errorf("cannot assign %v to %v", src, dstVal.Type())
	}

	dstVal.Set(srcVal.Convert(dstVal.Type()))

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

// Optional wraps any type and allows it to be missing from the source data.
type Optional[T any] struct {
	Value   T
	Present bool
}

func (opt *Optional[T]) ScanMissingField() {
	*opt = Optional[T]{}
}
