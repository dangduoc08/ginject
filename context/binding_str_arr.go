package context

import (
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/dangduoc08/gooh/utils"
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

func bindStrArr(d map[string][]string, s any) any {
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
				bindedIndex, bindedField := getTagParamIndex(bindParams[0])
				if bindedValues, ok := d[bindedField]; ok {

					// check each type of struct
					switch structField.Type.Kind() {
					case reflect.Bool:
						if boolStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(false)
							continue
						} else if boolean, err := strconv.ParseBool(boolStr); err != nil {
							setValueToStructField(false)
							continue
						} else {
							setValueToStructField(boolean)
							continue
						}

					case reflect.Int:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(0)
							continue
						} else if intNum, err := strconv.Atoi(intStr); err != nil {
							setValueToStructField(0)
							continue
						} else {
							setValueToStructField(intNum)
							continue
						}

					case reflect.Int8:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(int8(0))
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 8); err != nil {
							setValueToStructField(int8(0))
							continue
						} else {
							setValueToStructField(int8(i64))
							continue
						}

					case reflect.Int16:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(int16(0))
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 16); err != nil {
							setValueToStructField(int16(0))
							continue
						} else {
							setValueToStructField(int16(i64))
							continue
						}

					case reflect.Int32:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(int32(0))
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 32); err != nil {
							setValueToStructField(int32(0))
							continue
						} else {
							setValueToStructField(int32(i64))
							continue
						}

					case reflect.Int64:
						if intStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(int64(0))
							continue
						} else if i64, err := strconv.ParseInt(intStr, 10, 64); err != nil {
							setValueToStructField(int64(0))
							continue
						} else {
							setValueToStructField(i64)
							continue
						}

					case reflect.Uint:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(uint(0))
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 0); err != nil {
							setValueToStructField(uint(0))
							continue
						} else {
							setValueToStructField(uint(u64))
							continue
						}

					case reflect.Uint8:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(uint8(0))
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 8); err != nil {
							setValueToStructField(uint8(0))
							continue
						} else {
							setValueToStructField(uint8(u64))
							continue
						}

					case reflect.Uint16:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(uint16(0))
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 16); err != nil {
							setValueToStructField(uint16(0))
							continue
						} else {
							setValueToStructField(uint16(u64))
							continue
						}

					case reflect.Uint32:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(uint32(0))
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 32); err != nil {
							setValueToStructField(uint32(0))
							continue
						} else {
							setValueToStructField(uint32(u64))
							continue
						}

					case reflect.Uint64:
						if uintStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(uint64(0))
							continue
						} else if u64, err := strconv.ParseUint(uintStr, 10, 64); err != nil {
							setValueToStructField(uint64(0))
							continue
						} else {
							setValueToStructField(u64)
							continue
						}

					case reflect.Float32:
						if fStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(float32(0))
							continue
						} else if f64, err := strconv.ParseFloat(fStr, 32); err != nil {
							setValueToStructField(float32(0))
							continue
						} else {
							setValueToStructField(float32(f64))
							continue
						}

					case reflect.Float64:
						if fStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(float64(0))
							continue
						} else if f64, err := strconv.ParseFloat(fStr, 64); err != nil {
							setValueToStructField(float64(0))
							continue
						} else {
							setValueToStructField(f64)
							continue
						}

					case reflect.Complex64:
						if cStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(complex64(0))
							continue
						} else if c128, err := strconv.ParseComplex(strings.ReplaceAll(cStr, " ", ""), 64); err != nil {
							setValueToStructField(complex64(0))
							continue
						} else {
							setValueToStructField(complex64(c128))
							continue
						}

					case reflect.Complex128:
						if cStr, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField(complex128(0))
							continue
						} else if c128, err := strconv.ParseComplex(strings.ReplaceAll(cStr, " ", ""), 128); err != nil {
							setValueToStructField(complex128(0))
							continue
						} else {
							setValueToStructField(c128)
							continue
						}

					case reflect.String:
						if str, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField("")
							continue
						} else {
							setValueToStructField(str)
							continue
						}

					case reflect.Interface:
						if strVal, ok := utils.ArrGet(bindedValues, bindedIndex); !ok {
							setValueToStructField("")
							continue
						} else {
							setValueToStructField(strVal)
							continue
						}

					case reflect.Slice:
						// if len(bindParams) > 1 {
						// 	// key, value := getTagKV(bindParams[1])
						// 	// fmt.Println("key=", key, " value=", value)
						// }
						switch structField.Type.Elem().Kind() {
						case reflect.Bool:
							setValueToStructField(utils.ArrParseBool(bindedValues))
							continue
						case reflect.Int:
							setValueToStructField(utils.ArrParseInt(bindedValues))
							continue
						case reflect.Int8:
							setValueToStructField(utils.ArrParseInt8(bindedValues))
							continue
						case reflect.Int16:
							setValueToStructField(utils.ArrParseInt16(bindedValues))
							continue
						case reflect.Int32:
							setValueToStructField(utils.ArrParseInt32(bindedValues))
							continue
						case reflect.Int64:
							setValueToStructField(utils.ArrParseInt64(bindedValues))
							continue
						case reflect.Uint:
							setValueToStructField(utils.ArrParseUint(bindedValues))
							continue
						case reflect.Uint8:
							setValueToStructField(utils.ArrParseUint8(bindedValues))
							continue
						case reflect.Uint16:
							setValueToStructField(utils.ArrParseUint16(bindedValues))
							continue
						case reflect.Uint32:
							setValueToStructField(utils.ArrParseUint32(bindedValues))
							continue
						case reflect.Uint64:
							setValueToStructField(utils.ArrParseUint64(bindedValues))
							continue
						case reflect.Float32:
							setValueToStructField(utils.ArrParseFloat32(bindedValues))
							continue
						case reflect.Float64:
							setValueToStructField(utils.ArrParseFloat64(bindedValues))
							continue
						case reflect.Complex64:
							setValueToStructField(utils.ArrParseComplex64(bindedValues))
							continue
						case reflect.Complex128:
							setValueToStructField(utils.ArrParseComplex128(bindedValues))
							continue
						case reflect.String:
							setValueToStructField(bindedValues)
							continue
						case reflect.Interface:
							setValueToStructField(utils.ArrParseAny(bindedValues))
							continue
						}

					default:
						continue
					}
				}
			}
		}
	}

	return reflect.Indirect(newStructuredData).Interface()
}
