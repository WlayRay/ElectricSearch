package service

import (
	"MiniES/util"
	"context"
	"sync"
	"time"

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

func (sentinal *Sentinel) GetGrpcConn(endpoint string) *grpc.ClientConn {
	if v, exists := sentinal.connPool.Load(endpoint); exists {
		conn := v.(*grpc.ClientConn)
		if conn.GetState() == connectivity.TransientFailure || conn.GetState() == connectivity.Shutdown {
			util.Log.Printf("sentinel %s is not serving, close it", endpoint)
			conn.Close()
			sentinal.connPool.Delete(endpoint)
		} else {
			return conn
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		util.Log.Printf("dial %s failed: %s", endpoint, err)
		return nil
	}
	util.Log.Printf("connect to grpc server %s", endpoint)
	sentinal.connPool.Store(endpoint, conn)
	return conn
}
