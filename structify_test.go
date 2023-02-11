package structify_test

import (
	"testing"

	"github.com/jackc/structify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserParseStructFieldWithoutTagNameVariants(t *testing.T) {
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

func TestParserParseStructFieldWithTag(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string `structify:"name"`
	}

	var p Person
	err := parser.Parse(map[string]any{"name": "Jack"}, &p)
	require.NoError(t, err)
	assert.Equal(t, "Jack", p.FirstName)
}

func TestParserParseStructMissingRequiredField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string
		LastName  string
	}

	var p Person
	err := parser.Parse(map[string]any{"name": "Jack"}, &p)
	require.Error(t, err)
}

func TestParserParseStructSkippedField(t *testing.T) {
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

func TestParserParseNestedStruct(t *testing.T) {
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

func TestParserParseArrayOfStruct(t *testing.T) {
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

func TestParserParseArrayOfPointerToStruct(t *testing.T) {
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

func TestParserMapInt32Field(t *testing.T) {
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

func TestParserMapFloat64Field(t *testing.T) {
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

func TestParserMapBoolField(t *testing.T) {
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

func TestParserMapAnyField(t *testing.T) {
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

func TestParserParseString(t *testing.T) {
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

func TestParserParseInteger(t *testing.T) {
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
}

func TestParserParseFloat(t *testing.T) {
	parser := &structify.Parser{}

	{
		var n float64
		err := parser.Parse("4", &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}

	{
		var n float64
		err := parser.Parse("4", &n)
		assert.NoError(t, err)
		assert.EqualValues(t, 4, n)
	}
}

func TestParserParseSlice(t *testing.T) {
	parser := &structify.Parser{}

	{
		src := []any{"foo", "bar", "baz"}
		var dst []string
		err := parser.Parse(src, &dst)
		assert.NoError(t, err)
		assert.Equal(t, len(src), len(dst))
		for i := 0; i < len(src) && i < len(dst); i++ {
			assert.Equalf(t, src[i], dst[i], "%d", i)
		}
	}

	{
		src := []int32{1, 2, 3}
		var dst []string
		err := parser.Parse(src, &dst)
		assert.NoError(t, err)
		assert.Equal(t, []string{"1", "2", "3"}, dst)
	}

	{
		src := []any{1.1, 2.2, 3.3}
		var dst []float64
		err := parser.Parse(src, &dst)
		assert.NoError(t, err)
		assert.Equal(t, []float64{1.1, 2.2, 3.3}, dst)
	}
}

func TestParserParseToAny(t *testing.T) {
	parser := &structify.Parser{}

	{
		src := "foo"
		var dst any
		err := parser.Parse(src, &dst)
		assert.NoError(t, err)
		assert.Equal(t, src, dst)
	}

	{
		src := map[string]string{"foo": "bar", "baz": "quz"}
		var dst any
		err := parser.Parse(src, &dst)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar", "baz": "quz"}, dst)
	}

	{
		src := map[string]any{"foo": "bar", "baz": "quz", "n": int32(42), "slice": []int32{1, 2, 3}, "nested": []any{[]int32{4, 5, 6}}}
		var dst any
		err := parser.Parse(src, &dst)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar", "baz": "quz", "n": int64(42), "slice": []any{int64(1), int64(2), int64(3)}, "nested": []any{[]any{int64(4), int64(5), int64(6)}}}, dst)
	}
}
