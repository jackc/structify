package structify_test

import (
	"testing"

	"github.com/jackc/structify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserMapFieldWithoutTagNameVariants(t *testing.T) {
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
		err := parser.Map(map[string]any{tt.key: "Jack"}, &p)
		require.NoErrorf(t, err, "%d. %s", i, tt.key)
		assert.Equalf(t, "Jack", p.FirstName, "%d. %s did not map to FirstName field", i, tt.key)
	}
}

func TestParserMapFieldWithTag(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string `structify:"name"`
	}

	var p Person
	err := parser.Map(map[string]any{"name": "Jack"}, &p)
	require.NoError(t, err)
	assert.Equal(t, "Jack", p.FirstName)
}

func TestParserMapMissingRequiredField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string
		LastName  string
	}

	var p Person
	err := parser.Map(map[string]any{"name": "Jack"}, &p)
	require.Error(t, err)
}

func TestParserMapSkippedField(t *testing.T) {
	parser := &structify.Parser{}

	type Person struct {
		FirstName string
		LastName  string `structify:"-"`
	}

	var p Person
	err := parser.Map(map[string]any{"FirstName": "Jack"}, &p)
	require.NoError(t, err)
	assert.Equal(t, "Jack", p.FirstName)
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
		err := parser.Map(map[string]any{"Age": tt.mapValue}, &p)
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
		err := parser.Map(map[string]any{"Age": tt.mapValue}, &p)
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
		err := parser.Map(map[string]any{"Alive": tt.mapValue}, &p)
		assert.NoErrorf(t, err, "%d. %#v", i, tt.mapValue)
		assert.Equalf(t, tt.structValue, p.Alive, "%d. %#v", i, tt.mapValue)
	}
}
