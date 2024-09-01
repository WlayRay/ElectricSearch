package utiltest

import (
	"ElectricSearch/util"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
)

var (
	conMap     = util.NewConcurrentHashMap(8, 1000)
	synMap     = sync.Map{}
	characters = "ABCDEFGHIJKLMNopqrstuvwxyz`!@#$%^&*(){};'/.,[]）（*&%#！~}："
	strings    = generateRandomString(50, 10000)
)

func generateRandomNum(min, max int) int {
	if min > max {
		fmt.Println("Error: min should not be greater than max.")
		return 0
	}
	return rand.Intn(max-min) + min
}

func generateRandomString(strLen, arrLen int) (res []string) {
	if strLen < 0 || arrLen < 0 {
		fmt.Println("Error: strLen and arrLen should not be negative.")
		return nil
	}
	for i := 0; i < arrLen; i++ {
		str := ""
		for j := 0; j < strLen; j++ {
			str += string(characters[generateRandomNum(0, len(characters))])
		}
		res = append(res, str)
	}
	return
}

func readConMap() {
	for i := 0; i < 5000; i++ {
		/*key := strconv.Itoa(int(rand.Int63()))
		conMap.Get(key)*/
		n := generateRandomNum(0, len(strings))
		conMap.Get(strconv.Itoa(n))
	}
}

func writeConMap() {
	for i := 0; i < 5000; i++ {
		/*key := strconv.Itoa(int(rand.Int63()))
		conMap.Set(key, key)*/
		n := generateRandomNum(0, len(strings))
		conMap.Set(strconv.Itoa(n), strings[n])
	}
}

func readSynMap() {
	for i := 0; i < 5000; i++ {
		/*key := strconv.Itoa(int(rand.Int63()))
		synMap.Load(key)*/
		n := generateRandomNum(0, len(strings))
		synMap.Load(strconv.Itoa(n))
	}
}

func writeSynMap() {
	for i := 0; i < 5000; i++ {
		/*key := strconv.Itoa(int(rand.Int63()))
		synMap.Store(key, key)*/
		n := generateRandomNum(0, len(strings))
		synMap.Store(strconv.Itoa(n), strings[n])
	}
}

func BenchmarkConMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		const P = 280
		wg := sync.WaitGroup{}
		wg.Add(2 * P)

		for i := 0; i < P; i++ {
			go func() {
				defer wg.Done()
				go readConMap()
			}()
		}

		for i := 0; i < P; i++ {
			go func() {
				defer wg.Done()
				go writeConMap()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkSynMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		const P = 280
		wg := sync.WaitGroup{}
		wg.Add(2 * P)

		for i := 0; i < P; i++ {
			go func() {
				defer wg.Done()
				go readSynMap()
			}()
		}

		for i := 0; i < P; i++ {
			go func() {
				defer wg.Done()
				go writeSynMap()
			}()
		}
		wg.Wait()
	}
}

// go test -run=none -bench=Benchmark.*Map -benchmem -count=1 -benchtime=100x ./util/test/concurrent_hash_map_test.go
/*
goos: linux
goarch: amd64
cpu: 12th Gen Intel(R) Core(TM) i7-12700
BenchmarkConMap-20           100          51027183 ns/op         4261595 B/op     988928 allocs/op
BenchmarkSynMap-20           100         105745341 ns/op        85242005 B/op    7865644 allocs/op
PASS
ok      command-line-arguments  16.469s

Author: WlayRay
Date:	2024/08/06
*/

func TestConcurrentHashMapIterator(t *testing.T) {
	for i := 0; i < len(strings); i++ {
		conMap.Set(strconv.Itoa(i), strings[i])
	}

	it := conMap.NewIterator()
	entry := it.Next()
	for entry != nil {
		idx, _ := strconv.Atoi(entry.Key)
		if entry.Value != strings[idx] {
			t.Fail()
		}
		entry = it.Next()
	}
}
