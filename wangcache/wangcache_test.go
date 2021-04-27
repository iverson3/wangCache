package wangcache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

// 模拟数据库
var db = map[string]string {
	"Tom": "630",
	"Jack": "589",
	"Sam": "567",
}

func TestGetter(t *testing.T)  {
	key := "key1"
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	
	expect := []byte(key)

	if val, _ := f.Get(key); !reflect.DeepEqual(val, expect) {
		t.Fatal("callback failed")
	}
}

func TestGet(t *testing.T)  {
	loadCounts := make(map[string]int, len(db))

	group := NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDb] search key:", key)
		if v, ok := db[key]; ok {
			if _, ok := loadCounts[key]; !ok {
				loadCounts[key] = 0
			}
			loadCounts[key]++
			return []byte(v), nil
		}
		return nil, fmt.Errorf("key [%s] not exist", key)
	}))

	for k, v := range db {
		//在缓存为空的情况下，能够通过回调函数获取到源数据
		view, err := group.Get(k)
		if err != nil || view.String() != v {
			t.Fatalf("failed to get value of Tom")
		}

		//在缓存已经存在的情况下，是否直接从缓存中获取
		_, err = group.Get(k)
		if err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	view, err := group.Get("unknown")
	if err == nil {
		t.Fatalf("the value of unknown should be empty, but got %s", view)
	}
}

func TestGetGroup(t *testing.T) {
	groupNmae := "scores"

	NewGroup(groupNmae, 2<<10, GetterFunc(func(key string) (bytes []byte, err error) {
		return
	}))

	group := GetGroup(groupNmae)
	if group == nil || group.name != groupNmae {
		t.Fatalf("group %s not exist", groupNmae)
	}

	group2 := GetGroup(groupNmae + "xxx")
	if group2 != nil {
		t.Fatalf("expect nil, but got %s", group2.name)
	}
}
