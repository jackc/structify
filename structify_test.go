package structify_test

import (
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/structify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserParsesIntoStruct_FieldWithoutTagNameVariants(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string
	}

	for i, tt := range []struct {
		key string
	}{
		{key: "FirstName"},
		{key: "firstName"},
		{key: "firstname"},
		{key: "first_name"},
	} {
		var p Person
		err := parser.Parse(map[string]any{tt.key: "Jack"}, &p)
		require.NoErrorf(t, err, "%d. %s", i, tt.key)
		assert.Equalf(t, "Jack", p.FirstName, "%d. %s did not map to FirstName field", i, tt.key)
	}
}

func TestParserParsesIntoStruct_FieldWithTag(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string `structify:"name"`
	}

	var p Person
	err := parser.Parse(map[string]any{"name": "Jack"}, &p)
	require.NoError(t, err)
	assert.Equal(t, "Jack", p.FirstName)
}

func TestParserParsesIntoStruct_MissingRequiredField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string
		LastName  string
	}

	var p Person
	err := parser.Parse(map[string]any{"name": "Jack"}, &p)
	require.Error(t, err)
	var srcErr *structify.StructAssignmentError
	require.ErrorAs(t, err, &srcErr)
	fieldNameErrorMap := srcErr.FieldNameErrorMap()
	require.Len(t, fieldNameErrorMap, 2)
	require.Equal(t, "missing value", fieldNameErrorMap["FirstName"].Error())
	require.Equal(t, "missing value", fieldNameErrorMap["LastName"].Error())
}

func TestParserParsesIntoStruct_MissingOptionalField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string
		LastName  structify.Optional[string]
	}

	var p Person
	err := parser.Parse(map[string]any{"firstName": "Jack"}, &p)
	require.NoError(t, err)
	require.Equal(t, structify.Optional[string]{}, p.LastName)
}

func TestParserParsesIntoStruct_SkippedField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string
		LastName  string `structify:"-"`
	}

	var p Person
	err := parser.Parse(map[string]any{"FirstName": "Jack"}, &p)
	require.NoError(t, err)
	assert.Equal(t, "Jack", p.FirstName)
}

func TestParserParsesIntoStruct_NestedStructField(t *testing.T) {
	parser := &structify.Parser{}

	type Name struct {
		First string
		Last  string
	}

	type Person struct {
		Name Name
		Age  int32
	}

	for i, tt := range []struct {
		m map[string]any
		p Person
	}{
		{
			m: map[string]any{"name": map[string]any{"first": "John", "last": "Smith"}, "age": 42},
			p: Person{Name: Name{First: "John", Last: "Smith"}, Age: 42},
		},
	} {
		var p Person
		err := parser.Parse(tt.m, &p)
		require.NoErrorf(t, err, "%d. %v", i, tt.m)
		assert.Equalf(t, tt.p, p, "%d. %v", i, tt.m)
	}
}

func TestParserParsesIntoStruct_ArrayOfStructField(t *testing.T) {
	parser := &structify.Parser{}

	type Player struct {
		Name   string
		Number int32
	}

	type Team struct {
		Name    string
		Players []Player
	}

	for i, tt := range []struct {
		m map[string]any
		t Team
	}{
		{
			m: map[string]any{
				"name": "Bulls",
				"players": []any{
					map[string]any{"name": "Michael", "number": 23},
					map[string]any{"name": "Scotty", "number": 33},
				},
			},
			t: Team{
				Name: "Bulls",
				Players: []Player{
					{Name: "Michael", Number: 23},
					{Name: "Scotty", Number: 33},
				},
			},
		},
	} {
		var team Team
		err := parser.Parse(tt.m, &team)
		require.NoErrorf(t, err, "%d. %v", i, tt.m)
		assert.Equalf(t, tt.t, team, "%d. %v", i, tt.m)
	}
}

func TestParserParsesIntoStruct_ArrayOfPointerToStructField(t *testing.T) {
	parser := &structify.Parser{}

	type Player struct {
		Name   string
		Number int32
	}

	type Team struct {
		Name    string
		Players []*Player
	}

	for i, tt := range []struct {
		m map[string]any
		t Team
	}{
		{
			m: map[string]any{
				"name": "Bulls",
				"players": []any{
					map[string]any{"name": "Michael", "number": 23},
					map[string]any{"name": "Scotty", "number": 33},
				},
			},
			t: Team{
				Name: "Bulls",
				Players: []*Player{
					{Name: "Michael", Number: 23},
					{Name: "Scotty", Number: 33},
				},
			},
		},
	} {
		var team Team
		err := parser.Parse(tt.m, &team)
		require.NoErrorf(t, err, "%d. %v", i, tt.m)
		assert.Equalf(t, tt.t.Name, team.Name, "%d. %v", i, tt.m)
		assert.Equalf(t, len(tt.t.Players), len(team.Players), "%d. %v", i, tt.m)
		for j := 0; j < len(tt.t.Players); j++ {
			assert.Equalf(t, tt.t.Players[j], team.Players[j], "%d. %d. %v", i, j, tt.m)
		}
	}
}

