package shardmap

import (
	"strconv"
	"sync"
	"testing"

	"example.com/nexus/internal/hash"
	"golang.org/x/sys/cpu"
)

type benchShardNoPad struct {
	data map[string]int64
	mu   sync.RWMutex
}

type benchMapNoPad struct {
	shards []*benchShardNoPad
}

func newBenchMapNoPad(numShards int) *benchMapNoPad {
	sm := &benchMapNoPad{shards: make([]*benchShardNoPad, numShards)}
	for i := range sm.shards {
		sm.shards[i] = &benchShardNoPad{data: make(map[string]int64)}
	}
	return sm
}

func (sm *benchMapNoPad) Set(key string, val int64) {
	sh := sm.shards[hash.FNVNew32aHash(key)%len(sm.shards)]
	sh.mu.Lock()
	sh.data[key] = val
	sh.mu.Unlock()
}

type benchShardPadded struct {
	data map[string]int64
	mu   sync.RWMutex
	_    cpu.CacheLinePad
}

type benchMapPadded struct {
	shards []*benchShardPadded
}

func newBenchMapPadded(numShards int) *benchMapPadded {
	sm := &benchMapPadded{shards: make([]*benchShardPadded, numShards)}
	for i := range sm.shards {
		sm.shards[i] = &benchShardPadded{data: make(map[string]int64)}
	}
	return sm
}

func (sm *benchMapPadded) Set(key string, val int64) {
	sh := sm.shards[hash.FNVNew32aHash(key)%len(sm.shards)]
	sh.mu.Lock()
	sh.data[key] = val
	sh.mu.Unlock()
}

func benchmarkSetNoPad(b *testing.B) {
	sm := newBenchMapNoPad(64)
	keys := benchKeys()

	for i, key := range keys {
		sm.Set(key, int64(i))
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

func benchmarkSetPadded(b *testing.B) {
	sm := newBenchMapPadded(64)
	keys := benchKeys()

	for i, key := range keys {
		sm.Set(key, int64(i))
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

func benchKeys() []string {
	keys := make([]string, 1024)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	return keys
}

func BenchmarkShardedMapSetNoPad(b *testing.B) {
	benchmarkSetNoPad(b)
}

func BenchmarkShardedMapSetPadded(b *testing.B) {
	benchmarkSetPadded(b)
}
