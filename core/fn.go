package core

import (
	"regexp"
)

func isDynamicModule(moduleType string) (bool, error) {
	return regexp.Match(`^func\(.*\*core.Module$`, []byte(moduleType))
}
