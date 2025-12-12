package ctx

import (
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/dangduoc08/ginject/utils"
)

/*
Support types:

  - Bool

  - Int

  - Int8

  - Int16

  - Int32

  - Int64

  - Uint

  - Uint8

  - Uint16

  - Uint32

  - Uint64

  - Float32

  - Float64

  - Complex64

  - Complex128

  - String

  - Interface

  - Slice
*/

func BindStrArr(d map[string][]string, fls *[]FieldLevel, s any) (any, []FieldLevel) {
	structureType := reflect.TypeOf(s)
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
				bindedIndex, bindedField := GetTagParamIndex(bindParams[0])
				if bindedValues, ok := d[bindedField]; ok {
					fl := FieldLevel{
						tag:       bindedField,
						nestedTag: bindedField,
						ns:        structureType.Name() + "." + structField.Name,
						field:     structField.Name,
						index:     bindedIndex,
						kind:      structField.Type.Kind(),
						typ:       structField.Type,
						isVal:     true,
					}

					// check each type of struct
					switch structField.Type.Kind() {
					case reflect.Bool:
						if boolStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := false
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if boolean, err := strconv.ParseBool(boolStr); err != nil {
							val := false
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := boolean
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Int:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := 0
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if intNum, err := strconv.Atoi(intStr); err != nil {
							val := 0
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := intNum
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Int8:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := int8(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 8); err != nil {
							val := int8(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := int8(i64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Int16:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := int16(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 16); err != nil {
							val := int16(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := int16(i64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Int32:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := int32(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 32); err != nil {
							val := int32(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := int32(i64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Int64:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := int64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 64); err != nil {
							val := int64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := i64
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Uint:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := uint(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 0); err != nil {
							val := uint(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := uint(u64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Uint8:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := uint8(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 8); err != nil {
							val := uint8(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := uint8(u64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Uint16:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := uint16(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 16); err != nil {
							val := uint16(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := uint16(u64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Uint32:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := uint32(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 32); err != nil {
							val := uint32(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := uint32(u64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Uint64:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := uint64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 64); err != nil {
							val := uint64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := u64
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Float32:
						if fStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := float32(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if f64, err := strconv.ParseFloat(fStr, 32); err != nil {
							val := float32(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := float32(f64)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Float64:
						if fStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := float64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if f64, err := strconv.ParseFloat(fStr, 64); err != nil {
							val := float64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := f64
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Complex64:
						if cStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := complex64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if c128, err := strconv.ParseComplex(strings.ReplaceAll(cStr, " ", ""), 64); err != nil {
							val := complex64(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := complex64(c128)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Complex128:
						if cStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := complex128(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else if c128, err := strconv.ParseComplex(strings.ReplaceAll(cStr, " ", ""), 128); err != nil {
							val := complex128(0)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := c128
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.String:
						if str, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := ""
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := str
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Interface:
						if strVal, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							val := ""
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						} else {
							val := strVal
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}

					case reflect.Slice:
						switch structField.Type.Elem().Kind() {
						case reflect.Bool:
							val := utils.ArrStrParseBool(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Int:
							val := utils.ArrStrParseInt(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Int8:
							val := utils.ArrStrParseInt8(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Int16:
							val := utils.ArrStrParseInt16(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Int32:
							val := utils.ArrStrParseInt32(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Int64:
							val := utils.ArrStrParseInt64(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Uint:
							val := utils.ArrStrParseUint(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Uint8:
							val := utils.ArrStrParseUint8(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Uint16:
							val := utils.ArrStrParseUint16(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Uint32:
							val := utils.ArrStrParseUint32(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Uint64:
							val := utils.ArrStrParseUint64(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Float32:
							val := utils.ArrStrParseFloat32(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Float64:
							val := utils.ArrStrParseFloat64(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Complex64:
							val := utils.ArrStrParseComplex64(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Complex128:
							val := utils.ArrStrParseComplex128(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.String:
							val := bindedValues
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						case reflect.Interface:
							val := utils.ArrStrParseAny(bindedValues)
							fl.val = val
							*fls = append(*fls, fl)
							setValueToStructField(val)
							continue
						}
					}
				} else {
					*fls = append(*fls, FieldLevel{
						tag:       bindedField,
						nestedTag: bindedField,
						ns:        structureType.Name() + "." + structField.Name,
						field:     structField.Name,
						index:     bindedIndex,
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
