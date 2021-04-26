package wangcache

import (
	"7go/wangCache/wangcache/lru"
	"sync"
)

type cache struct {
	mu         sync.Mutex
	lru       *lru.Cache
	cacheBytes int64     // 最大使用内存字节数
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}

	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	val, ok := c.lru.Get(key)
	if ok {
		return val.(ByteView), ok
	}
	return
}