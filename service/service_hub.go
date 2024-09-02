package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/WlayRay/ElectricSearch/util"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

const (
	SERVICE_ROOT_PATH = "/github.com/WlayRay/ElectricSearch/index" // etcd key的前缀
)

// 服务注册中心
type ServiceHub struct {
	client       *etcdv3.Client
	heartRate    int64    // server 每间隔heartRate向etcd发送心跳，同时续约
	watched      sync.Map // 存储已经监听过的service
	loadBalancer LoadBalancer
}

// 使用单例模式创建ServiceHub，包外需通过GetServiceHub获取实例
var (
	serviceHub *ServiceHub
	hubOnce    sync.Once
)

func GetServiceHub(etcdEndpoints []string, heartRate int64) *ServiceHub {
	if serviceHub == nil {
		hubOnce.Do(func() {
			if client, err := etcdv3.New(etcdv3.Config{
				Endpoints:   etcdEndpoints, // TODO 将etcd的endpoints配置放到配置文件中
				DialTimeout: 5 * time.Second,
			}); err != nil {
				util.Log.Fatalf("etcd client init failed: %v", err)
			} else {
				serviceHub = &ServiceHub{
					client:       client,
					heartRate:    heartRate,
					loadBalancer: &RoundRobin{}, // TODO 将使用的负载均衡算法放到配置文件中
				}
			}
		})
	}
	return serviceHub
}

func (hub *ServiceHub) Regist(service, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) {
	ctx := context.Background()
	if leaseID <= 0 {
		// 创建一个有效期为heartRate的租约（单位：秒）
		if lease, err := hub.client.Grant(ctx, hub.heartRate); err != nil {
			util.Log.Printf("create lease failed: %v", err)
			return 0, err
		} else {
			keys := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/" + endpoint
			// 服务注册
			if _, err := hub.client.Put(ctx, keys, "", etcdv3.WithLease(lease.ID)); err != nil {
				util.Log.Printf("regist service %s endpoint %s failed: %v", service, endpoint, err)
				return leaseID, err
			} else {
				return leaseID, nil
			}
		}
	} else {
		// 续租
		if _, err := hub.client.KeepAliveOnce(ctx, leaseID); err != rpctypes.ErrLeaseNotFound {
			return hub.Regist(service, endpoint, leaseID)
		} else if err != nil {
			util.Log.Printf("keep lease %d failed: %v", leaseID, err)
			return 0, err
		} else {
			return leaseID, nil
		}
	}
}

// 注销服务
func (hub *ServiceHub) UnRegist(service, endpoint string) error {
	ctx := context.Background()
	key := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/" + endpoint
	if _, err := hub.client.Delete(ctx, key); err != nil {
		util.Log.Printf("unregist service %s endpoint %s failed: %v", service, endpoint, err)
		return err
	} else {
		util.Log.Printf("unregist service %s endpoint %s success", service, endpoint)
		return nil
	}
}

// 服务发现，client每次进行RPC调用之前都查询etcd，获取server集合，然后采用负载均衡算法选择一台server。或者也可以把负载均衡的功能放到注册中心，即放到getServiceEndpoints函数里，让它只返回一个server
func (hub *ServiceHub) GetServiceEndpoints(service string) []string {
	ctx := context.Background()
	prefix := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service
	if resp, err := hub.client.Get(ctx, prefix, etcdv3.WithPrefix()); err != nil {
		util.Log.Printf("get service %s endpoints failed: %v", service, err)
		return nil
	} else {
		endpoints := make([]string, 0, len(resp.Kvs))
		for _, kv := range resp.Kvs {
			path := strings.Split(string(kv.Key), "/") // 只需要key，不需要value
			endpoints = append(endpoints, path[len(path)-1])
		}
		util.Log.Printf("now the %s service has endpoints: %v", service, endpoints)
		return endpoints
	}
}

// 根据负载均衡，从众多endpoint中选择一个
func (hub *ServiceHub) GetServiceEndpoint(service string) string {
	return hub.loadBalancer.Take(hub.GetServiceEndpoints(service))
}

// 关闭etcd客户端连接
func (hub *ServiceHub) Close() {
	hub.client.Close()
}
