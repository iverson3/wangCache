package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T)  {
	// 自定义Hash函数，方便测试分析结果
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	hash.Add("6", "4", "2")

	// 添加上面三个节点后，会生成 3*3 个虚拟节点： 6 16 26  4 14 24  2 12 22 (都是哈希值)
	// 6 16 26 对应真实节点"6"    4 14 24 对应真实节点"4"   2 12 22 对应真实节点"2"
    // 添加到哈希环上并排序之后会变成： 2 4 6 12 14 16 22 24 26  (此时哈希环上只有这9个哈希值)

    testCases := map[string]string {
    	"2":  "2",  // key="2"  在哈希环上找到的虚拟节点的hash值就是 2   所以对应的真实节点是"2"
    	"11": "2",  // key="11" 在哈希环上找到的虚拟节点的hash值就是 12  所以对应的真实节点是"2"
    	"23": "4",  // key="23" 在哈希环上找到的虚拟节点的hash值就是 24  所以对应的真实节点是"4"
    	"27": "2",  // key="27" 在哈希环上找到的虚拟节点的hash值就是 2   所以对应的真实节点是"2"
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// 8 18 28 对应真实节点"8"
	hash.Add("8")

	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}