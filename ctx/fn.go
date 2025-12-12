package ctx

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/dangduoc08/ginject/utils"
)

func toJSONBuffer(args ...any) ([]byte, error) {
	data := args[0]
	switch args[0].(type) {
	case string:
		jsonStr := fmt.Sprintf(data.(string), args[1:]...)
		data = json.RawMessage(jsonStr)
	}

	return json.Marshal(&data)
}

func toJSONP(jsonStr, callback string) string {
	return fmt.Sprintf("/**/ typeof %v === 'function' && %v(%v);", callback, callback, jsonStr)
}

func GetTagParams(v string) []string {
	return utils.ArrFilter(utils.ArrMap(
		strings.Split(v, ","), func(el string, i int) string {
			return strings.TrimSpace(el)
		}), func(el string, i int) bool {
		return el != ""
	})
}

func GetTagParamIndex(v string) (int, string) {
	splittedBindParams := strings.Split(v, ".")
	bindedField := v
	bindedIndex := 0

	if len(splittedBindParams) > 1 {

		// bind:"int_5.3"
		bindedField = strings.TrimSpace(splittedBindParams[0])
		parsedInt, err := strconv.Atoi(strings.TrimSpace(splittedBindParams[1]))

		if err == nil && parsedInt > -1 {
			bindedIndex = parsedInt
		}

		return bindedIndex, bindedField
	}

	return bindedIndex, bindedField
}

func setValueToStructField(s reflect.Value) func(i int) func(v any) {
	return func(i int) func(v any) {
		return func(v any) {
			s.Elem().Field(i).Set(reflect.ValueOf(v))
		}
	}
}

func fromStrucValueToStructPointerValue(val any) any {
	ptrStruct := reflect.New(reflect.TypeOf(val))
	ptrStruct.Elem().Set(reflect.ValueOf(val))
	return ptrStruct.Interface()
}

