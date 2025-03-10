package service

import (
	"context"
	"fmt"
	"github.com/WlayRay/ElectricSearch/etcd"
	"github.com/dgryski/go-farm"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WlayRay/ElectricSearch/types"
	"github.com/WlayRay/ElectricSearch/util"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Sentinel struct {
	Hub      IServiceHub
	seed     int
	connPool sync.Map
}

func NewSentinel(etcdServers []string) *Sentinel {
	return &Sentinel{
		// Hub: GetServiceHub(etcdServers, 10), //直接访问ServiceHub
		Hub:      GetServiceHubProxy(etcdServers, 10, 100), //走代理HubProxy
		connPool: sync.Map{},
		seed:     0,
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
	groupIndex := sentinel.getGroupIndex(doc.Id)
	endpoints := sentinel.Hub.GetServiceEndpoints(fmt.Sprintf("group-%d", groupIndex))
	if len(endpoints) == 0 {
		return 0, fmt.Errorf("there is no gourp can be used")
	}

	var total uint32
	var wg sync.WaitGroup
	wg.Add(len(endpoints))
	for _, endpoint := range endpoints {
		go func(endpoint string) {
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn != nil {
				client := NewIndexServiceClient(conn)
				affected, err := client.AddDoc(context.Background(), &doc)
				if err != nil {
					util.Log.Printf("add doc %s to worker %s failed: %s", doc.Id, endpoint, err)
				} else {
					atomic.AddUint32(&total, uint32(affected.Count))
				}
			}

		}(endpoint)
	}

	wg.Wait()
	util.Log.Printf("add doc %s to workers %v, affected %d", doc.Id, endpoints, total)
	return int(total), nil
}

func (sentinel *Sentinel) DeleteDoc(docId string) int {
	groupIndex := sentinel.getGroupIndex(docId)
	endpoints := sentinel.Hub.GetServiceEndpoints(fmt.Sprintf("group-%d", groupIndex))
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
					atomic.AddUint32(&n, affected.Count)
					util.Log.Printf("delete doc %s from worker %s, affected %d", docId, endpoint, affected.Count)
				}
			}
		}(endpoint)
	}

	wg.Wait()
	return int(n)
}

func (sentinel *Sentinel) Search(querys *types.TermQuery, onFlag, offFlag uint64, orFlags []uint64) []*types.Document {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	docs := make([]*types.Document, 0, 1000)
	resultCh := make(chan *types.Document, 1000)
	errCh := make(chan error, 1)

	groupCount := sentinel.getGroupCount()
	if groupCount == 0 {
		return nil
	}

	var wg sync.WaitGroup
	for i := 0; i < groupCount; i++ {
		group := fmt.Sprintf("group-%d", i)
		endpoints := sentinel.Hub.GetServiceEndpoints(group)
		if len(endpoints) == 0 {
			continue
		}

		wg.Add(len(endpoints))

		// 通过负载均衡获取一个worker的endpoint
		endpoint := sentinel.Hub.GetServiceEndpoint(group)
		go func(endpoint string) {
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn == nil {
				errCh <- fmt.Errorf("failed to get connection for endpoint %s", endpoint)
				return
			}

			client := NewIndexServiceClient(conn)
			result, err := client.Search(ctx, &SearchRequest{
				Query:   querys,
				OnFlag:  onFlag,
				OffFlag: offFlag,
				OrFlags: orFlags,
			})

			if err != nil {
				select {
				case errCh <- fmt.Errorf("search from worker %s failed: %w", endpoint, err):
				default:
				}
				return
			}

			for _, doc := range result.Documents {
				select {
				case resultCh <- doc:
				case <-ctx.Done():
					return
				}
			}
		}(endpoint)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for {
		select {
		case doc, ok := <-resultCh:
			if !ok {
				return docs
			}
			docs = append(docs, doc)
		case err := <-errCh:
			util.Log.Printf("search error: %v", err)
		case <-ctx.Done():
			util.Log.Printf("search timeout")
			return docs
		}
	}
}

func (sentinel *Sentinel) Count() int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var n uint32
	resultCh := make(chan uint32, 1000)
	errCh := make(chan error, 1)

	groupCount := sentinel.getGroupCount()
	if groupCount == 0 {
		return 0
	}

	var wg sync.WaitGroup
	for i := 0; i < groupCount; i++ {
		group := fmt.Sprintf("group-%d", i)
		endpoints := sentinel.Hub.GetServiceEndpoints(group)
		if len(endpoints) == 0 {
			continue
		}

		wg.Add(len(endpoints))
		endpoint := sentinel.Hub.GetServiceEndpoint(group)
		go func(endpoint string) {
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn == nil {
				errCh <- fmt.Errorf("failed to get connection for endpoint %s", endpoint)
				return
			}

			client := NewIndexServiceClient(conn)
			result, err := client.Count(ctx, &CountRequest{})
			if err != nil {
				select {
				case errCh <- fmt.Errorf("count from worker %s failed: %w", endpoint, err):
				default:
				}
				return
			}

			select {
			case resultCh <- result.Count:
			case <-ctx.Done():
				return
			}
		}(endpoint)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for {
		select {
		case count, ok := <-resultCh:
			if !ok {
				return int(n)
			}
			atomic.AddUint32(&n, count)
		case err := <-errCh:
			util.Log.Printf("count error: %v", err)
		case <-ctx.Done():
			util.Log.Printf("count timeout")
			return int(n)
		}
	}
}

func (sentinel *Sentinel) Close() (err error) {
	sentinel.connPool.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*grpc.ClientConn); ok {
			err = conn.Close()
		}
		return true
	})
	sentinel.Hub.Close()
	return
}

func (*Sentinel) getGroupCount() int {
	etcdServers := util.ConfigMap["etcd"].(map[string]any)["servers"].([]string)
	etcdConn, err := etcd.GetEtcdClient(etcdServers)
	if err != nil {
		util.Log.Fatalf("get etcd client failed: %s", err)
	}

	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	resp, getErr := etcdConn.Get(timeoutCtx, ServiceRootPath+indexName+"/total-shards", etcdv3.WithPrefix())
	if getErr == nil {
		if resp != nil {
			totalGroups, _ := strconv.Atoi(string(resp.Kvs[0].Value))
			return totalGroups
		}
	}

	return 0
}

func (sentinel *Sentinel) getGroupIndex(key string) int {
	return int(farm.Hash32WithSeed([]byte(key), uint32(sentinel.seed)) % uint32(sentinel.getGroupCount()))
}
