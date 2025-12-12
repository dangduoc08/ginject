package devtool

import (
	"go/token"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
)

var format = map[reflect.Kind]string{
	reflect.Bool:       "boolean",
	reflect.Int:        "number",
	reflect.Int8:       "number",
	reflect.Int16:      "number",
	reflect.Int32:      "number",
	reflect.Int64:      "number",
	reflect.Uint:       "number",
	reflect.Uint8:      "number",
	reflect.Uint16:     "number",
	reflect.Uint32:     "number",
	reflect.Uint64:     "number",
	reflect.Float32:    "number",
	reflect.Float64:    "number",
	reflect.Complex64:  "number",
	reflect.Complex128: "number",
	reflect.String:     "string",
	reflect.Interface:  "any",
	reflect.Slice:      "array",
	reflect.Map:        "map",
	reflect.Struct:     "object",
	reflect.Ptr:        "object",
}

var typ = map[reflect.Kind]string{
	reflect.Bool:       "boolean",
	reflect.Int:        "integer",
	reflect.Int8:       "integer-8",
	reflect.Int16:      "integer-16",
	reflect.Int32:      "integer-32",
	reflect.Int64:      "integer-64",
	reflect.Uint:       "unsigned-integer",
	reflect.Uint8:      "unsigned-integer8",
	reflect.Uint16:     "unsigned-integer16",
	reflect.Uint32:     "unsigned-integer32",
	reflect.Uint64:     "unsigned-integer64",
	reflect.Float32:    "floating-point-32",
	reflect.Float64:    "floating-point-64",
	reflect.Complex64:  "complex-number-64",
	reflect.Complex128: "complex-number-128",
	reflect.String:     "string",
	reflect.Interface:  "any",
	reflect.Slice:      "array",
	reflect.Map:        "object",
	reflect.Struct:     "object",
	reflect.Ptr:        "object",
}

func GenerateSchema(s reflect.Type, tag string) []*Schema {
	schema := []*Schema{}

	for i := 0; i < s.NumField(); i++ {
		structField := s.Field(i)

		if !token.IsExported(structField.Name) {
			continue
		}

		if bindValues, ok := structField.Tag.Lookup(ctx.TagBind); ok {
			bindParams := ctx.GetTagParams(bindValues)

			if len(bindParams) > 0 {
				_, bindedField := ctx.GetTagParamIndex(bindParams[0])

				switch structField.Type.Kind() {
				case
					reflect.Bool,
					reflect.Int,
					reflect.Int8,
					reflect.Int16,
					reflect.Int32,
					reflect.Int64,
					reflect.Uint,
					reflect.Uint8,
					reflect.Uint16,
					reflect.Uint32,
					reflect.Uint64,
					reflect.Float32,
					reflect.Float64,
					reflect.Complex64,
					reflect.Complex128,
					reflect.String,
					reflect.Interface:
					schema = append(schema, &Schema{
						Name:   bindedField,
						Type:   typ[structField.Type.Kind()],
						Format: format[structField.Type.Kind()],
					})

				case reflect.Slice:
					schema = append(schema, &Schema{
						Name:   bindedField,
						Type:   typ[structField.Type.Kind()],
						Format: format[structField.Type.Kind()],
						Item:   explainObj(structField.Type.Elem(), tag),
					})

				case reflect.Map:
					schema = append(schema, &Schema{
						Name:       bindedField,
						Type:       typ[structField.Type.Kind()],
						Format:     format[reflect.Struct],
						Properties: []*Schema{explainObj(structField.Type.Elem(), tag)},
					})

				case reflect.Struct:
					schema = append(schema, &Schema{
						Name:       bindedField,
						Type:       typ[structField.Type.Kind()],
						Format:     format[structField.Type.Kind()],
						Properties: GenerateSchema(structField.Type, tag),
					})

				case reflect.Ptr:
					if structField.Type.Elem().Kind() == reflect.Struct {
						schema = append(schema, &Schema{
							Name:       bindedField,
							Type:       typ[structField.Type.Kind()],
							Format:     format[structField.Type.Kind()],
							Properties: GenerateSchema(structField.Type.Elem(), tag),
						})
					}
				}
			}
		}
	}

	return schema
}

func explainObj(ob reflect.Type, tag string) *Schema {
	switch ob.Kind() {
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.String,
		reflect.Interface:
		return &Schema{
			Type:   typ[ob.Kind()],
			Format: format[ob.Kind()],
		}

	case reflect.Slice:
		return &Schema{
			Type:   typ[ob.Kind()],
			Format: format[ob.Kind()],
			Item:   explainObj(ob.Elem(), tag),
		}

	case reflect.Map:
		return &Schema{
			Type:       typ[ob.Kind()],
			Format:     format[reflect.Struct],
			Properties: []*Schema{explainObj(ob.Elem(), tag)},
		}

	case reflect.Struct:
		return &Schema{
			Type:       typ[ob.Kind()],
			Format:     format[ob.Kind()],
			Properties: GenerateSchema(ob, tag),
		}

	case reflect.Ptr:
		if ob.Elem().Kind() == reflect.Struct {
			return &Schema{
				Type:       typ[ob.Kind()],
				Format:     format[ob.Kind()],
				Properties: GenerateSchema(ob.Elem(), tag),
			}
		}
	}

	return nil
}
