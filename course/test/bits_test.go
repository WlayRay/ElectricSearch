package coursetest

import (
	"MiniES/course"
	"fmt"
	"sort"
	"testing"
)

func TestIsBit(t *testing.T) {
	tests := []struct {
		n        uint64
		i        int
		expected bool
	}{
		{0b10101, 1, true},
		{0b10101, 2, false},
		{0b10101, 3, true},
		{0b10101, 4, false},
		{0b10101, 5, true},
		{0b10101, 6, true},
	}

	for _, tc := range tests {
		if got := course.IsBit(tc.n, tc.i); got != tc.expected {
			fmt.Printf("IsBit(%08b, %d) = %v; expected %v", tc.n, tc.i, got, tc.expected)
		} else {
			fmt.Printf("IsBit(%08b, %d) = %v\n", tc.n, tc.i, got)
		}
	}
}

func TestSetBit(t *testing.T) {
	tests := []struct {
		n        uint64
		i        int
		expected uint64
	}{
		{0b10101, 1, 0b10101},
		{0b10101, 2, 0b10111},
		{0b10101, 3, 0b10101},
		{0b10101, 4, 0b11101},
		{0b10101, 5, 0b10101},
		{0b10101, 6, 0b110101},
	}

	for _, tc := range tests {
		if got := course.SetBit(tc.n, tc.i); got != tc.expected {
			fmt.Printf("SetBit(%08b, %d) = %08b; expected %b", tc.n, tc.i, got, tc.expected)
		} else {
			fmt.Printf("SetBit(%08b, %d) = %08b\n", tc.n, tc.i, got)
		}
	}
}

func TestCountBits(t *testing.T) {
	tests := []struct {
		n        uint64
		expected int
	}{
		{0b10101, 3},
		{0b10111, 4},
		{0b10101, 3},
		{0b11101, 4},
		{0b10101, 3},
		{0b110101, 4},
	}

	for _, tc := range tests {
		if got := course.CountBits(tc.n); got != tc.expected {
			fmt.Printf("CountBits(%08b) = %d; expected %d", tc.n, got, tc.expected)
		} else {
			fmt.Printf("CountBits(%08b) = %d\n", tc.n, got)
		}
	}
}

func TestIntersection(t *testing.T) {
	basic := 0
	arr1 := []int{1, 2, 5, 2, 4}
	arr2 := []int{4, 1, 8, 5, 2, 3}
	bitMap1 := course.NewBitMap(basic, arr1)
	bitMap2 := course.NewBitMap(basic, arr2)

	fmt.Println(course.BitMapIntersection(basic, bitMap1, bitMap2))

	sort.Slice(arr1, func(i, j int) bool { return arr1[i] < arr1[j] })
	sort.Slice(arr2, func(i, j int) bool { return arr2[i] < arr2[j] })
	fmt.Println(course.OrderedListIntersection(arr1, arr2))
}
