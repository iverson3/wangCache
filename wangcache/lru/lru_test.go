package lru

import (
	"reflect"
	"testing"
)

type String string

// 实现Value接口
func (d String) Len() int {
	return len(d)
}

// 测试添加和获取缓存
func TestGet(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Add("key1", String("123"))

	v, ok := lru.Get("key1")
	if !ok || string(v.(String)) != "123" {
		t.Fatalf("cache hit key1=123 failed!")
	}

	_, ok2 := lru.Get("key2")
	if !ok2 {
		t.Fatalf("cache miss key2 failed!")
	}
}

//测试当使用内存超过了设定值时，是否会触发“无用”节点的移除
func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := "value1", "value2", "value3"

	cap := len(k1 + v1 + k2 + v2)
	lru := New(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	_, ok := lru.Get(k1)
	if ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key[%s] failed!", k1)
	}
}

//测试回调函数能否被调用
func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}

	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))

	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s, but got keys is %s", expect, keys)
	}
}










