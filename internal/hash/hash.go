package hash

import "hash/fnv"

func FNVNew32aHash(key string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32())
}
