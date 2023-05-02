package common

import (
	"reflect"
	"runtime"
	"strings"
)

func getFnName(handler any) string {
	strs := strings.Split(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(), ".")
	fnName := strs[len(strs)-1]
	return fnName[:len(fnName)-3]
}
