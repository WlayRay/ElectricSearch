package service

import (
	"math/rand"
	"sync/atomic"
)

type LoadBalancer interface {
	Take([]string) string
}

type RoundRobin struct {
	acc int64
}

func (rr *RoundRobin) Take(endPoints []string) string {
	if len(endPoints) == 0 {
		return ""
	}
	rr.acc = atomic.AddInt64(&rr.acc, 1) % int64(len(endPoints))
	return endPoints[rr.acc]
}

type RandomSelect struct{}

func (rs *RandomSelect) Take(endPoints []string) string {
	if len(endPoints) == 0 {
		return ""
	}
	return endPoints[rand.Intn(len(endPoints))]
}
