package main

import (
	"7go/wangCache/wangcache"
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

func main() {
	wangcache.NewGroup("scores", 2<<10, wangcache.GetterFunc(
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

	addr := "localhost:9999"
	peers := wangcache.NewHTTPPool(addr)

	log.Println("wangcache is running at ", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
