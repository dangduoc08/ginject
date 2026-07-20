package ctx

import (
	"go/token"
	"mime/multipart"
	"reflect"

	"github.com/dangduoc08/ginject/internal/slice"
)

func BindFile(f File, s any) (map[string][]*DataFile, any) {
	structureType := reflect.TypeOf(s)
	newStructuredData := reflect.New(structureType)
	setValueToStructField := setValueToStructField(newStructuredData)
	filteredFile := map[string][]*DataFile{}
	fieldTags := getFieldBindTags(structureType)

	for i := 0; i < structureType.NumField(); i++ {
		structField := structureType.Field(i)
		setValueToStructField := setValueToStructField(i)
		if !token.IsExported(structField.Name) {
			continue
		}

		if ft := fieldTags[i]; ft.ok {
			bindedIndex, bindedField := ft.index, ft.field
			if bindedValue, ok := f[bindedField]; ok {
				switch structField.Type.Kind() {

				case reflect.Ptr:
					if len(bindedValue) > 0 {
						if fileHeader, ok := slice.Get(bindedValue, bindedIndex); ok {
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
					dataFile := slice.Map(
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

	return filteredFile, reflect.Indirect(newStructuredData).Interface()
}
