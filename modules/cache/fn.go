package cache

import (
	"time"
)

func setExpiry(ex time.Duration) time.Duration {
	var exAt time.Duration = -1
	if ex > 0 {
		exAt = time.Duration(time.Now().Add(ex * time.Millisecond).UnixNano())
	} else if ex == 0 {
		exAt = 0
	}
	return exAt
}

func isExpired(exAt time.Duration) bool {
	if exAt == -1 {
		return false
	}

	if exAt == 0 {
		return true
	}

	return time.Now().UnixNano() > int64(exAt)
}
