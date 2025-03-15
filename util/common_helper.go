package util

import (
	"bytes"
	"encoding/binary"
	"slices"
)

// IntToBytes 整型转换成字节
func IntToBytes(n int) []byte {
	x := int64(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// BytesToInt 字节转换成整型
func BytesToInt(bs []byte) int {
	bytesBuffer := bytes.NewBuffer(bs)
	var x int64
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}

// CombineUint32 把两个uint32拼接成一个uint64
func CombineUint32(a, b uint32) uint64 {
	return uint64(a)<<32 + uint64(b)
}

// DisassembleUint64 把一个uint64拆成两个uint32
func DisassembleUint64(x uint64) (uint32, uint32) {
	return uint32(x >> 32), uint32(x)
}

func RemoveElement[T comparable](s []T, target T) []T {
	for i, v := range s {
		if v == target {
			return slices.Delete(s, i, i+1)
		}
	}
	return s
}
