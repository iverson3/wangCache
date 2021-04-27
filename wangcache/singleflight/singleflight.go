package singleflight

import "sync"

//call 代表正在进行中，或已经结束的请求
type call struct {
	wg  sync.WaitGroup  // 使用 sync.WaitGroup锁避免重入
	val interface{}
	err error
}

// 管理每个key各自的请求(call)
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	// 如果指定key已经存在对应的call，那么就通过Wait等待call的结束 (call结束后就有了数据或错误信息)
	// 由此就避免了针对同一个key的重复且无效的请求
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	// 调用函数，函数本身就是获取数据的实现 (通常都是访问其他节点获取缓存值)
	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}





























