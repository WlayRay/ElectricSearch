package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/WlayRay/ElectricSearch/util"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

type Lock struct {
	Key     string
	Client  *etcdv3.Client
	LeaseID etcdv3.LeaseID
	Context context.Context
}

// AcquireDistributedLock 尝试加锁
func AcquireDistributedLock(client *etcdv3.Client, lockKey string, maxRetries int, retryInterval time.Duration, leaseTTL int64) (*Lock, error) {
	var lock *etcdv3.TxnResponse
	var err error

	// 创建租约
	lease, err := client.Grant(context.Background(), leaseTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create lease: %v", err)
	}

	for range maxRetries {
		timeoutCtx, _ := util.GetDefaultTimeoutContext()

		lock, err = client.Txn(timeoutCtx).
			If(etcdv3.Compare(etcdv3.CreateRevision(lockKey), "=", 0)).
			Then(etcdv3.OpPut(lockKey, "locked", etcdv3.WithLease(lease.ID))).
			Commit()

		if err == nil && lock.Succeeded {
			return &Lock{
				Key:     lockKey,
				Client:  client,
				LeaseID: lease.ID,
				Context: timeoutCtx,
			}, nil
		}

		time.Sleep(retryInterval)
	}

	// 如果最终未能获取锁，释放租约
	client.Revoke(context.Background(), lease.ID)
	return nil, fmt.Errorf("failed to acquire lock after %d retries: %v", maxRetries, err)
}

// Release 释放锁
func (l *Lock) Release() error {
	_, err := l.Client.Delete(l.Context, l.Key)
	if err != nil {
		return fmt.Errorf("failed to release lock: %v", err)
	}
	// 释放租约
	_, err = l.Client.Revoke(l.Context, l.LeaseID)
	if err != nil {
		return fmt.Errorf("failed to revoke lease: %v", err)
	}
	return nil
}
