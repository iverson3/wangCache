package wangcache

import (
	"fmt"
	"log"
	"sync"
)

// Group 是 GeeCache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程

//                           是
//接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
//                |  否                         是
//                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
//                            |  否
//                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶


//设计了一个回调函数(callback)，在缓存不存在时，调用这个函数，得到源数据

type Getter interface {
	Get(key string) ([]byte, error)  // 通过指定的key获取数据
}

// 自定义一个函数类型
type GetterFunc func(key string) ([]byte, error)

// 为自定义的函数类型实现Getter接口
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}


type Group struct {
	name      string  // 缓存的命名空间 (缓存的分类)
	getter    Getter  // 即缓存未命中时获取源数据的回调(callback)
	mainCache cache
}

var (
	mu sync.RWMutex
	groups = make(map[string]*Group)  // 保存着所有的缓存实例，key是对应缓存的命名空间name
)

// 新建一个缓存实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	val, ok := g.mainCache.get(key)
	if ok {
		log.Printf("key [%s] cache hit\n", key)
		return val, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	//分布式场景下会调用 getFromPeer 从其他节点获取
	return g.getLocally(key)
}

//getLocally 调用用户回调函数 g.getter.Get()获取源数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 将源数据添加到缓存中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}






