package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/WlayRay/ElectricSearch/types"
	"github.com/WlayRay/ElectricSearch/util"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Sentinel struct {
	hub      IServiceHub
	connPool sync.Map
}

func NewSentinel(etcdServers []string) *Sentinel {
	return &Sentinel{
		// hub: GetServiceHub(etcdServers, 10), //直接访问ServiceHub
		hub:      GetServiceHubProxy(etcdServers, 10, 100), //走代理HubProxy
		connPool: sync.Map{},
	}
}

func (sentinel *Sentinel) GetGrpcConn(endpoint string) *grpc.ClientConn {
	if v, exists := sentinel.connPool.Load(endpoint); exists {
		conn := v.(*grpc.ClientConn)
		// 检查连接状态是否为Ready
		if conn.GetState() != connectivity.Ready {
			util.Log.Printf("sentinel %s is not ready (state: %v), close it", endpoint, conn.GetState())
			conn.Close()
			sentinel.connPool.Delete(endpoint)
		} else {
			return conn
		}
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		util.Log.Printf("dial %s failed: %s", endpoint, err)
		return nil
	}
	util.Log.Printf("successfully connected to grpc server %s", endpoint)
	sentinel.connPool.Store(endpoint, conn)
	return conn
}

func (sentinel *Sentinel) AddDoc(doc types.Document) (int, error) {
	endpoint := sentinel.hub.GetServiceEndpoint(indexService)
	if len(endpoint) == 0 {
		return 0, fmt.Errorf("there is no alive index worker")
	}

	conn := sentinel.GetGrpcConn(endpoint)
	if conn == nil {
		return 0, fmt.Errorf("connection to worker %s failed", endpoint)
	}

	client := NewIndexServiceClient(conn)
	affected, err := client.AddDoc(context.Background(), &doc)
	if err != nil {
		return 0, err
	}

	util.Log.Printf("add doc %s to worker %s, affected %d", doc.Id, endpoint, affected.Count)
	return int(affected.Count), nil
}

func (sentinel *Sentinel) DeleteDoc(docId string) int {
	endpoints := sentinel.hub.GetServiceEndpoints(indexService)
	if len(endpoints) == 0 {
		return 0
	}

	var n uint32
	wg := sync.WaitGroup{}
	wg.Add(len(endpoints))
	for _, endpoint := range endpoints {
		go func(endpoint string) {
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn != nil {
				client := NewIndexServiceClient(conn)
				affected, err := client.DeleteDoc(context.Background(), &DocId{docId})
				if err != nil {
					util.Log.Printf("delete doc %s from worker %s failed: %s", docId, endpoint, err)
				} else if affected.Count > 0 {
					atomic.AddUint32(&n, uint32(affected.Count))
					util.Log.Printf("delete doc %s from worker %s, affected %d", docId, endpoint, affected.Count)
				}
			}
		}(endpoint)
	}
	wg.Wait()
	return int(n)
}

func (sentinel *Sentinel) Search(querys *types.TermQuery, onFlag, offFlag uint64, orFlags []uint64) []*types.Document {
	endpoints := sentinel.hub.GetServiceEndpoints(indexService)
	if len(endpoints) == 0 {
		return nil
	}

	docs := make([]*types.Document, 0, 1000)
	resultCh := make(chan *types.Document, 1000)
	wg := sync.WaitGroup{}
	wg.Add(len(endpoints))
	for _, endpoint := range endpoints {
		go func(endpoint string) {
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn != nil {
				client := NewIndexServiceClient(conn)
				result, err := client.Search(context.Background(), &SearchRequest{querys, onFlag, offFlag, orFlags})
				if err != nil {
					util.Log.Printf("search from worker %s failed: %s", endpoint, err)
				} else if len(result.Documents) > 0 {
					for _, doc := range result.Documents {
						resultCh <- doc
					}
				}
			}
		}(endpoint)
	}

	received := make(chan struct{})
	go func() {
		for {
			doc, ok := <-resultCh
			if !ok {
				received <- struct{}{}
				break
			}
			docs = append(docs, doc)
		}
	}()
	wg.Wait()
	close(resultCh)
	<-received
	return docs
}

func (sentinel *Sentinel) Count() int {
	var n uint32
	endpoints := sentinel.hub.GetServiceEndpoints(indexService)
	if len(endpoints) == 0 {
		return 0
	}
	wg := sync.WaitGroup{}
	wg.Add(len(endpoints))
	for _, endpoint := range endpoints {
		go func(endpoint string) {
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn != nil {
				client := NewIndexServiceClient(conn)
				result, err := client.Count(context.Background(), &CountRequest{})
				if err != nil {
					util.Log.Printf("count from worker %s failed: %s", endpoint, err)
				} else if result.Count > 0 {
					atomic.AddUint32(&n, uint32(result.Count))
					util.Log.Printf("count from worker %s, count %d", endpoint, result.Count)
				}
			}
		}(endpoint)
	}
	wg.Wait()
	return int(n)
}

func (sentinel *Sentinel) Close() (err error) {
	sentinel.connPool.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*grpc.ClientConn); ok {
			err = conn.Close()
		}
		return true
	})
	sentinel.hub.Close()
	return
}
