package structify_test

import (
	"testing"

	"github.com/jackc/structify"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	type Person struct {
		Name string
	}

	var p Person
	structify.Map(map[string]any{"Name": "Jack"}, &p)
	assert.Equal(t, "Jack", p.Name)
}
