package wangcache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

//提供被其他节点访问的能力(基于http)

//分布式缓存需要实现节点间通信，建立基于 HTTP 的通信机制是比较常见和简单的做法。
//如果一个节点启动了 HTTP 服务，那么这个节点就可以被其他节点访问

// 比如http://example.com/_wangcache/ 开头的请求，就用于节点间的访问。
// 因为一个主机上还可能承载其他的服务，加一段 Path 是一个好习惯。比如，大部分网站的 API接口，一般以 /api 作为前缀
const defaultBasePath = "/_wangcache/"

type HTTPPool struct {
	self     string   // 用来记录自己的地址，包括主机名/IP和端口
	basePath string   // 作为节点间通讯地址的前缀，默认是 /_wangcache/
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

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

	log.Println("get cache successfully.")
}
























