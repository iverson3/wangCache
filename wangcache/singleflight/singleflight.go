package singleflight

import "sync"

// 在一瞬间有大量请求get(key)，而且key未被缓存或者未被缓存在当前节点 如果不用singleflight，那么这些请求都会发送远端节点或者从本地数据库读取，会造成远端节点或本地数据库压力猛增。
// 使用singleflight，第一个get(key)请求到来时，singleflight会记录当前key正在被处理，后续的请求只需要等待第一个请求处理完成，取返回值即可。
// 所以使用singleflight 能有效的避免缓存击穿或缓存穿透

// singleflight 这个机制在缓存领域使用是非常广泛的，geecache 里实现的是一个比较简单的版本。感兴趣可以再看一看标准库里面的实现 x/sync/singleflight
// golang非常著名的开源项目 groupcache 里面也有关于singleflight更复杂的实现

/*
Q: 为什么要使用waitgroup呢，实际上c.wg中同时最多只会有一个任务，使用waitgroup是不是太浪费了
A: 如果用channel，接受方和发送方需要一一对应， 虽然waitgroup中 Add(1) 和 Done() 也是一一对应的，但是可以有多个请求同时调用 Wait()，同时等待该任务结束，一般锁和channel是做不到这一点的。
*/

//call 代表正在进行中或已经结束的请求
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





























