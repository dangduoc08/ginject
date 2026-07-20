package ctx

import (
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/dangduoc08/ginject/internal/slice"
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
	fieldTags := getFieldBindTags(structureType)

	for i := 0; i < structureType.NumField(); i++ {
		structField := structureType.Field(i)
		setValueToStructField := setValueToStructField(i)

		if !token.IsExported(structField.Name) {
			continue
		}

		if ft := fieldTags[i]; ft.ok {
			bindedIndex, bindedField := ft.index, ft.field
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

				switch structField.Type.Kind() {
				case reflect.Bool:
					var val bool
					if boolStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						val, _ = strconv.ParseBool(boolStr)
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Int:
					var val int
					if intStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						val, _ = strconv.Atoi(intStr)
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Int8:
					var val int8
					if intStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if i64, err := strconv.ParseInt(intStr, 10, 8); err == nil {
							val = int8(i64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Int16:
					var val int16
					if intStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if i64, err := strconv.ParseInt(intStr, 10, 16); err == nil {
							val = int16(i64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Int32:
					var val int32
					if intStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if i64, err := strconv.ParseInt(intStr, 10, 32); err == nil {
							val = int32(i64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Int64:
					var val int64
					if intStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						val, _ = strconv.ParseInt(intStr, 10, 64)
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Uint:
					var val uint
					if uintStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if u64, err := strconv.ParseUint(uintStr, 10, 0); err == nil {
							val = uint(u64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Uint8:
					var val uint8
					if uintStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if u64, err := strconv.ParseUint(uintStr, 10, 8); err == nil {
							val = uint8(u64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Uint16:
					var val uint16
					if uintStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if u64, err := strconv.ParseUint(uintStr, 10, 16); err == nil {
							val = uint16(u64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Uint32:
					var val uint32
					if uintStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if u64, err := strconv.ParseUint(uintStr, 10, 32); err == nil {
							val = uint32(u64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Uint64:
					var val uint64
					if uintStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						val, _ = strconv.ParseUint(uintStr, 10, 64)
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Float32:
					var val float32
					if fStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if f64, err := strconv.ParseFloat(fStr, 32); err == nil {
							val = float32(f64)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Float64:
					var val float64
					if fStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						val, _ = strconv.ParseFloat(fStr, 64)
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Complex64:
					var val complex64
					if cStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						if c128, err := strconv.ParseComplex(strings.ReplaceAll(cStr, " ", ""), 64); err == nil {
							val = complex64(c128)
						}
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Complex128:
					var val complex128
					if cStr, ok := slice.Get(bindedValues, bindedIndex); ok {
						val, _ = strconv.ParseComplex(strings.ReplaceAll(cStr, " ", ""), 128)
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.String:
					var val string
					if str, ok := slice.Get(bindedValues, bindedIndex); ok {
						val = str
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Interface:
					var val string
					if strVal, ok := slice.Get(bindedValues, bindedIndex); ok {
						val = strVal
					}
					fl.val = val
					*fls = append(*fls, fl)
					setValueToStructField(val)
					continue

				case reflect.Slice:
					switch structField.Type.Elem().Kind() {
					case reflect.Bool:
						val := slice.StrParseBool(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Int:
						val := slice.StrParseInt(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Int8:
						val := slice.StrParseInt8(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Int16:
						val := slice.StrParseInt16(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Int32:
						val := slice.StrParseInt32(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Int64:
						val := slice.StrParseInt64(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Uint:
						val := slice.StrParseUint(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Uint8:
						val := slice.StrParseUint8(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Uint16:
						val := slice.StrParseUint16(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Uint32:
						val := slice.StrParseUint32(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Uint64:
						val := slice.StrParseUint64(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Float32:
						val := slice.StrParseFloat32(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Float64:
						val := slice.StrParseFloat64(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Complex64:
						val := slice.StrParseComplex64(bindedValues)
						fl.val = val
						*fls = append(*fls, fl)
						setValueToStructField(val)
						continue
					case reflect.Complex128:
						val := slice.StrParseComplex128(bindedValues)
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
						val := slice.StrParseAny(bindedValues)
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

	return reflect.Indirect(newStructuredData).Interface(), *fls
}