func bindArray(arr []any, fls *[]FieldLevel, typ reflect.Type, parentNS string, parentTag string) any {
	switch typ.Elem().Kind() {

	case reflect.Bool:
		var boolArr []bool
		for _, el := range arr {
			if boolean, ok := el.(bool); ok {
				boolArr = append(boolArr, boolean)
				continue
			}

			boolArr = []bool{}
			break
		}
		return boolArr

	case reflect.Int:
		var intArr []int
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				intArr = append(intArr, int(f64))
				continue
			}

			intArr = []int{}
			break
		}

		return intArr

	case reflect.Int8:
		var int8Arr []int8
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				int8Arr = append(int8Arr, int8(f64))
				continue
			}

			int8Arr = []int8{}
			break
		}
		return int8Arr

	case reflect.Int16:
		var in16Arr []int16
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				in16Arr = append(in16Arr, int16(f64))
				continue
			}

			in16Arr = []int16{}
			break
		}
		return in16Arr

	case reflect.Int32:
		var int32Arr []int32
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				int32Arr = append(int32Arr, int32(f64))
				continue
			}

			int32Arr = []int32{}
			break
		}
		return int32Arr

	case reflect.Int64:
		var int64Arr []int64
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				int64Arr = append(int64Arr, int64(f64))
				continue
			}

			int64Arr = []int64{}
			break
		}
		return int64Arr

	case reflect.Uint:
		var uintArr []uint
		for _, el := range arr {
			if f64, ok := el.(float64); ok && f64 >= 0 {
				uintArr = append(uintArr, uint(f64))
				continue
			}

			uintArr = []uint{}
			break
		}
		return uintArr

	case reflect.Uint8:
		var uint8Arr []uint8
		for _, el := range arr {
			if f64, ok := el.(float64); ok && f64 >= 0 {
				uint8Arr = append(uint8Arr, uint8(f64))
				continue
			}

			uint8Arr = []uint8{}
			break
		}
		return uint8Arr

	case reflect.Uint16:
		var uint16Arr []uint16
		for _, el := range arr {
			if f64, ok := el.(float64); ok && f64 >= 0 {
				uint16Arr = append(uint16Arr, uint16(f64))
				continue
			}

			uint16Arr = []uint16{}
			break
		}
		return uint16Arr

	case reflect.Uint32:
		var uint32Arr []uint32
		for _, el := range arr {
			if f64, ok := el.(float64); ok && f64 >= 0 {
				uint32Arr = append(uint32Arr, uint32(f64))
				continue
			}

			uint32Arr = []uint32{}
			break
		}
		return uint32Arr

	case reflect.Uint64:
		var uint64Arr []uint64
		for _, el := range arr {
			if f64, ok := el.(float64); ok && f64 >= 0 {
				uint64Arr = append(uint64Arr, uint64(f64))
				continue
			}

			uint64Arr = []uint64{}
			break
		}
		return uint64Arr

	case reflect.Float32:
		var float32Arr []float32
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				float32Arr = append(float32Arr, float32(f64))
				continue
			}

			float32Arr = []float32{}
			break
		}
		return float32Arr

	case reflect.Float64:
		var float64Arr []float64
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				float64Arr = append(float64Arr, f64)
				continue
			}

			float64Arr = []float64{}
			break
		}
		return float64Arr

	case reflect.Complex64:
		var complex64Arr []complex64
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				complex64Arr = append(complex64Arr, complex64(complex(f64, 0)))
				continue
			}

			complex64Arr = []complex64{}
			break
		}
		return complex64Arr

	case reflect.Complex128:
		var complex128Arr []complex128
		for _, el := range arr {
			if f64, ok := el.(float64); ok {
				complex128Arr = append(complex128Arr, complex(f64, 0))
				continue
			}

			complex128Arr = []complex128{}
			break
		}
		return complex128Arr

	case reflect.String:
		var stringArr []string
		for _, el := range arr {
			if str, ok := el.(string); ok {
				stringArr = append(stringArr, str)
				continue
			}

			stringArr = []string{}
			break
		}
		return stringArr

	case reflect.Interface:
		return arr

	case reflect.Slice:
		// define dynamic mutli dimension slice
		lv1ArrType := reflect.SliceOf(typ.Elem())
		lv1Arr := reflect.MakeSlice(lv1ArrType, 0, 0)

		// this slice use for hold each slice dimension
		eachElemArr := []reflect.Value{
			lv1Arr,
		}

		// detect dimension of slice
		dimensions := strings.Count(lv1Arr.String(), "[]") - 1
		flag := lv1ArrType

		// fill slice dimension into map
		for i := 0; i < dimensions; i++ {
			childElemType := flag.Elem()
			eachElemArr = append(eachElemArr, reflect.MakeSlice(childElemType, 0, 0))

			flag = childElemType
		}

		declaredElem := eachElemArr[dimensions].Type().Elem()
		declaredTyp := declaredElem.Kind()

		// recursion loop
		utils.ArrIter(arr, dimensions, func(el any, currentDimension int) {

			// switch case is for actual kind from JSON
			// we also need to check declared type

			switch reflect.TypeOf(el).Kind() {
			case reflect.Bool:
				if declaredTyp == reflect.Bool {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(el.(bool)))
				}

			case reflect.Float64:
				if declaredTyp == reflect.Int {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(int(el.(float64))))
				} else if declaredTyp == reflect.Int8 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(int8(el.(float64))))
				} else if declaredTyp == reflect.Int16 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(int16(el.(float64))))
				} else if declaredTyp == reflect.Int32 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(int32(el.(float64))))
				} else if declaredTyp == reflect.Int64 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(int64(el.(float64))))
				} else if declaredTyp == reflect.Uint && el.(float64) >= 0 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(uint(el.(float64))))
				} else if declaredTyp == reflect.Uint8 && el.(float64) >= 0 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(uint8(el.(float64))))
				} else if declaredTyp == reflect.Uint16 && el.(float64) >= 0 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(uint16(el.(float64))))
				} else if declaredTyp == reflect.Uint32 && el.(float64) >= 0 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(uint32(el.(float64))))
				} else if declaredTyp == reflect.Uint64 && el.(float64) >= 0 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(uint64(el.(float64))))
				} else if declaredTyp == reflect.Float32 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(float32(el.(float64))))
				} else if declaredTyp == reflect.Float64 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(el.(float64)))
				} else if declaredTyp == reflect.Complex64 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(complex64(complex(el.(float64), 0))))
				} else if declaredTyp == reflect.Complex128 {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(complex(el.(float64), 0)))
				}

			case reflect.String:
				if declaredTyp == reflect.String {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(el.(string)))
				}

			case reflect.Interface:
				if declaredTyp == reflect.Interface {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(el))
				}

			case reflect.Slice:
				eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], eachElemArr[dimensions-(currentDimension-1)])
				eachElemArr[dimensions-(currentDimension-1)] = reflect.MakeSlice(eachElemArr[dimensions-(currentDimension-1)].Type(), 0, 0)

			case reflect.Map:
				if declaredTyp == reflect.Map {
					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(
						bindMap(
							el.(map[string]any),
							fls,
							declaredElem,
							parentNS,
							parentTag,
						)))
				} else if declaredTyp == reflect.Struct {
					newS, _ := BindStruct(
						el.(map[string]any),
						fls,
						reflect.Indirect(reflect.New(declaredElem)).Interface(),
						parentNS,
						parentTag,
					)

					eachElemArr[dimensions-currentDimension] = reflect.Append(eachElemArr[dimensions-currentDimension], reflect.ValueOf(newS))
				}
			}
		})

		return eachElemArr[0].Interface()

	case reflect.Map:

		// define dynamic slice map
		mapType := reflect.SliceOf(typ.Elem())
		mapStruct := reflect.MakeSlice(mapType, 0, 0)

		for i, el := range arr {
			if obj, ok := el.(map[string]any); ok {
				parentNSWithIndex := fmt.Sprintf("%v.%v", parentNS, i)
				parentTagWithIndex := fmt.Sprintf("%v.%v", parentTag, i)

				eachArrayValue := bindMap(
					obj,
					fls,
					typ.Elem(),
					parentNSWithIndex,
					parentTagWithIndex,
				)

				// set value to sub-map
				mapStruct = reflect.Append(mapStruct, reflect.ValueOf(eachArrayValue))
			}
		}

		return mapStruct.Interface()
	case reflect.Struct:

		// define dynamic slice struct
		sliceType := reflect.SliceOf(typ.Elem())
		sliceStruct := reflect.MakeSlice(sliceType, 0, 0)

		for i, el := range arr {
			if obj, ok := el.(map[string]any); ok {
				parentNSWithIndex := fmt.Sprintf("%v.%v", parentNS, i)
				parentTagWithIndex := fmt.Sprintf("%v.%v", parentTag, i)

				eachArrayValue, _ := BindStruct(
					obj,
					fls,
					reflect.Indirect(reflect.New(typ.Elem())).Interface(),
					parentNSWithIndex,
					parentTagWithIndex,
				)

				// set value to sub-struct
				sliceStruct = reflect.Append(sliceStruct, reflect.ValueOf(eachArrayValue))
			}
		}
		return sliceStruct.Interface()
	}

	return nil
}

