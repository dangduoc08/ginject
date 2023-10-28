package context

import (
	"go/token"
	"reflect"

	"github.com/dangduoc08/gooh/utils"
)

const (
	tagBind = "bind"
)

func BindStruct(d map[string]any, s any) any {
	structureType := reflect.TypeOf(s)
	newStructuredData := reflect.New(structureType)
	setValueToStructField := setValueToStructField(newStructuredData)

	for i := 0; i < structureType.NumField(); i++ {
		structField := structureType.Field(i)
		setValueToStructField := setValueToStructField(i)

		if !token.IsExported(structField.Name) {
			continue
		}

		if bindValues, ok := structField.Tag.Lookup(tagBind); ok {
			bindParams := getTagParams(bindValues)

			if len(bindParams) > 0 {
				_, bindedField := getTagParamIndex(bindParams[0])
				if bindedValue, ok := d[bindedField]; ok {
					// fmt.Println(bindedField, bindedValue, reflect.TypeOf(bindedValue).Kind())
					switch structField.Type.Kind() {

					case reflect.Bool:
						if boolean, ok := bindedValue.(bool); ok {
							setValueToStructField(boolean)
						}
						continue

					case
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
						reflect.Complex128:
						if f64, ok := bindedValue.(float64); ok {
							setValueToStructField(utils.NumF64ToAnyNum(f64, structField.Type.Kind()))
						}
						continue

					case reflect.String:
						if str, ok := bindedValue.(string); ok {
							setValueToStructField(str)
						}
						continue

					case reflect.Interface:
						setValueToStructField(bindedValue)
						continue

					case reflect.Slice:
						if bindedValue, ok := bindedValue.([]any); ok {
							setValueToStructField(
								bindArray(
									bindedValue,
									structField.Type,
								),
							)
						}
						continue

					case reflect.Map:
						if bindedValue, ok := bindedValue.(map[string]any); ok {
							setValueToStructField(
								bindMap(
									bindedValue,
									structField.Type,
								),
							)
						}
						continue

					case reflect.Struct:
						setValueToStructField(
							BindStruct(
								bindedValue.(map[string]any),
								newStructuredData.Elem().Field(i).Interface(),
							))
						continue
					}
				}
			}
		}
	}

	return reflect.Indirect(newStructuredData).Interface()
}
