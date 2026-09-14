package memorycache

import "hash/maphash"

var hashSeed = maphash.MakeSeed()

func hashKey(key string) uint64 {
	return maphash.String(hashSeed, key)
}
