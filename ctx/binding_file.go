package ctx

import (
	"go/token"
	"reflect"

	"github.com/dangduoc08/gooh/utils"
)

func BindFile(f File, s any) ([]string, any) {
	structureType := reflect.TypeOf(s)
	newStructuredData := reflect.New(structureType)
	setValueToStructField := setValueToStructField(newStructuredData)
	keys := []string{}

	for i := 0; i < structureType.NumField(); i++ {
		structField := structureType.Field(i)
		setValueToStructField := setValueToStructField(i)
		if !token.IsExported(structField.Name) {
			continue
		}

		if bindValues, ok := structField.Tag.Lookup(tagBind); ok {
			bindParams := getTagParams(bindValues)
			if len(bindParams) > 0 {
				bindedIndex, bindedField := getTagParamIndex(bindParams[0])
				keys = append(keys, bindedField)

				if bindedValue, ok := f[bindedField]; ok {
					switch structField.Type.Kind() {

					case reflect.Ptr:
						if len(bindedValue) > 0 {
							if fileHeader, ok := utils.ArrGet(bindedValue, bindedIndex); ok {
								setValueToStructField(fileHeader)
							}
						}
						continue

					case reflect.Slice:
						setValueToStructField(bindedValue)
						continue
					}
				}
			}
		}
	}

	return keys, reflect.Indirect(newStructuredData).Interface()
}
