package servicetest

import (
	"ElectricSearch/service"
	"fmt"
	"time"

	// "fmt"
	"testing"
)

func TestGetServiceEndpointByProxy(t *testing.T) {
	var (
		qps         = 10
		etcdServers = []string{"127.0.0.1:2379"}
		Endpoints   = []string{"127.0.0.1:6080", "127.0.0.2:6081", "127.0.0.3:6082"}
		serviceName = "TestService"
	)

	proxy := service.GetServiceHubProxy(etcdServers, 10, qps)

	for i := 0; i < 3; i++ {
		proxy.Regist(serviceName, Endpoints[i], 0)
		defer proxy.UnRegist(serviceName, Endpoints[i])
		_ = proxy.GetServiceEndpoints(serviceName)
	}

	time.Sleep(1 * time.Second)
	for i := 0; i < qps+5; i++ { // 桶里只有10个令牌，后五次访问会失败
		endpoints := proxy.GetServiceEndpoints(serviceName)
		fmt.Printf("%d endpoints: %v\n", len(Endpoints), endpoints)
	}

	time.Sleep(1 * time.Second) // 暂停一秒使令牌容量恢复
	for i := 0; i < qps+5; i++ {
		endpoints := proxy.GetServiceEndpoints(serviceName)
		fmt.Printf("%d endpoints: %v\n", len(Endpoints), endpoints)
	}
}