func bindMap(obj map[string]any, fls *[]FieldLevel, typ reflect.Type, parentNS string, parentTag string) any {
	switch typ.Elem().Kind() {

	case reflect.Bool:
		boolMap := map[string]bool{}
		for objKey, objValue := range obj {
			if boolean, ok := objValue.(bool); ok {
				boolMap[objKey] = boolean
				continue
			}

			boolMap = map[string]bool{}
			break
		}
		return boolMap

	case reflect.Int:
		intMap := map[string]int{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				intMap[objKey] = int(f64)
				continue
			}

			intMap = map[string]int{}
			break
		}
		return intMap

	case reflect.Int8:
		int8Map := map[string]int8{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				int8Map[objKey] = int8(f64)
				continue
			}

			int8Map = map[string]int8{}
			break
		}
		return int8Map

	case reflect.Int16:
		int16Map := map[string]int16{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				int16Map[objKey] = int16(f64)
				continue
			}

			int16Map = map[string]int16{}
			break
		}
		return int16Map

	case reflect.Int32:
		int32Map := map[string]int32{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				int32Map[objKey] = int32(f64)
				continue
			}

			int32Map = map[string]int32{}
			break
		}
		return int32Map

	case reflect.Int64:
		int64Map := map[string]int64{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				int64Map[objKey] = int64(f64)
				continue
			}

			int64Map = map[string]int64{}
			break
		}
		return int64Map

	case reflect.Uint:
		uintMap := map[string]uint{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok && f64 >= 0 {
				uintMap[objKey] = uint(f64)
				continue
			}

			uintMap = map[string]uint{}
			break
		}
		return uintMap

	case reflect.Uint8:
		uint8Map := map[string]uint8{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok && f64 >= 0 {
				uint8Map[objKey] = uint8(f64)
				continue
			}

			uint8Map = map[string]uint8{}
			break
		}
		return uint8Map

	case reflect.Uint16:
		uint16Map := map[string]uint16{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok && f64 >= 0 {
				uint16Map[objKey] = uint16(f64)
				continue
			}

			uint16Map = map[string]uint16{}
			break
		}
		return uint16Map

	case reflect.Uint32:
		uint32Map := map[string]uint32{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok && f64 >= 0 {
				uint32Map[objKey] = uint32(f64)
				continue
			}

			uint32Map = map[string]uint32{}
			break
		}
		return uint32Map

	case reflect.Uint64:
		uint64Map := map[string]uint64{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok && f64 >= 0 {
				uint64Map[objKey] = uint64(f64)
				continue
			}

			uint64Map = map[string]uint64{}
			break
		}
		return uint64Map

	case reflect.Float32:
		f32Map := map[string]float32{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				f32Map[objKey] = float32(f64)
				continue
			}

			f32Map = map[string]float32{}
			break
		}
		return f32Map

	case reflect.Float64:
		f64Map := map[string]float64{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				f64Map[objKey] = f64
				continue
			}

			f64Map = map[string]float64{}
			break
		}
		return f64Map

	case reflect.Complex64:
		complex64Map := map[string]complex64{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				complex64Map[objKey] = complex64(complex(f64, 0))
				continue
			}

			complex64Map = map[string]complex64{}
			break
		}
		return complex64Map

	case reflect.Complex128:
		complex128Map := map[string]complex128{}
		for objKey, objValue := range obj {
			if f64, ok := objValue.(float64); ok {
				complex128Map[objKey] = complex(f64, 0)
				continue
			}

			complex128Map = map[string]complex128{}
			break
		}
		return complex128Map

	case reflect.String:
		stringMap := map[string]string{}
		for objKey, objValue := range obj {
			if str, ok := objValue.(string); ok {
				stringMap[objKey] = str
				continue
			}

			stringMap = map[string]string{}
			break
		}
		return stringMap

	case reflect.Interface:
		return obj

	case reflect.Slice:

		// define dynamic map slice
		mapType := reflect.MapOf(reflect.TypeOf(""), typ.Elem())
		mapSlice := reflect.MakeMap(mapType)

		for objKey, objValue := range obj {
			if arr, ok := objValue.([]any); ok {
				parentNSWithKey := fmt.Sprintf("%v.%v", parentNS, objKey)
				parentTagWithKey := fmt.Sprintf("%v.%v", parentTag, objKey)

				eachSliceValue := bindArray(arr, fls, typ.Elem(), parentNSWithKey, parentTagWithKey)

				// set value to sub-slice
				mapSlice.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(eachSliceValue))
			}
		}
		return mapSlice.Interface()

	case reflect.Map:
		// define dynamic map slice
		mapType := reflect.MapOf(reflect.TypeOf(""), typ.Elem())
		mapMap := reflect.MakeMap(mapType)
		subElem := typ.Elem().Elem()
		declaredTyp := subElem.Kind()

		for objKey, objValue := range obj {
			switch declaredTyp {
			case reflect.Bool:
				boolMap := map[string]bool{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if boolean, ok := subObjValue.(bool); ok {
						boolMap[subObjKey] = boolean
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(boolMap))

			case reflect.Int:
				intMap := map[string]int{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						intMap[subObjKey] = int(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(intMap))

			case reflect.Int8:
				int8Map := map[string]int8{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						int8Map[subObjKey] = int8(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(int8Map))

			case reflect.Int16:
				int16Map := map[string]int16{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						int16Map[subObjKey] = int16(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(int16Map))

			case reflect.Int32:
				int32Map := map[string]int32{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						int32Map[subObjKey] = int32(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(int32Map))

			case reflect.Int64:
				int64Map := map[string]int64{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						int64Map[subObjKey] = int64(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(int64Map))

			case reflect.Uint:
				uintMap := map[string]uint{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok && f64 >= 0 {
						uintMap[subObjKey] = uint(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(uintMap))

			case reflect.Uint8:
				uint8Map := map[string]uint8{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok && f64 >= 0 {
						uint8Map[subObjKey] = uint8(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(uint8Map))

			case reflect.Uint16:
				uint16Map := map[string]uint16{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok && f64 >= 0 {
						uint16Map[subObjKey] = uint16(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(uint16Map))

			case reflect.Uint32:
				uint32Map := map[string]uint32{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok && f64 >= 0 {
						uint32Map[subObjKey] = uint32(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(uint32Map))

			case reflect.Uint64:
				uint64Map := map[string]uint64{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok && f64 >= 0 {
						uint64Map[subObjKey] = uint64(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(uint64Map))

			case reflect.Float32:
				f32Map := map[string]float32{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						f32Map[subObjKey] = float32(f64)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(f32Map))

			case reflect.Float64:
				f64Map := map[string]float64{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						f64Map[subObjKey] = f64
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(f64Map))

			case reflect.Complex64:
				complex64Map := map[string]complex64{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						complex64Map[subObjKey] = complex64(complex(f64, 0))
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(complex64Map))

			case reflect.Complex128:
				complex128Map := map[string]complex128{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if f64, ok := subObjValue.(float64); ok {
						complex128Map[subObjKey] = complex(f64, 0)
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(complex128Map))

			case reflect.String:
				stringMap := map[string]string{}
				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if str, ok := subObjValue.(string); ok {
						stringMap[subObjKey] = str
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(stringMap))

			case reflect.Interface:
				mapMap.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(objValue.(map[string]any)))

			case reflect.Slice:

				// define dynamic map slice
				mapType := reflect.MapOf(reflect.TypeOf(""), subElem)
				mapSlice := reflect.MakeMap(mapType)

				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if subObjValue, ok := subObjValue.([]any); ok {
						parentNSWithKey := fmt.Sprintf("%v.%v", parentNS, subObjKey)
						parentTagWithKey := fmt.Sprintf("%v.%v", parentTag, subObjKey)

						eachSliceValue := bindArray(subObjValue, fls, subElem, parentNSWithKey, parentTagWithKey)
						mapSlice.SetMapIndex(reflect.ValueOf(subObjKey), reflect.ValueOf(eachSliceValue))
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), mapSlice)

			case reflect.Map:

				// define dynamic map slice
				mapType := reflect.MapOf(reflect.TypeOf(""), subElem)
				mapMapMap := reflect.MakeMap(mapType)

				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if subObjValue, ok := subObjValue.(map[string]any); ok {
						parentNSWithKey := fmt.Sprintf("%v.%v", parentNS, subObjKey)
						parentTagWithKey := fmt.Sprintf("%v.%v", parentTag, subObjKey)

						eachSliceValue := bindMap(subObjValue, fls, subElem, parentNSWithKey, parentTagWithKey)
						mapMapMap.SetMapIndex(reflect.ValueOf(subObjKey), reflect.ValueOf(eachSliceValue))
					}
				}
				mapMap.SetMapIndex(reflect.ValueOf(objKey), mapMapMap)

			case reflect.Struct:

				// define dynamic map struct
				mapType := reflect.MapOf(reflect.TypeOf(""), subElem)
				mapStruct := reflect.MakeMap(mapType)

				for subObjKey, subObjValue := range objValue.(map[string]any) {
					if subObjValue, ok := subObjValue.(map[string]any); ok {
						parentNSWithKey := fmt.Sprintf("%v.%v", parentNS, subObjKey)
						parentTagWithKey := fmt.Sprintf("%v.%v", parentTag, subObjKey)

						eachMapValue, _ := BindStruct(
							subObjValue,
							fls,
							reflect.Indirect(reflect.New(subElem)).Interface(),
							parentNSWithKey,
							parentTagWithKey,
						)

						mapStruct.SetMapIndex(reflect.ValueOf(subObjKey), reflect.ValueOf(eachMapValue))
					}
				}

				mapMap.SetMapIndex(reflect.ValueOf(objKey), mapStruct)
			}
		}

		return mapMap.Interface()

	case reflect.Struct:

		// define dynamic map struct
		mapType := reflect.MapOf(reflect.TypeOf(""), typ.Elem())
		mapStruct := reflect.MakeMap(mapType)

		for objKey, objValue := range obj {
			if subObj, ok := objValue.(map[string]any); ok {
				parentNSWithKey := fmt.Sprintf("%v.%v", parentNS, objKey)
				parentTagWithKey := fmt.Sprintf("%v.%v", parentTag, objKey)

				eachMapValue, _ := BindStruct(
					subObj,
					fls,
					reflect.Indirect(reflect.New(typ.Elem())).Interface(),
					parentNSWithKey,
					parentTagWithKey,
				)

				// set value to sub-struct
				mapStruct.SetMapIndex(reflect.ValueOf(objKey), reflect.ValueOf(eachMapValue))
			}
		}
		return mapStruct.Interface()
	}

	return nil
}

func ResolveWSEventname(e string) (string, string) {
	regex, _ := regexp.Compile("^(.*?)_")
	matchedStr := regex.FindStringSubmatch(e)
	subprotocol := matchedStr[1]
	e = strings.Replace(e, matchedStr[0], "", 1)
	return subprotocol, e
}
