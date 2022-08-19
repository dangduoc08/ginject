package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dangduoc08/go-go/helper"
)

const PARAM_SYMBOL = "P"

type Param[T interface{}] struct{ KeyValue (map[string]T) }

func NewParam(s string) (string, *Param[interface{}]) {
	matchParamPatternReg := regexp.MustCompile(fmt.Sprintf(`\%v(.*?)\%v`, helper.COLON, helper.SLASH))
	p := &Param[interface{}]{
		KeyValue: make(map[string]interface{}),
	}

	for _, v := range matchParamPatternReg.FindAll([]byte(s), -1) {
		keyWithoutLastSlash := helper.RemoveLastSlash(string(v))
		s = strings.Replace(s, keyWithoutLastSlash, PARAM_SYMBOL, 1)
		paramKey := helper.RemoveFirstColon(keyWithoutLastSlash)
		p.Set(paramKey, nil)
	}

	return s, p
}

func (p *Param[T]) Get(k string) T {
	return p.KeyValue[k]
}

func (p *Param[T]) Set(k string, v T) {
	p.KeyValue[k] = v
}

func (p *Param[T]) Delete(k string) {
	delete(p.KeyValue, k)
}
