package utiltest

import (
	"ElectricSearch/util"
	"fmt"
	"sync"
	"testing"

	"github.com/dgryski/go-farm"
)

func TestNewWorker(t *testing.T) {
	_, err := util.NewWorker(0)
	if err != nil {
		t.Errorf("NewWorker() error = %v", err)
	}

	_, err = util.NewWorker(1023)
	if err != nil {
		t.Errorf("NewWorker() error = %v", err)
	}

	_, err = util.NewWorker(1024)
	if err != nil {
		t.Logf("NewWorker error: %v", err)
	}

	ip, _ := util.GetLocalIP()
	workerId := uint64(farm.Hash64WithSeed([]byte(ip), 0)) % 1023
	w, _ := util.NewWorker(uint64(workerId))
	t.Logf("workerId: %d", workerId)
	if workerId != w.GetWorkerId() {
		t.Errorf("GetWorkerId() = %v, want %v", workerId, 1)
	}

	var (
		p = 360
		k = 360
	)
	var wg sync.WaitGroup
	result := make(chan uint64, p*k)
	wg.Add(p)
	for i := 0; i < p; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < k; i++ {
				result <- w.GetId()
			}
		}()
	}
	wg.Wait()
	close(result)

	resultSet := make(map[uint64]struct{}, p*k)
	count, errors := 0, 0
	for v := range result {
		if _, exists := resultSet[v]; exists {
			fmt.Printf("Incorrect generation of duplicate ID: %d\n", v)
			errors++
		} else {
			resultSet[v] = struct{}{}
		}
		count++
	}
	fmt.Printf("%d IDs generated, %d errors, the error rate is %.2f%%\n", count, errors, float64(errors)/float64(count)*100)
}
