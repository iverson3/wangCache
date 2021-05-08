package wangcache

// 定义两个接口

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)  // 根据传入的 key选择相应节点 PeerGetter
}

type PeerGetter interface {
	Get(group string, key string) ([]byte, error)   // 从对应 group查找缓存值
}
