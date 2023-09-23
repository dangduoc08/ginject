package context

import (
	"fmt"
	"go/token"
	"reflect"
	"strconv"
)

const (
	tagBind = "bind"
)

func Bind(d, s any, isParseArray, isParseString bool) any {
	structureType := reflect.TypeOf(s)
	newStructuredData := reflect.New(structureType)

	// write code for body first
	if obj, ok := d.(map[string]any); ok {
		for i := 0; i < structureType.NumField(); i++ {
			structField := structureType.Field(i)
			if !token.IsExported(structField.Name) {
				continue
			}

			// bind value is the field of
			// body, query, header, param
			// which struct want to map to
			if bindValues, ok := structField.Tag.Lookup(tagBind); ok {
				bindParams := getTagParams(bindValues)

				if len(bindParams) > 0 {
					bindedIndex, bindedField := getTagParamIndex(bindParams[0])

					if bindedValue, ok := obj[bindedField]; ok {
						switch structField.Type.Kind() {

						case reflect.Bool:

							// value in body
							// it's actually boolean value
							if boolBindedValue, ok := bindedValue.(bool); ok {
								bindedValue = boolBindedValue

								// value in header param query
								// it's boolean string
								// ex: "true" or "false"
								// if not matched, then fallback to default struct value
								// meaning = false
							} else if strBindedValue, ok := bindedValue.(string); ok && isParseString {
								bindedValue, _ = strconv.ParseBool(strBindedValue)

								// values in header param query
								// it's string array
								// get first index
							} else if bindedValues, ok := bindedValue.([]any); ok && isParseArray {

								// avoid index out range error
								if bindedIndex > len(bindedValues)-1 {
									bindedIndex = 0
								}

								if strBindedValue, ok := bindedValues[bindedIndex].(string); ok && isParseString && len(bindedValues) > 0 {
									bindedValue, _ = strconv.ParseBool(strBindedValue)
								} else if boolBindedValue, ok := bindedValues[bindedIndex].(bool); ok && len(bindedValues) > 0 {
									bindedValue = boolBindedValue
								} else {
									continue
								}
							} else {
								continue
							}

						case reflect.Int:

							// value in body
							// it's actually integer value
							if intBindedValue, ok := bindedValue.(int); ok {
								bindedValue = intBindedValue

								// number in map[string]any can be float64
							} else if f64BindedValue, ok := bindedValue.(float64); ok {
								bindedValue = int(f64BindedValue)

								// value is integer string
								// ex: "123" or "-123"
								// if not matched, then fallback to default struct value
								// meaning = 0
							} else if strBindedValue, ok := bindedValue.(string); ok && isParseString {
								integer, err := strconv.Atoi(strBindedValue)
								if err != nil {
									continue
								}
								bindedValue = integer

								// values in header param query
								// it's string array
								// get first index
							} else if bindedValues, ok := bindedValue.([]any); ok && isParseArray {
								if bindedIndex > len(bindedValues)-1 {

									// avoid index out range error
									bindedIndex = 0
								}

								if strBindedValue, ok := bindedValues[bindedIndex].(string); ok && isParseString && len(bindedValues) > 0 {
									integer, err := strconv.Atoi(strBindedValue)
									if err != nil {
										continue
									}
									bindedValue = integer
								} else if intBindedValue, ok := bindedValues[bindedIndex].(int); ok && len(bindedValues) > 0 {
									bindedValue = intBindedValue
								} else if f64BindedValue, ok := bindedValues[bindedIndex].(float64); ok && len(bindedValues) > 0 {
									bindedValue = int(f64BindedValue)
								} else {
									continue
								}
							} else {
								continue
							}

						case reflect.Int8:
							if i8BindedValue, ok := bindedValue.(int8); ok {
								bindedValue = i8BindedValue
							} else if f64BindedValue, ok := bindedValue.(float64); ok {
								bindedValue = int8(f64BindedValue)
							} else if strBindedValue, ok := bindedValue.(string); ok {
								i64, err := strconv.ParseInt(strBindedValue, 10, 8)
								if err != nil {
									continue
								}
								bindedValue = int8(i64)
							}

						case reflect.Int16:
							if i16BindedValue, ok := bindedValue.(int16); ok {
								bindedValue = i16BindedValue
							} else if f64BindedValue, ok := bindedValue.(float64); ok {
								bindedValue = int16(f64BindedValue)
							} else if strBindedValue, ok := bindedValue.(string); ok {
								i64, err := strconv.ParseInt(strBindedValue, 10, 16)
								if err != nil {
									continue
								}
								bindedValue = int16(i64)
							}

						case reflect.Int32:
							if i32BindedValue, ok := bindedValue.(int32); ok {
								bindedValue = i32BindedValue
							} else if f64BindedValue, ok := bindedValue.(float64); ok {
								bindedValue = int32(f64BindedValue)
							} else if strBindedValue, ok := bindedValue.(string); ok {
								i64, err := strconv.ParseInt(strBindedValue, 10, 32)
								if err != nil {
									continue
								}
								bindedValue = int32(i64)
							}

						case reflect.Int64:
							if i64BindedValue, ok := bindedValue.(int64); ok {
								bindedValue = i64BindedValue
							} else if f64BindedValue, ok := bindedValue.(float64); ok {
								bindedValue = int64(f64BindedValue)
							} else if strBindedValue, ok := bindedValue.(string); ok {
								i64, err := strconv.ParseInt(strBindedValue, 10, 64)
								if err != nil {
									continue
								}
								bindedValue = i64
							}

						case reflect.Uint:

						case reflect.Uint8:

						case reflect.Uint16:

						case reflect.Uint32:

						case reflect.Uint64:

						case reflect.Float32:
							if f32BindedValue, ok := bindedValue.(float32); ok {
								bindedValue = f32BindedValue
							} else if strBindedValue, ok := bindedValue.(string); ok {
								f64, err := strconv.ParseFloat(strBindedValue, 32)
								if err != nil {
									continue
								}
								bindedValue = float32(f64)
							}

						case reflect.Float64:
							if f64BindedValue, ok := bindedValue.(float64); ok {
								bindedValue = f64BindedValue
							} else if strBindedValue, ok := bindedValue.(string); ok {
								f64, err := strconv.ParseFloat(strBindedValue, 64)
								if err != nil {
									continue
								}
								fmt.Println(f64)
								bindedValue = f64
							}

						case reflect.Complex64:

						case reflect.Complex128:

						case reflect.Array:

						case reflect.Interface:
							// just using current bindedValue

						case reflect.Map:
							if objectBindedValue, ok := bindedValue.(map[string]any); ok {

								// structField = map[string]struct{} case
								if structField.Type.Elem().Kind() == reflect.Struct {
									mapType := reflect.MapOf(reflect.TypeOf(""), structField.Type.Elem())
									mapStruct := reflect.MakeMap(mapType)

									for objectBindedValueKey, objectBindedValueValue := range objectBindedValue {
										eachMapValue := Bind(
											objectBindedValueValue,
											reflect.Indirect(reflect.New(structField.Type.Elem())).Interface(),
											isParseArray,
											isParseString,
										)
										mapStruct.SetMapIndex(reflect.ValueOf(objectBindedValueKey), reflect.ValueOf(eachMapValue))
									}

									bindedValue = mapStruct.Interface()
								}

							}

						case reflect.Slice:
							if sliceBindedValue, ok := bindedValue.([]any); ok {
								bindedValue = sliceBindedValue
							}

						case reflect.String:
							if strBindedValue, ok := bindedValue.(string); ok {
								bindedValue = strBindedValue
							}

						case reflect.Struct:
							bindedValue = Bind(
								bindedValue.(map[string]any),
								newStructuredData.Elem().Field(i).Interface(),
								isParseArray,
								isParseString,
							)

						case reflect.Invalid:
						case reflect.Chan:
						case reflect.Func:
						case reflect.Uintptr:
						case reflect.Pointer:
						case reflect.UnsafePointer:
						default:
							continue
						}

						fmt.Println(structField.Name, bindedValue)
						newStructuredData.Elem().Field(i).Set(reflect.ValueOf(bindedValue))
					}
				}
			}
		}
	}

	return reflect.Indirect(newStructuredData).Interface()
}
