package course

import "fmt"

// 判断 n 的二进制表示第 i 位是否为 1
func IsBit(n uint64, i int) bool {
	return n&(1<<(i-1)) != 0
}

// 将 n 的二进制表示第 i 位置为 1
func SetBit(n uint64, i int) uint64 {
	return n | (1 << (i - 1))
}

// 计算 n 的二进制表示中 1 的个数
func CountBits(n uint64) int {
	count := 0
	for i := 0; i < 64; i++ {
		if n&(1<<i) != 0 {
			count++
		}
	}
	return count
}

// 简易位图实现
type BitMap struct {
	table uint64
}

func NewBitMap(min int, arr []int) *BitMap {
	bitMap := &BitMap{}
	for _, v := range arr {
		if v < min || v > 64+min {
			panic(fmt.Sprintf("index out of range %v", v))
		}
		n := v - min
		bitMap.table = SetBit(bitMap.table, n)
	}
	return bitMap
}

// 求交集时必须指定一个相对的最小值
func BitMapIntersection(min int, a, b *BitMap) (res []int) {
	if a == nil || b == nil {
		return nil
	}

	arr := a.table & b.table
	for i := 1; i <= 64; i++ {
		if IsBit(arr, i) {
			res = append(res, i+min)
		}
	}
	return
}

// 有序列表求交集
func OrderedListIntersection(a, b []int) (res []int) {
	n, m := len(a), len(b)
	if n <= 0 || m <= 0 {
		return nil
	}

	for i, j := 0, 0; i < n && j < m; {
		if a[i] == b[j] {
			res = append(res, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}
	return
}
