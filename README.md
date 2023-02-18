[![Go Reference](https://pkg.go.dev/badge/github.com/jackc/structify.svg)](https://pkg.go.dev/github.com/jackc/structify)

# structify

structify is designed to parse loosely-typed, client-controlled input such as JSON or a web form submission into a
struct. It's purpose is to validate input shape and convert to a struct, not to validate input contents. For example,
validating and converting an input field to an integer, not validating that the integer is between 0 and 10.

## Example Usage

```go
type Person struct {
  FirstName string
  LastName string
}

var person Person
err := structify.Parse(map[string]any{"FirstName": "John", "LastName": "Smith"}, &person)
```

## Features

* Supports nested structs
* Supports slices
* Automatically maps between camelcase and snakecase. That is, `first_name` will be mapped to `FirstName` without needing a struct field tag
* Structured errors that accumulate all field errors
* Automatically uses database/sql.Scanner interface if available
* Can define scanner method on types or register on parser when not convenient to add method to type
* Includes generic Optional type
