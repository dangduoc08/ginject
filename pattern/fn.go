package pattern

func matchSegments(pattern, topic []string) bool {
	pi, ti := 0, 0
	for pi < len(pattern) {
		if pattern[pi] == "*" {
			if pi == len(pattern)-1 {
				return ti < len(topic)
			}
			if ti >= len(topic) {
				return false
			}
			pi++
			ti++
			continue
		}
		if ti >= len(topic) || pattern[pi] != topic[ti] {
			return false
		}
		pi++
		ti++
	}
	return pi == len(pattern) && ti == len(topic)
}
