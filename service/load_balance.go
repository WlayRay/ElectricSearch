package service

import (
	"math/rand"
	"sync/atomic"
)

type LoadBalancer interface {
	Take([]string) string
}

type RoundRobin struct {
	acc uint64
}

func (rr *RoundRobin) Take(endPoints []string) string {
	if len(endPoints) == 0 {
		return ""
	}
	atomic.AddUint64(&rr.acc, 1)
	return endPoints[rr.acc%uint64(len(endPoints))]
}

type RandomSelect struct{}

func (rs *RandomSelect) Take(endPoints []string) string {
	if len(endPoints) == 0 {
		return ""
	}
	return endPoints[rand.Intn(len(endPoints))]
}