func TestParserParsesIntoStruct_Int32Field(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		Age int32
	}

	for i, tt := range []struct {
		mapValue    any
		structValue int32
	}{
		{mapValue: int32(30), structValue: 30},
		{mapValue: int64(30), structValue: 30},
		{mapValue: int(30), structValue: 30},
		{mapValue: float32(30), structValue: 30},
		{mapValue: float64(30), structValue: 30},
		{mapValue: "30", structValue: 30},
	} {
		var p Person
		err := parser.Parse(map[string]any{"Age": tt.mapValue}, &p)
		assert.NoErrorf(t, err, "%d. %#v", i, tt.mapValue)
		assert.Equalf(t, int32(30), p.Age, "%d. %#v", i, tt.mapValue)
	}
}

func TestParserParsesIntoStruct_Float64Field(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		Age float64
	}

	for i, tt := range []struct {
		mapValue    any
		structValue float64
	}{
		{mapValue: float32(30.5), structValue: 30.5},
		{mapValue: float64(30.5), structValue: 30.5},
		{mapValue: int32(30), structValue: 30},
		{mapValue: int64(30), structValue: 30},
		{mapValue: int(30), structValue: 30},
		{mapValue: "30.5", structValue: 30.5},
	} {
		var p Person
		err := parser.Parse(map[string]any{"Age": tt.mapValue}, &p)
		assert.NoErrorf(t, err, "%d. %#v", i, tt.mapValue)
		assert.Equalf(t, tt.structValue, p.Age, "%d. %#v", i, tt.mapValue)
	}
}

func TestParserParsesIntoStruct_BoolField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		Alive bool
	}

	for i, tt := range []struct {
		mapValue    any
		structValue bool
	}{
		{mapValue: true, structValue: true},
		{mapValue: false, structValue: false},
	} {
		var p Person
		err := parser.Parse(map[string]any{"Alive": tt.mapValue}, &p)
		assert.NoErrorf(t, err, "%d. %#v", i, tt.mapValue)
		assert.Equalf(t, tt.structValue, p.Alive, "%d. %#v", i, tt.mapValue)
	}
}

func TestParserParsesIntoStruct_AnyField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		Name  string
		Other any
	}

	var p Person
	err := parser.Parse(map[string]any{"Name": "John", "Other": map[string]string{"foo": "bar", "baz": "quz"}}, &p)
	assert.NoError(t, err)
	assert.Equal(t, "John", p.Name)
	assert.Equal(t, map[string]any{"foo": "bar", "baz": "quz"}, p.Other)
}

func TestParserParsesIntoString(t *testing.T) {
	parser := &structify.Parser{}

	{
		var s string
		err := parser.Parse("foo", &s)
		assert.NoError(t, err)
		assert.Equal(t, "foo", s)
	}

	{
		var i64 int64
		err := parser.Parse("42", &i64)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), i64)
	}

	{
		var f64 float64
		err := parser.Parse("42.5", &f64)
		assert.NoError(t, err)
		assert.Equal(t, float64(42.5), f64)
	}
}

