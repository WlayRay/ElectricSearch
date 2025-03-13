package service

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WlayRay/ElectricSearch/etcd"
	"github.com/WlayRay/ElectricSearch/util"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	etcdv3 "go.etcd.io/etcd/client/v3"
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
)

func GetServiceHub(etcdEndpoints []string, heartRate int64) *ServiceHub {
	if serviceHub == nil {
		if etcdClient, err := etcd.GetEtcdClient(etcdEndpoints); err != nil {
			util.Log.Fatalf("etcd client init failed: %v", err)
		} else {
			serviceHub = &ServiceHub{
				client:       etcdClient,
				heartRate:    heartRate,
				loadBalancer: &RoundRobin{}, // TODO 将使用的负载均衡算法放到配置文件中
			}
		}
	}
	return serviceHub
}

func (Hub *ServiceHub) Register(group, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) {
	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	if leaseID <= 0 {
		// 创建一个有效期为heartRate的租约（单位：秒）
		if lease, err := Hub.client.Grant(timeoutCtx, Hub.heartRate); err != nil {
			util.Log.Printf("create lease failed: %v", err)
			return 0, err
		} else {
			keys := ServiceRootPath + indexName + "/" + group + "/" + endpoint
			// 服务注册(向ETCD中写入一个key)
			if _, err := Hub.client.Put(timeoutCtx, keys, "", etcdv3.WithLease(lease.ID)); err != nil {
				util.Log.Printf("register service %s endpoint %s failed: %v", group, endpoint, err)
				return leaseID, err
			} else {
				return leaseID, nil
			}
		}
	} else {
		// 续租
		if _, err := Hub.client.KeepAliveOnce(timeoutCtx, leaseID); !errors.Is(err, rpctypes.ErrLeaseNotFound) {
			return Hub.Register(group, endpoint, leaseID)
		} else if err != nil {
			util.Log.Printf("keep lease %d failed: %v", leaseID, err)
			return 0, err
		} else {
			return leaseID, nil
		}
	}
}

// 注销服务
func (Hub *ServiceHub) UnRegister(group, endpoint string) error {
	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	key := ServiceRootPath + indexName + "/" + group + "/" + endpoint
	if _, err := Hub.client.Delete(timeoutCtx, key); err != nil {
		util.Log.Printf("unregister worker %s endpoint %s failed: %v", group, endpoint, err)
		return err
	} else {
		util.Log.Printf("unregister worker %s endpoint %s success", group, endpoint)
		return nil
	}
}

// 服务发现，client每次进行RPC调用之前都查询etcd，获取server集合，然后采用负载均衡算法选择一台server。
func (Hub *ServiceHub) GetServiceEndpoints(group string) []string {
	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	prefix := ServiceRootPath + indexName + "/" + group
	if resp, err := Hub.client.Get(timeoutCtx, prefix, etcdv3.WithPrefix()); err != nil {
		util.Log.Printf("get group %s endpoints failed: %v", group, err)
		return nil
	} else if resp.Count != 0 {
		endpoints := make([]string, 0, len(resp.Kvs))
		for _, kv := range resp.Kvs {
			path := strings.Split(string(kv.Key), "/") // 只需要key，不需要value
			endpoints = append(endpoints, path[len(path)-1])
		}
		util.Log.Printf("now the %s group has endpoints: %v", group, endpoints)
		return endpoints
	} else {
		return nil
	}
}

// 根据负载均衡，从众多endpoint中选择一个
func (Hub *ServiceHub) GetServiceEndpoint(group string) string {
	return Hub.loadBalancer.Take(Hub.GetServiceEndpoints(group))
}

// 关闭etcd客户端连接
func (Hub *ServiceHub) Close() {
	_ = Hub.client.Close()
}

func (Hub *ServiceHub) addIndexGroup() int {
	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	// 获取分布式锁
	lock, err := etcd.AcquireDistributedLock(Hub.client, ServiceRootPath+indexName+"/group-lock", 3, 2*time.Second, 10)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			util.Log.Fatalf("failed to release lock: %v", err)
		}
	}()

	key := ServiceRootPath + indexName + "/total-shards"
	if res, err := Hub.client.Get(timeoutCtx, key, etcdv3.WithPrefix()); err == nil {
		if res == nil {
			return 0
		}

		var (
			count int
			err   error
		)
		if len(res.Kvs) > 0 {
			value := res.Kvs[0].Value
			count, err = strconv.Atoi(string(value))
		}
		if err != nil {
			util.Log.Printf("failed to convert value to int: %v", err)
			return 0
		}
		count++
		_, err = Hub.client.Put(timeoutCtx, key, strconv.Itoa(count))
		if err != nil {
			util.Log.Printf("failed to put updated value to etcd: %v", err)
			return 0
		}
		return count
	}
	return 0
}

func (Hub *ServiceHub) subIndexGroup() {
	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	// 获取分布式锁
	lock, err := etcd.AcquireDistributedLock(Hub.client, ServiceRootPath+indexName+"/group-lock", 3, 2*time.Second, 10)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			util.Log.Fatalf("failed to release lock: %v", err)
		}
	}()

	key := ServiceRootPath + indexName + "/total-shards"
	if res, err := Hub.client.Get(timeoutCtx, key, etcdv3.WithPrefix()); err == nil {
		if res == nil {
			return
		}
		value := res.Kvs[0].Value
		count, err := strconv.Atoi(string(value))
		if err != nil {
			util.Log.Printf("failed to convert value to int: %v", err)
			return
		}
		count--
		if count <= 0 {
			_, _ = Hub.client.Delete(timeoutCtx, key)
			return
		}
		_, err = Hub.client.Put(timeoutCtx, key, strconv.Itoa(count))
		if err != nil {
			util.Log.Printf("failed to put updated value to etcd: %v", err)
			return
		}
		return
	}
}

func (Hub *ServiceHub) CountIndexGroup() int {
	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	// 获取分布式锁
	lock, err := etcd.AcquireDistributedLock(Hub.client, ServiceRootPath+indexName+"/group-lock", 3, 2*time.Second, 10)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			util.Log.Fatalf("failed to release lock: %v", err)
		}
	}()

	key := ServiceRootPath + indexName + "/total-shards"
	if res, err := Hub.client.Get(timeoutCtx, key, etcdv3.WithPrefix()); err == nil {
		if res == nil {
			return 0
		}
		if res.Count > 0 {
			value := res.Kvs[0].Value
			count, err := strconv.Atoi(string(value))
			if err != nil {
				util.Log.Printf("failed to convert value to int: %v", err)
				return 0
			}
			return count
		}
	}
	return 0
}
