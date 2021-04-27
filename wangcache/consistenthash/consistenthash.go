package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//实现一致性哈希算法

//一般来说，哈希函数考虑两个点：一个是碰撞率，一个是性能。比如 CRC、MD5、SHA1。
//对于缓存来说，hash 之后再根据节点数量取模，因此 hash 函数的碰撞率影响并不大，而是模的大小，也就是节点的数量比较关键，这也是引入虚拟节点的原因，但是缓存对性能比较敏感。
//而对于需要完整性校验的场合，碰撞率比较关键，而性能就比较次要了。一般使用 256位的 SHA1 算法，MD5 已经不再推荐了。CRC 即循环冗余校验，编码简单，性能高，但安全性就很差了。作为缓存的 hash 算法还是很合适的。

// 定义了函数类型 Hash，采取依赖注入的方式，允许替换成自定义的 Hash函数
type Hash func(data []byte) uint32

type Map struct {
	hash     Hash   // Hash函数
	replicas int    // 虚拟节点倍数 (即一个真实节点对应哈希环上几个虚拟节点)
	keys     []int  // 哈希环, sorted
	hashMap  map[int]string  // 虚拟节点与真实节点的映射表 (键是虚拟节点的哈希值，值是真实节点的名称)
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		// 默认使用 crc32.ChecksumIEEE算法
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点/机器
// 参数为 真实节点的名称
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 为每个真实节点生成对应数量的虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 根据Hash算法计算出虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将虚拟节点的hash值添加到哈希环上
			m.keys = append(m.keys, hash)
			// 设置真实节点与虚拟节点的映射关系
			m.hashMap[hash] = key
		}
	}

	// 对哈希环进行排序，保证哈希环是有序的
	sort.Ints(m.keys)
}

// 移除节点/机器
func (m *Map) Remove(key string) {
	for i := 0; i < m.replicas; i++ {
		// 根据Hash算法计算出虚拟节点的哈希值
		hash := int(m.hash([]byte(strconv.Itoa(i) + key)))

		// 从哈希环上寻找hash值对应的下标
		idx := sort.SearchInts(m.keys, hash)
		// 从哈希环上移除idx下标对应的哈希值
		m.keys = append(m.keys[:idx], m.keys[idx+1:]...)

		// 从m.hashMap中移除虚拟节点与真实节点的映射关系
		delete(m.hashMap, hash)
	}
}

// 选择节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	// 计算出key的哈希值
	hash := int(m.hash([]byte(key)))
	// 顺时针找到第一个匹配的虚拟节点的下标 idx
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 获取下标对应的虚拟节点的哈希值，m.keys是哈希环，属于首尾相连的环状结构，所以使用取余数的方式来避免下标越界
	vHash := m.keys[idx % len(m.keys)]
	// 从hashMap中获取虚拟节点对应的真实节点
	return m.hashMap[vHash]
}
