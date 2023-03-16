package config

import (
	"strconv"
	"strings"
)

var (
	newline      byte = 10 // '/n'
	doubleQuotes byte = 34 // '"'
	hash         byte = 35 // '#'
	equal        byte = 61 // '='
)

type DotENV struct {
	data   []byte
	envMap map[string]any
}

func (e *DotENV) Unmarshal() map[string]any {
	k := []byte{}
	v := []byte{}
	isKey := true
	isValue := false
	isComment := false
	isQuotation := false

	for i, r := range e.data {
		if !isQuotation && r == newline ||
			isQuotation && r == doubleQuotes ||
			i == len(e.data)-1 {
			envKey := strings.TrimSpace(string(k))
			if isValidKey(envKey) {
				envValue := strings.TrimSpace(string(v))
				var err error
				e.envMap[envKey], err = strconv.Unquote(`"` + envValue + `"`)
				if err != nil {
					e.envMap[envKey] = envValue
				}
			}

			// reset flags
			k = []byte{}
			v = []byte{}
			isKey = true
			isValue = false
			isComment = false
			isQuotation = false
			continue
		}

		if isComment {
			continue
		}

		if !isQuotation && r == hash {
			isKey = false
			isValue = false
			isComment = true
			isQuotation = false
			continue
		}

		if !isQuotation && r == equal {
			isKey = false
			isValue = true
			isComment = false
			isQuotation = false
			continue
		}

		if r == doubleQuotes {
			isKey = false
			isValue = true
			isComment = false
			isQuotation = true
			continue
		}

		if isKey {
			k = append(k, r)
			continue
		}

		if isValue {
			v = append(v, r)
			continue
		}
	}

	return e.envMap
}
