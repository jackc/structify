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
	StructifyScan(parser *Parser, source any) error
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

func Parse(m map[string]any, target any) error {
	return DefaultParser.Parse(m, target)
}

type Parser struct {
	typeScannerFuncs map[reflect.Type]TypeScannerFunc
}

type TypeScannerFunc func(parser *Parser, source, target any) error

// RegisterTypeScanner configures parser to call fn for any scan target with the same type as value.
func (p *Parser) RegisterTypeScanner(value any, fn TypeScannerFunc) {
	if p.typeScannerFuncs == nil {
		p.typeScannerFuncs = make(map[reflect.Type]TypeScannerFunc)
	}

	p.typeScannerFuncs[reflect.TypeOf(value)] = fn
}

// Parse
func (p *Parser) Parse(source, target any) error {
	source, err := normalizeSource(source)
	if err != nil {
		return fmt.Errorf("structify: %v", err)
	}

	return p.parseNormalizedSource(source, target)
}

func (p *Parser) parseNormalizedSource(source, target any) error {
	if p.typeScannerFuncs != nil {
		targetType := reflect.TypeOf(target)
		if fn, ok := p.typeScannerFuncs[targetType]; ok {
			err := fn(p, source, target)
			if err != nil {
				return fmt.Errorf("structify: %v", err)
			}
			return nil
		}
	}

	switch target := target.(type) {
	case StructifyScanner:
		err := target.StructifyScan(p, source)
		if err != nil {
			return fmt.Errorf("structify: %v", err)
		}
		return nil
	case Scanner:
		err := target.Scan(source)
		if err != nil {
			return fmt.Errorf("structify: %v", err)
		}
		return nil
	}

	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return fmt.Errorf("structify.Parse: target is not a pointer, %v", targetVal.Kind())
	}
	if targetVal.IsNil() {
		return fmt.Errorf("structify.Parse: target cannot be nil")
	}

	targetElemVal := targetVal.Elem()

	switch targetElemVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		err := p.setAnyInt(source, targetElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Float32, reflect.Float64:
		err := p.setAnyFloat(source, targetElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.String:
		err := p.setAnyString(source, targetElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Bool:
		err := p.setAnyBool(source, targetElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Struct:
		err := p.setAnyStruct(source, targetElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Slice:
		err := p.setAnySlice(source, targetElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Interface:
		err := p.setAnyInterface(source, targetElemVal)
		if err != nil {
			return fmt.Errorf("structify.Parse: %v", err)
		}
	case reflect.Pointer:
		if source == nil {
			targetElemVal.Set(reflect.Zero(targetElemVal.Type()))
		} else {
			targetElemVal.Set(reflect.New(targetElemVal.Type().Elem()))
			err := p.parseNormalizedSource(source, targetElemVal.Interface())
			if err != nil {
				return fmt.Errorf("structify.Parse: %v", err)
			}
		}

	default:
		return fmt.Errorf("cannot assign %T to %v", source, targetVal.Type())
	}

	return nil
}

func normalizeSource(source any) (any, error) {
	switch source := source.(type) {
	case string:
		return source, nil

	case int:
		return int64(source), nil
	case int8:
		return int64(source), nil
	case int16:
		return int64(source), nil
	case int32:
		return int64(source), nil
	case int64:
		return int64(source), nil

	// Not supporting unsigned int inputs to avoid having to deal with overflow for uint and uint64.

	case float32:
		return float64(source), nil
	case float64:
		return float64(source), nil

	case bool:
		return source, nil

	case map[string]any:
		normSrc := make(map[string]any, len(source))
		for k, v := range source {
			normV, err := normalizeSource(v)
			if err != nil {
				return nil, err
			}
			normSrc[k] = normV
		}
		return normSrc, nil

	case map[string]string:
		newMap := make(map[string]any, len(source))
		for k, v := range source {
			newMap[k] = v
		}
		return newMap, nil

	case []any:
		normSrc := make([]any, len(source))
		for i := range source {
			normV, err := normalizeSource(source[i])
			if err != nil {
				return nil, err
			}
			normSrc[i] = normV
		}
		return normSrc, nil
	}

	sourceVal := reflect.ValueOf(source)
	if sourceVal.Kind() == reflect.Slice {
		newSlice := make([]any, sourceVal.Len())
		for i := 0; i < sourceVal.Len(); i++ {
			normSrcVal, err := normalizeSource(sourceVal.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			newSlice[i] = normSrcVal
		}
		return newSlice, nil
	}

	// Normalize typed nils into untyped nils
	if source == nil || sourceVal.IsNil() {
		return nil, nil
	}

	return nil, fmt.Errorf("unsupported source type: %T", source)
}

func (p *Parser) parseString(source string, targetVal reflect.Value) error {
	switch targetVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseInt(source, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
		}
		if targetVal.OverflowInt(n) {
			return fmt.Errorf("%v overflows %v", n, targetVal.Type())
		}
		targetVal.SetInt(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(source, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
		}
		targetVal.SetFloat(n)

	case reflect.String:
		targetVal.SetString(source)
	default:
		return fmt.Errorf("cannot assign %T to %v", source, targetVal.Type())
	}

	return nil
}

func (p *Parser) setAnyInt(source any, targetVal reflect.Value) error {
	var n int64
	switch source := source.(type) {
	case int64:
		n = source
	case float64:
		n = int64(source)
		if source != float64(n) {
			return fmt.Errorf("%v is not an integer", source)
		}
	case string:
		var err error
		n, err = strconv.ParseInt(source, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
		}
	default:
		return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
	}
	if targetVal.OverflowInt(n) {
		return fmt.Errorf("%v overflows %v", n, targetVal.Type())
	}
	targetVal.SetInt(n)

	return nil
}

func (p *Parser) setAnyFloat(source any, targetVal reflect.Value) error {
	var n float64
	switch source := source.(type) {
	case float64:
		n = source
	case int64:
		n = float64(source)
	case string:
		var err error
		n, err = strconv.ParseFloat(source, 64)
		if err != nil {
			return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
		}
	default:
		return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
	}
	targetVal.SetFloat(n)

	return nil
}

func (p *Parser) setAnyString(source any, targetVal reflect.Value) error {
	var s string
	switch source := source.(type) {
	case string:
		s = source
	case int64:
		s = strconv.FormatInt(source, 10)
	case float64:
		s = strconv.FormatFloat(source, 'f', -1, 64)
	default:
		return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
	}
	targetVal.SetString(s)

	return nil
}

func (p *Parser) setAnyBool(source any, targetVal reflect.Value) error {
	var b bool
	switch source := source.(type) {
	case bool:
		b = source
	default:
		return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
	}
	targetVal.SetBool(b)

	return nil
}

func (p *Parser) setAnyStruct(source any, targetVal reflect.Value) error {
	var sourceMap map[string]any
	var ok bool
	if sourceMap, ok = source.(map[string]any); !ok {
		return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
	}

	normalizedNameToMapKey := make(map[string]string, len(sourceMap))
	for key := range sourceMap {
		normalizedNameToMapKey[normalizeFieldName(key)] = key
	}

	targetElemType := targetVal.Type()

	for i := 0; i < targetElemType.NumField(); i++ {
		structField := targetElemType.Field(i)
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

		mapValue, found := sourceMap[mapKey]
		if found {
			err := p.parseNormalizedSource(mapValue, targetVal.Field(i).Addr().Interface())
			if err != nil {
				return fmt.Errorf("unable to set value for %s: %v", structField.Name, err)
			}
		} else {
			field := targetVal.Field(i).Addr().Interface()
			if mfc, ok := field.(MissingFieldScanner); ok {
				mfc.ScanMissingField()
			} else {
				return fmt.Errorf("missing value for %s", structField.Name)
			}
		}

	}

	return nil
}

func (p *Parser) setAnySlice(source any, targetVal reflect.Value) error {
	sourceVal := reflect.ValueOf(source)
	if sourceVal.Kind() != reflect.Slice {
		return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
	}

	targetVal.Set(reflect.MakeSlice(targetVal.Type(), sourceVal.Len(), sourceVal.Cap()))

	for i := 0; i < sourceVal.Len(); i++ {
		err := p.parseNormalizedSource(sourceVal.Index(i).Interface(), targetVal.Index(i).Addr().Interface())
		if err != nil {
			return fmt.Errorf("cannot assign [%d]: %v", i, err)
		}
	}

	return nil
}

func (p *Parser) setAnyInterface(source any, targetVal reflect.Value) error {
	sourceVal := reflect.ValueOf(source)

	if !sourceVal.CanConvert(targetVal.Type()) {
		return fmt.Errorf("cannot assign %v to %v", source, targetVal.Type())
	}

	targetVal.Set(sourceVal.Convert(targetVal.Type()))

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
