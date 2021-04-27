package wangcache

import (
	"7go/wangCache/wangcache/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	url2 "net/url"
	"strings"
	"sync"
)

//提供被其他节点访问的能力(基于http)

//分布式缓存需要实现节点间通信，建立基于 HTTP 的通信机制是比较常见和简单的做法。
//如果一个节点启动了 HTTP 服务，那么这个节点就可以被其他节点访问

// 比如http://example.com/_wangcache/ 开头的请求，就用于节点间的访问。
// 因为一个主机上还可能承载其他的服务，加一段 Path 是一个好习惯。比如，大部分网站的 API接口，一般以 /api 作为前缀
const (
	defaultBasePath = "/_wangcache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	self        string   // 用来记录自己的地址，包括主机名/IP和端口
	basePath    string   // 作为节点间通讯地址的前缀，默认是 /_wangcache/
	mu          sync.Mutex
	peers       *consistenthash.Map  // 一致性哈希算法的Map，用来根据具体的 key选择节点
	httpGetters map[string]*httpGetter   // 映射远程节点与对应的httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	// 添加节点
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		// 为每一个节点创建一个HTTP客户端 httpGetter
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 包装了一致性哈希算法的 Get()方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端
// 实现PeerPicker接口
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 使用一致性hash算法根据key获取节点
	peer := p.peers.Get(key)
	// 如果返回的节点是自己(当前节点)，说明这个key就是由当前节点负责处理(包括缓存值的获取和存储)
	// 所以不需要返回对应的httpGetter，因为key对应的缓存需要在当前节点中获取，不再需要请求其他节点
	if peer != "" && peer != p.self {
		log.Printf("[Server %s] select node is (%s)", p.self, peer)
		return p.httpGetters[peer], true
	}
	// 该判断和输出主要是为了调试，无具体逻辑意义
	if peer == p.self {
		log.Printf("select node is self")
	}
	// 条件成立则说明当前哈希环上没有任何节点 (即所有cacheServer节点都挂掉了，但目前这个系统没有考虑缓存服务器宕机的情况以及相应的处理)
	if peer == "" {
		log.Printf("select node is null, there is no node to be selected.")
	}

	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)


func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 实现http的Handler接口，处理所有的http请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// url地址约定格式是 /<basepath>/<groupname>/<key>
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log("%s %s", r.Method, r.URL.Path)

	// 切割出url后面的部分，约定格式是 <groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: " + groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	n, err := w.Write(view.ByteSlice())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n != view.Len() {
		http.Error(w, "the length of send data is not equal than the length of view", http.StatusInternalServerError)
		return
	}

	log.Printf("node [%s]: get cache successfully.", p.self)
}





//**********************************
// 实现Http客户端
//**********************************

type httpGetter struct {
	baseURL string  // 表示将要访问的远程节点的地址
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 拼装请求的url
	url := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url2.QueryEscape(group),
		url2.QueryEscape(key))

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed, error: %v", err)
	}

	return data, nil
}

var _ PeerGetter = (*httpGetter)(nil)












