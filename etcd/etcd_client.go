package etcd

import (
	"sync"
	"time"

	etcdv3 "go.etcd.io/etcd/client/v3"
)

var (
	etcdClient *etcdv3.Client
	once       sync.Once
	err        error
)

func GetEtcdClient(etcdEndpoints []string) (*etcdv3.Client, error) {
	once.Do(func() {
		etcdClient, err = etcdv3.New(etcdv3.Config{
			Endpoints:          etcdEndpoints,
			DialTimeout:        5 * time.Second,
			MaxCallSendMsgSize: 10 * 1024 * 1024, // 设置发送消息的最大大小
			MaxCallRecvMsgSize: 10 * 1024 * 1024, // 设置接收消息的最大大小
		})
	})

	if err != nil {
		return nil, err
	}
	return etcdClient, nil
}