func TestParserParsesIntoInteger(t *testing.T) {
	parser := &structify.Parser{}

	{
		var n int8
		err := parser.Parse("4", &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}

	{
		var n int32
		err := parser.Parse(int16(4), &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}

	{
		var n int32
		err := parser.Parse(float64(4), &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}

	{
		var n int16
		err := parser.Parse(float32(4), &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}
}

func TestParserParsesIntoFloat(t *testing.T) {
	parser := &structify.Parser{}

	{
		var n float64
		err := parser.Parse("4", &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}

	{
		var n float32
		err := parser.Parse("4", &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}
}

func TestParserParsesIntoSlice(t *testing.T) {
	parser := &structify.Parser{}

	{
		source := []any{"foo", "bar", "baz"}
		var target []string
		err := parser.Parse(source, &target)
		require.NoError(t, err)
		assert.Equal(t, len(source), len(target))
		for i := 0; i < len(source) && i < len(target); i++ {
			assert.Equalf(t, source[i], target[i], "%d", i)
		}
	}

	{
		source := []int32{1, 2, 3}
		var target []string
		err := parser.Parse(source, &target)
		require.NoError(t, err)
		assert.Equal(t, []string{"1", "2", "3"}, target)
	}

	{
		source := []any{1.1, 2.2, 3.3}
		var target []float64
		err := parser.Parse(source, &target)
		require.NoError(t, err)
		assert.Equal(t, []float64{1.1, 2.2, 3.3}, target)
	}

	{
		source := []any{1.1, 2.2, 3.3}
		var target []*float64
		err := parser.Parse(source, &target)
		require.NoError(t, err)
		for i := 0; i < len(source); i++ {
			assert.Equalf(t, source[i], *target[i], "%d", i)
		}
	}

	{
		source := []any{1.1, nil, 2.2, nil, 3.3}
		var target []*float64
		err := parser.Parse(source, &target)
		require.NoError(t, err)
		for i := 0; i < len(source); i++ {
			require.Equalf(t, source[i] == nil, target[i] == nil, "%d", i)
			if source[i] != nil {
				assert.Equalf(t, source[i], *target[i], "%d", i)
			}
		}
	}
}

func TestParserParseReturnsSliceAssignmentError(t *testing.T) {
	parser := &structify.Parser{}

	source := []any{42, "bar", "7", "baz"}
	var target []int32
	err := parser.Parse(source, &target)
	require.Error(t, err)
	var sliceAssignmentError *structify.SliceAssignmentError
	require.ErrorAs(t, err, &sliceAssignmentError)
	elementErrors := sliceAssignmentError.ElementErrors()
	require.Len(t, elementErrors, 2)
	require.Equal(t, 1, elementErrors[0].Index)
	require.ErrorIs(t, elementErrors[0].Err, structify.ErrCannotConvertToInteger)
	require.Equal(t, 3, elementErrors[1].Index)
	require.ErrorIs(t, elementErrors[1].Err, structify.ErrCannotConvertToInteger)

	indexErrorMap := sliceAssignmentError.IndexErrorMap()
	require.Len(t, indexErrorMap, 2)
	require.ErrorIs(t, indexErrorMap[1], structify.ErrCannotConvertToInteger)
	require.ErrorIs(t, indexErrorMap[3], structify.ErrCannotConvertToInteger)
}

func TestParserParsesIntoAny(t *testing.T) {
	parser := &structify.Parser{}

	{
		source := "foo"
		var target any
		err := parser.Parse(source, &target)
		assert.NoError(t, err)
		assert.Equal(t, source, target)
	}

	{
		source := map[string]string{"foo": "bar", "baz": "quz"}
		var target any
		err := parser.Parse(source, &target)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar", "baz": "quz"}, target)
	}

	{
		source := map[string]any{"foo": "bar", "baz": "quz", "n": int32(42), "slice": []int32{1, 2, 3}, "nested": []any{[]int32{4, 5, 6}}}
		var target any
		err := parser.Parse(source, &target)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar", "baz": "quz", "n": int64(42), "slice": []any{int64(1), int64(2), int64(3)}, "nested": []any{[]any{int64(4), int64(5), int64(6)}}}, target)
	}
}

type testStructifyScanner string

func (tss *testStructifyScanner) StructifyScan(parser *structify.Parser, source any) error {
	*(*string)(tss) = fmt.Sprintf("%v %v", source, source)
	return nil
}

func (tss *testStructifyScanner) Scan(value any) error {
	return fmt.Errorf("should never be called because also implements StructifyScanner")
}

func TestParserParsesIntoStructifyScanner(t *testing.T) {
	parser := &structify.Parser{}

	{
		var tss testStructifyScanner
		err := parser.Parse("4", &tss)
		assert.NoError(t, err)
		assert.EqualValues(t, "4 4", string(tss))
	}

	{
		var tss testStructifyScanner
		err := parser.Parse(42, &tss)
		assert.NoError(t, err)
		assert.EqualValues(t, "42 42", string(tss))
	}
}

type testScanner string

func (ts *testScanner) Scan(value any) error {
	*(*string)(ts) = fmt.Sprintf("%v %v", value, value)
	return nil
}

func TestParserParsesIntoScanner(t *testing.T) {
	parser := &structify.Parser{}

	{
		var ts testScanner
		err := parser.Parse("4", &ts)
		assert.NoError(t, err)
		assert.EqualValues(t, "4 4", string(ts))
	}

	{
		var ts testScanner
		err := parser.Parse(42, &ts)
		assert.NoError(t, err)
		assert.EqualValues(t, "42 42", string(ts))
	}

	{
		var ns sql.NullString
		err := parser.Parse(nil, &ns)
		assert.NoError(t, err)
		assert.EqualValues(t, sql.NullString{}, ns)
	}

	{
		var ni64 sql.NullInt64
		err := parser.Parse(42, &ni64)
		assert.NoError(t, err)
		assert.EqualValues(t, sql.NullInt64{Int64: 42, Valid: true}, ni64)
	}
}

func TestParserParsesUsesRegisteredTypeScannerForNewType(t *testing.T) {
	parser := &structify.Parser{}
	parser.RegisterTypeScanner(new(time.Time), func(parser *structify.Parser, source, target any) error {
		seconds, err := strconv.ParseInt(fmt.Sprint(source), 10, 64)
		if err != nil {
			return err
		}

		*(target.(*time.Time)) = time.Unix(seconds, 0)
		return nil
	})
	var tm time.Time
	err := parser.Parse("1676164903", &tm)
	assert.NoError(t, err)
	assert.True(t, tm.Equal(time.Unix(1676164903, 0)))
}

func TestParserParsesUsesRegisteredTypeScannerToOverrideType(t *testing.T) {
	parser := &structify.Parser{}
	parser.RegisterTypeScanner(new(string), func(parser *structify.Parser, source, target any) error {
		*(target.(*string)) = "overridden"
		return nil
	})
	var s string
	err := parser.Parse("foobar", &s)
	assert.NoError(t, err)
	assert.Equal(t, "overridden", s)
}
