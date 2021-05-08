package main

import (
	"7go/wangCache/wangcache"
	"flag"
	"fmt"
	"log"
	"net/http"

)

// 模拟数据库
var db = map[string]string{
	"Tom": "630",
	"Jack": "589",
	"Sam": "567",
}

func createGroup() *wangcache.Group {
	return wangcache.NewGroup("scores", 2<<10, wangcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key: ", key)

			// 模拟从数据库中获取数据
			val, ok := db[key]
			if ok {
				log.Println("get data from db success")
				return []byte(val), nil
			}

			err := fmt.Errorf("key:[%s] not exist", key)
			log.Printf("get data from db failed, %s\n", err.Error())
			return nil, err
		}))
}

// 启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 gee 中，启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知
func startCacheServer(addr string, addrs []string, group *wangcache.Group) {
	peers := wangcache.NewHTTPPool(addr)
	peers.Set(addrs...)
	group.RegisterPeers(peers)
	log.Println("wangCache is running at ", addr)

	// addr 格式是 http://localhost:9999
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动一个 API服务（端口 9999），与用户进行交互，用户感知
func startAPIServer(apiAddr string, group *wangcache.Group) {
	http.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("-----api request deal starting-----")
		key := r.URL.Query().Get("key")
		view, err := group.Get(key)

		log.Printf("-----api request deal is end-----")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	}))

	log.Println("fontend server is running at ", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool

	flag.IntVar(&port, "port", 8001, "wangCache server port")
	flag.BoolVar(&api, "api", false, "start a api server?")
	flag.Parse()

	// 定义了apiServer的地址和三个cacheServer的地址
	// 这里属于硬编码，可以考虑使用配置文件的方式来动态修改
	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	group := createGroup()
	if api {
		go startAPIServer(apiAddr, group)
	}

	startCacheServer(addrMap[port], addrs, group)
}
