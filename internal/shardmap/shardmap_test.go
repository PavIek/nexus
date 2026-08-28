package shardmap

import (
	"strconv"
	"testing"
)

// func BenchmarkShardedMapSet(b *testing.B) {
// 	sm := New(64)
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			sm.Set("key", 1)
// 		}
// 	})
// }

func BenchmarkShardedMapSet(b *testing.B) {
	sm := New(64)

	keys := make([]string, 1024)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sm.Set(keys[i&1023], int64(i))
			i++
		}
	})
}
