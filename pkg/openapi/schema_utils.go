package openapi

import (
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GenerateSchema creates an OpenAPI schema from a Go struct using reflection
func GenerateSchema(v interface{}) *Schema {
	if v == nil {
		return nil
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return typeToSchema(t)
}

func typeToSchema(t reflect.Type) *Schema {
	// Handle special types
	if t == reflect.TypeOf(time.Time{}) {
		return &Schema{Type: "string", Format: "date-time"}
	}
	if t == reflect.TypeOf(uuid.UUID{}) {
		return &Schema{Type: "string", Format: "uuid"}
	}

	switch t.Kind() {
	case reflect.Struct:
		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
		}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			// Parse json tag name
			name := field.Name
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				name = parts[0]
			}

			propSchema := typeToSchema(field.Type)
			if propSchema != nil {
				schema.Properties[name] = propSchema
			}
		}
		return schema

	case reflect.Slice, reflect.Array:
		return &Schema{
			Type:  "array",
			Items: typeToSchema(t.Elem()),
		}

	case reflect.Map:
		return &Schema{
			Type: "object",
			// OpenAPI 3.0 supports additionalProperties for maps
		}

	case reflect.String:
		return &Schema{Type: "string"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}

	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}

	case reflect.Bool:
		return &Schema{Type: "boolean"}

	case reflect.Ptr:
		return typeToSchema(t.Elem())

	default:
		return &Schema{Type: "string"} // Fallback
	}
}
