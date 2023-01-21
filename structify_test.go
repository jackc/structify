package structify_test

import (
	"testing"

	"github.com/jackc/structify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapFieldWithoutTagNameVariants(t *testing.T) {
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
		err := structify.Map(map[string]any{tt.key: "Jack"}, &p)
		require.NoErrorf(t, err, "%d. %s", i, tt.key)
		assert.Equalf(t, "Jack", p.FirstName, "%d. %s did not map to FirstName field", i, tt.key)
	}
}

func TestMapFieldWithTag(t *testing.T) {
	type Person struct {
		FirstName string `structify:"name"`
	}

	var p Person
	err := structify.Map(map[string]any{"name": "Jack"}, &p)
	require.NoError(t, err)
	assert.Equal(t, "Jack", p.FirstName)
}

func TestMapMissingRequiredField(t *testing.T) {
	type Person struct {
		FirstName string
		LastName  string
	}

	var p Person
	err := structify.Map(map[string]any{"name": "Jack"}, &p)
	require.Error(t, err)
}

func TestMapSkippedField(t *testing.T) {
	type Person struct {
		FirstName string
		LastName  string `structify:"-"`
	}

	var p Person
	err := structify.Map(map[string]any{"FirstName": "Jack"}, &p)
	require.NoError(t, err)
	assert.Equal(t, "Jack", p.FirstName)
}
