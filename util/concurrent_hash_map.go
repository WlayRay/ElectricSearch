package util

import (
	"sync"

	"github.com/dgryski/go-farm"
	"golang.org/x/exp/maps"
)

type ConcurrentHashMap struct {
	childMaps []map[string]any
	locks     []sync.RWMutex
	segment   int
	seed      int
}

func NewConcurrentHashMap(segment int, capacity int) *ConcurrentHashMap {
	concurrentHashMap := &ConcurrentHashMap{
		childMaps: make([]map[string]any, segment),
		locks:     make([]sync.RWMutex, segment),
		segment:   segment,
		seed:      0,
	}
	for i := 0; i < segment; i++ {
		concurrentHashMap.childMaps[i] = make(map[string]any, capacity/segment)
	}
	return concurrentHashMap
}

func (c *ConcurrentHashMap) getSegIndex(key string) int {
	return int(farm.Hash32WithSeed([]byte(key), uint32(c.seed)) % uint32(c.segment))
}

func (c *ConcurrentHashMap) Get(key string) (any, bool) {
	segment := c.getSegIndex(key)
	c.locks[segment].RLock()
	defer c.locks[segment].RUnlock()
	return c.childMaps[segment][key], c.childMaps[segment][key] != nil
}

func (c *ConcurrentHashMap) Set(key string, value any) {
	segment := c.getSegIndex(key)
	c.locks[segment].Lock()
	defer c.locks[segment].Unlock()
	c.childMaps[segment][key] = value
}

// 迭代器模式
type MapEntry struct {
	Key   string
	Value any
}

type MapIterator interface {
	Next() *MapEntry
}

type ConcurrentHashMapIterator struct {
	cm       *ConcurrentHashMap
	keys     [][]string
	rowIndex int
	colIndex int
}

func (c *ConcurrentHashMap) NewIterator() *ConcurrentHashMapIterator {
	keys := make([][]string, len(c.childMaps))
	for _, v := range c.childMaps {
		rowKeys := maps.Keys(v) // 使用golang.org/x/exp/maps包的Keys方法获取一个map的所有key
		// var rowKeys []string
		// for i := range v {
		// rowKeys = append(rowKeys, i)
		// }
		keys = append(keys, rowKeys)
	}

	return &ConcurrentHashMapIterator{
		cm:       c,
		keys:     keys,
		rowIndex: 0,
		colIndex: 0,
	}
}

func (iter *ConcurrentHashMapIterator) Next() *MapEntry {
	if iter.rowIndex >= len(iter.keys) {
		return nil
	}
	if len(iter.keys[iter.rowIndex]) == 0 {
		iter.rowIndex++
		return iter.Next()
	}

	key := iter.keys[iter.rowIndex][iter.colIndex]
	value, _ := iter.cm.Get(key)

	if iter.colIndex >= len(iter.keys[iter.rowIndex])-1 {
		iter.rowIndex++
		iter.colIndex = 0
	} else {
		iter.colIndex++
	}
	return &MapEntry{
		Key:   key,
		Value: value,
	}
}
