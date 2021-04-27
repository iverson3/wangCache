package wangcache

import (
	"7go/wangCache/wangcache/singleflight"
	"fmt"
	"log"
	"sync"
)

//负责与外部交互，控制缓存存储和获取的主流程

//                           是
// 接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
//                 |  否                         是
//                 |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
//                             |  否
//                             |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶


// 设计了一个回调函数(callback)，在缓存不存在时，调用这个函数，得到源数据
type Getter interface {
	Get(key string) ([]byte, error)  // 通过指定的key获取数据
}

// 自定义一个函数类型
type GetterFunc func(key string) ([]byte, error)

// 为自定义的函数类型实现Getter接口
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}


// Group是 wangCache最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程
type Group struct {
	name      string  // 缓存的命名空间 (缓存的分类)
	getter    Getter  // 缓存未命中时获取源数据的回调(callback)
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that each key is only fetched once
	loader *singleflight.Group
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
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// 获取指定name的Group实例
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

// 从当前group中获取缓存数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	val, ok := g.mainCache.get(key)
	if ok {
		log.Printf("[Server %s] key [%s] cache hit\n", g.peers.(*HTTPPool).self, key)
		return val, nil
	}
	log.Printf("[Server %s] local cache is missed, now go to load data for key[%s]", g.peers.(*HTTPPool).self, key)
	return g.load(key)
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) load(key string) (value ByteView, err error) {
	//将原来的 load相关逻辑，使用 g.loader.Do包裹起来，这样确保了并发场景下针对相同的 key，load过程只会调用一次
	// 不管是远程调用获取还是本地获取，并发场景下，每个key都只会获取缓存值一次
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// 根据key选择节点
			if peer, ok := g.peers.PickPeer(key); ok {
				// 分布式场景下会调用 getFromPeer从其他远程节点获取缓存值
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Printf("[wangCache] failed to get key[%s] from peer, error: %v", key, err)
			}
		}
		// 如果远程节点取不到缓存值或者目标节点就是本机节点，则直接从本地获取 (一般是从数据库中查询获取数据)
		return g.getLocally(key)
	})

	if err == nil {
		// viewi 是interface{}类型的，所以需要转类型
		return viewi.(ByteView), nil
	}
	return
}

// 访问远程节点，获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	log.Printf("=====fetch data from remote node is starting====")
	data, err := peer.Get(g.name, key)
	log.Printf("=====fetch data from remote node is end====")
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: data}, nil
}

// getLocally 调用用户回调函数 g.getter.Get()获取源数据
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






