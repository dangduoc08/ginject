package ds

import "strings"

func matchWildcard(str, route string) bool {
	prefix, rest, found := strings.Cut(route, "*")
	if !found {
		return str == route
	}

	if prefix != "" {
		if len(str) < len(prefix) || str[:len(prefix)] != prefix {
			return false
		}
		str = str[len(prefix):]
	}

	for {
		mid, next, found := strings.Cut(rest, "*")
		if !found {
			if mid == "" {
				return true
			}
			return len(str) >= len(mid) && str[len(str)-len(mid):] == mid
		}
		rest = next

		if mid == "" {
			continue
		}

		idx := strings.Index(str, mid)
		if idx < 0 {
			return false
		}
		str = str[idx+len(mid):]
	}
}
