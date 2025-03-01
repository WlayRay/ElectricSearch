package service

import (
	"strings"
	"sync"
	"time"

	"github.com/WlayRay/ElectricSearch/util"

	etcdv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/time/rate"
)

type IServiceHub interface {
	Register(service, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) // 注册服务
	UnRegister(service, endpoint string) error                                         // 注销服务
	GetServiceEndpoints(service string) []string                                       // 服务发现
	GetServiceEndpoint(service string) string                                          // 根据负载均衡获取一台服务的endpoint
	Close()                                                                            // 关闭etcd连接
}

// 代理模式，对ServiceHub做一层代理，提供缓存和限流保护
type ServiceHubProxy struct {
	*ServiceHub // ServiceHubProxy拥有ServiceHub的所有方法（匿名成员变量）
	// 服务端地址缓存
	endpointCache sync.Map //维护每一个service下的所有servers
	limiter       *rate.Limiter
}

var (
	proxy     *ServiceHubProxy
	proxyOnce sync.Once
)

func GetServiceHubProxy(etcdEndpoints []string, heartRate int64, qps int) *ServiceHubProxy {
	if proxy == nil {
		proxyOnce.Do(func() {
			serviceHub := GetServiceHub(etcdEndpoints, heartRate)
			proxy = &ServiceHubProxy{
				ServiceHub:    serviceHub,
				endpointCache: sync.Map{},
				limiter:       rate.NewLimiter(rate.Every(time.Duration(1e9/qps)*time.Nanosecond), qps), //每隔1E9/qps纳秒产生一个令牌，即一秒钟之内产生qps个令牌。令牌桶的容量为qps
			}
		})
	}
	return proxy
}

func (proxy *ServiceHubProxy) watchEndpointsOfService(service string) {
	if _, exists := proxy.watched.LoadOrStore(service, true); exists {
		return
	}

	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	prefix := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/"
	watchChan := proxy.client.Watch(timeoutCtx, prefix, etcdv3.WithPrefix())
	util.Log.Printf("watch service: %s", service)

	go func() {
		for response := range watchChan {
			for _, event := range response.Events {
				util.Log.Printf("etcd event type: %s", event.Type)
				path := strings.Split(string(event.Kv.Key), "/")
				if len(path) > 2 {
					service := path[len(path)-2]
					endpoints := proxy.ServiceHub.GetServiceEndpoints(service)
					if len(endpoints) > 0 {
						proxy.endpointCache.Store(service, endpoints)
					} else {
						proxy.endpointCache.Delete(service)
					}
				}
			}
		}
	}()
}

func (proxy *ServiceHubProxy) GetServiceEndpoints(service string) []string {
	if !proxy.limiter.Allow() {
		return nil
	}
	proxy.watchEndpointsOfService(service)
	if endpoints, exists := proxy.endpointCache.Load(service); exists {
		return endpoints.([]string)
	} else {
		endpoints := proxy.ServiceHub.GetServiceEndpoints(service)
		if len(endpoints) > 0 {
			proxy.endpointCache.Store(service, endpoints)
		}
		return endpoints
	}
}
