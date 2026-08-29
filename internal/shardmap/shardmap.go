package shardmap

import (
	"sync"

	"example.com/nexus/internal/hash"
	"golang.org/x/sys/cpu"
)

type Shard struct {
	data map[string]int64
	mu   sync.RWMutex
	_    cpu.CacheLinePad
}

type ShardedMap struct {
	shards []*Shard
}

func New(numShards int) *ShardedMap {
	sm := &ShardedMap{shards: make([]*Shard, numShards)}
	for i := range sm.shards {
		sm.shards[i] = &Shard{data: make(map[string]int64)}
	}
	return sm
}

func (sm *ShardedMap) getShard(key string) *Shard {
	return sm.shards[hash.FNVNew32aHash(key)%len(sm.shards)]
}

func (sm *ShardedMap) Set(key string, val int64) {
	sh := sm.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.data[key] = val
}

func (sm *ShardedMap) Get(key string) (int64, bool) {
	sh := sm.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	val, ok := sh.data[key]
	return val, ok
}
