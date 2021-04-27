package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
// 使用 lru 缓存淘汰策略
type Cache struct {
	maxBytes int64   // 缓存允许使用的最大字节数
	nbytes int64     // 当前已使用的字节数
	ll *list.List    // 直接使用Go语言标准库实现的双向链表
	cache map[string]*list.Element  // 键是字符串，值是双向链表中对应节点的指针
	// optional and executed when an entry is purged.
	// 某条记录被移除时的回调函数，可以为 nil
	OnEvicted func(key string, value Value)
}

// 自定义双向链表中节点的数据类型 (因为双向链表中节点类型是interface{})
type entry struct {
	key string
	value Value  // 为了通用性，我们允许值是实现了 Value 接口的任意类型; 即只要实现了Value接口就可以作为缓存值
}

// Value use Len() to count how many bytes it takes
type Value interface {
	Len() int  // 用于返回值所占用的内存大小
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 获取缓存
func (c *Cache) Get(key string) (value Value, ok bool) {
	ele, ok := c.cache[key]
	if ok {
		// 将当前元素移动到链表最前面 (删除元素的时候是从链表最后面开始移除的)
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, ok
	}
	return
}

// 移除缓存，即淘汰缓存，移除最近最少访问的节点 (链表最后面的元素)
// 缓存淘汰策略使用的是 LRU算法  (常用的三种缓存淘汰(失效)算法：FIFO，LFU 和 LRU)
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增/修改缓存
func (c *Cache) Add(key string, value Value) {
	ele, ok := c.cache[key]
	// 存在则更新值
	if ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 重新计算所占字节数
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		// 更新值
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	// 超过了缓存允许使用的最大值，则开始淘汰缓存
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}











