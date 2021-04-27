package wangcache

//缓存值的抽象与封装

//抽象出一个只读数据结构 ByteView 用来表示缓存值
type ByteView struct {
	b []byte  // 存储真实的缓存值；选择byte类型是为了能够支持任意的数据类型的存储，例如字符串、图片等。
}

// Len returns the view's length (实现lru中的Value接口)
func (v ByteView) Len() int {
	return len(v.b)
}

// b 是只读的，使用ByteSlice()方法返回一个拷贝，防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String returns the data as a string, making a copy if necessary.
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}