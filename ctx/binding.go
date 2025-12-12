package ctx

import (
	"go/token"
	"reflect"

	"github.com/dangduoc08/ginject/utils"
)

const (
	TagBind = "bind"
)

/*
Support types:

	- Struct pointer

*/

func BindStruct(d map[string]any, fls *[]FieldLevel, s any, parentNS string, parentTag string) (any, []FieldLevel) {

	// check struct pointer case
	// when recursive will pass s as reflect.Type
	var structureType reflect.Type
	if sType, ok := s.(reflect.Type); ok {
		structureType = sType
	} else {
		structureType = reflect.TypeOf(s)
	}

	newStructuredData := reflect.New(structureType)
	setValueToStructField := setValueToStructField(newStructuredData)

	for i := 0; i < structureType.NumField(); i++ {
		structField := structureType.Field(i)
		setValueToStructField := setValueToStructField(i)

		if !token.IsExported(structField.Name) {
			continue
		}

		if bindValues, ok := structField.Tag.Lookup(TagBind); ok {
			bindParams := GetTagParams(bindValues)

			if len(bindParams) > 0 {
				_, bindedField := GetTagParamIndex(bindParams[0])
				if bindedValue, ok := d[bindedField]; ok {
					ns := ""
					if parentNS != "" {
						ns = parentNS + "."
					}
					ns = ns + structureType.Name() + "." + structField.Name

					nestedTag := ""
					if parentTag != "" {
						nestedTag = parentTag + "."
					}
					nestedTag = nestedTag + bindedField

					fl := FieldLevel{
						tag:       bindedField,
						nestedTag: nestedTag,
						ns:        ns,
						field:     structField.Name,
						kind:      structField.Type.Kind(),
						typ:       structField.Type,
						isVal:     true,
					}

					switch structField.Type.Kind() {

					case reflect.Bool:
						if boolean, ok := bindedValue.(bool); ok {
							val := boolean
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
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
							val := utils.NumF64ToAnyNum(f64, structField.Type.Kind())
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
						}
						continue

					case reflect.String:
						if str, ok := bindedValue.(string); ok {
							val := str
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
						}
						continue

					case reflect.Interface:
						val := bindedValue
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue

					case reflect.Slice:
						if bindedValue, ok := bindedValue.([]any); ok {
							val := bindArray(
								bindedValue,
								fls,
								structField.Type,
								ns,
								nestedTag,
							)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
						}
						continue

					case reflect.Map:
						if bindedValue, ok := bindedValue.(map[string]any); ok {
							val := bindMap(
								bindedValue,
								fls,
								structField.Type,
								ns,
								nestedTag,
							)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
						}
						continue

					case reflect.Struct:
						val, _ := BindStruct(
							bindedValue.(map[string]any),
							fls,
							newStructuredData.Elem().Field(i).Interface(),
							ns,
							nestedTag,
						)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue

					case reflect.Ptr:
						// TODO: For the rest type pointers
						// need to enhance later

						// if structField.Type.Elem().Kind() == reflect.Float64 {
						// 	if f64, ok := bindedValue.(float64); ok {
						// 		val := utils.NumF64ToAnyNum(f64, structField.Type.Kind())
						// 		fl.val = val
						// 		*fls = append(*fls, fl)
						// 		setValueToStructField(&f64)
						// 	}
						// }

						if structField.Type.Elem().Kind() == reflect.Struct {
							if bindedValue, ok := bindedValue.(map[string]any); ok {
								val, _ := BindStruct(
									bindedValue,
									fls,
									structField.Type.Elem(),
									ns,
									nestedTag,
								)
								fl.val = val
								*fls = append(*fls, fl)
								setValueToStructField(fromStrucValueToStructPointerValue(val))
							}
						}
						continue
					}
				} else {
					ns := ""
					if parentNS != "" {
						ns = parentNS + "."
					}
					ns = ns + structureType.Name() + "." + structField.Name

					nestedTag := ""
					if parentTag != "" {
						nestedTag = parentTag + "."
					}
					nestedTag = nestedTag + bindedField

					*fls = append(*fls, FieldLevel{
						tag:       bindedField,
						nestedTag: nestedTag,
						ns:        ns,
						field:     structField.Name,
						kind:      structField.Type.Kind(),
						typ:       structField.Type,
						val:       nil,
						isVal:     false,
					})
				}
			}
		}
	}

	return reflect.Indirect(newStructuredData).Interface(), *fls
}
