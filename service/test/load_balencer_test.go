package servicetest

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/WlayRay/ElectricSearch/v1.0.0/service"
)

var (
	balancer  service.LoadBalancer
	endpoints = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
)

func testLB(balancer service.LoadBalancer) {
	const P = 100
	const LOOP = 100
	selected := make(chan string, P*LOOP)
	wg := sync.WaitGroup{}
	wg.Add(P)
	for i := 0; i < P; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < LOOP; i++ {
				endpoint := balancer.Take(endpoints)
				selected <- endpoint
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	close(selected)

	cm := make(map[string]int, len(endpoints))
	for {
		endpoint, ok := <-selected
		if !ok {
			break
		}
		value, ok := cm[endpoint]
		if ok {
			cm[endpoint] = value + 1
		} else {
			cm[endpoint] = 1
		}
	}

	for k, v := range cm {
		fmt.Println(k, v) //打印每个endpoint被使用了几次
	}
}

func TestRandomSelect(t *testing.T) {
	balancer = new(service.RandomSelect)
	testLB(balancer)
}

func TestRoudRobin(t *testing.T) {
	balancer = new(service.RoundRobin)
	testLB(balancer)
}
