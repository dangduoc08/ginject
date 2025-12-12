package ctx

import (
	"go/token"
	"mime/multipart"
	"reflect"

	"github.com/dangduoc08/ginject/utils"
)

func BindFile(f File, s any) (map[string][]*DataFile, any) {
	structureType := reflect.TypeOf(s)
	newStructuredData := reflect.New(structureType)
	setValueToStructField := setValueToStructField(newStructuredData)
	filteredFile := map[string][]*DataFile{}

	for i := 0; i < structureType.NumField(); i++ {
		structField := structureType.Field(i)
		setValueToStructField := setValueToStructField(i)
		if !token.IsExported(structField.Name) {
			continue
		}

		if bindValues, ok := structField.Tag.Lookup(TagBind); ok {
			bindParams := GetTagParams(bindValues)
			if len(bindParams) > 0 {
				bindedIndex, bindedField := GetTagParamIndex(bindParams[0])
				if bindedValue, ok := f[bindedField]; ok {
					switch structField.Type.Kind() {

					case reflect.Ptr:
						if len(bindedValue) > 0 {
							if fileHeader, ok := utils.ArrGet(bindedValue, bindedIndex); ok {
								dataFile := []*DataFile{
									{
										FileHeader: fileHeader,
										Index:      bindedIndex,
										Size:       fileHeader.Size,
										Total:      1,
										Key:        bindedField,
										Filename:   fileHeader.Filename,
										Type:       fileHeader.Header.Get("Content-Type"),
									},
								}

								filteredFile[bindedField] = dataFile
								setValueToStructField(dataFile[0])
							}
						}
						continue

					case reflect.Slice:
						dataFile := utils.ArrMap(
							bindedValue,
							func(fileHeader *multipart.FileHeader, index int) *DataFile {
								return &DataFile{
									FileHeader: fileHeader,
									Index:      index,
									Size:       fileHeader.Size,
									Total:      len(bindedValue),
									Key:        bindedField,
									Filename:   fileHeader.Filename,
									Type:       fileHeader.Header.Get("Content-Type"),
								}
							})

						filteredFile[bindedField] = dataFile
						setValueToStructField(dataFile)
						continue
					}
				}
			}
		}
	}

	return filteredFile, reflect.Indirect(newStructuredData).Interface()
}
