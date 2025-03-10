package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/WlayRay/ElectricSearch/internal/kvdb"
	"github.com/WlayRay/ElectricSearch/types"
	"github.com/WlayRay/ElectricSearch/util"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

const (
	ServiceRootPath = "/electric-search/" // etcd key的前缀
)

var (
	indexName      string
	currentGroup   string
	distributedMap map[string]any
)

func init() {
	// 尝试从环境变量中读取 index-name 和 group-index
	indexName = os.Getenv("INDEX_NAME")
	groupIndexStr := os.Getenv("GROUP_INDEX")

	if indexName == "" || groupIndexStr == "" {
		// 如果环境变量为空，从 ConfigMap 中读取
		var ok bool
		distributedMap, ok = util.ConfigMap["distributed"].(map[string]any)
		if ok {
			if indexName == "" {
				indexName = distributedMap["index-name"].(string)
			}
			if groupIndexStr == "" {
				groupIndex, _ := distributedMap["group-index"].(int)
				groupIndexStr = strconv.Itoa(groupIndex)
			}
		} else {
			panic("distributed configuration not found or error!")
		}
	}

	groupIndex, err := strconv.Atoi(groupIndexStr)
	if err != nil {
		panic("group-index invalid!")
	}

	currentGroup = fmt.Sprintf("group-%d", groupIndex)
}

// IndexServiceWorker 是一个grpc服务，用于索引文档
type IndexServiceWorker struct {
	Indexer  *Indexer
	Hub      *ServiceHub
	selfAddr string
}

func (service *IndexServiceWorker) Init(etcdEndpoints []string, currentGroup, heartRate int) error {
	Hub := GetServiceHub(etcdEndpoints, int64(heartRate))
	service.Hub = Hub
	service.Indexer = new(Indexer)

	var docNumEstimate, dbType int
	var dbPath string

	indexConfig, ok := util.ConfigMap["index"].(map[string]any)
	if !ok {
		panic("index configuration not found!")
	}

	// 初始化预估最大文档数量
	if v, ok := indexConfig["document-estimate-num"]; ok {
		docNumEstimate, _ = v.(int)
	} else {
		docNumEstimate = 50000
	}

	// 初始化正排索引文件存储路径
	if v, ok := indexConfig["db-path"]; ok {
		dbPath = util.RootPath + strings.Replace(v.(string), "\"", "", -1)
		if dbPath[len(dbPath)-1] != '/' {
			dbPath += "/"
		}
		// 初始化正排索引使用的数据库类型
		if v, ok := indexConfig["db-type"]; ok {
			switch v {
			case "bolt":
				dbType = kvdb.BOLT
				dbPath += "bolt_db/bolt"
			default:
				dbType = kvdb.BADGER
				dbPath += "badger_db"
			}
		} else {
			dbType = kvdb.BOLT
		}

		dbPath += "_" + strconv.Itoa(currentGroup)
		if ip, err := util.GetLocalIP(); err == nil {
			dbPath += "/" + ip
			if port, ok := util.ConfigMap["server"].(map[string]any)["port"].(int); ok {
				dbPath += strconv.Itoa(port)
			}
		}
		util.Log.Println("db path:", dbPath)
	}
	return service.Indexer.Init(docNumEstimate, dbType, dbPath)
}

func (service *IndexServiceWorker) Register(servicePort int) error {
	// 向注册中心注册自己
	if servicePort < 1024 {
		return fmt.Errorf("invalid listen port %d, should more than 1024", servicePort)
	}
	selfLocalIp, err := util.GetLocalIP()
	if err != nil {
		panic(err)
	}
	// selfLocalIp := "127.0.0.1" // 仅在本机器模拟分布式部署用
	service.selfAddr = fmt.Sprintf("%s:%d", selfLocalIp, servicePort)

	timeoutCtx, cancel := util.GetDefaultTimeoutContext()
	defer cancel()

	leaseId, err := service.Hub.Register(currentGroup, service.selfAddr, 0)
	if err != nil {
		return err
	}

	if res, err := service.Hub.client.Get(timeoutCtx, ServiceRootPath+indexName+"/"+currentGroup, etcdv3.WithPrefix()); err == nil {
		if len(res.Kvs) == 1 { // 新group中的worker数量为1时才代表有新group加入集群
			service.Hub.addIndexGroup()
		}
	} else {
		util.Log.Printf("failed to get key %s: %v", ServiceRootPath+indexName+currentGroup, err)
	}

	go func() {
		for {
			service.Hub.Register(currentGroup, service.selfAddr, leaseId)
			time.Sleep(time.Duration(service.Hub.heartRate)*time.Second - 100*time.Millisecond)
		}
	}()

	return nil
}

// 向索引中添加文档，如果文档已存在则会覆盖
func (service *IndexServiceWorker) AddDoc(ctx context.Context, doc *types.Document) (*AffectedCount, error) {
	n, err := service.Indexer.AddDoc(*doc)
	return &AffectedCount{uint32(n)}, err
}

// 从索引上删除文档
func (service *IndexServiceWorker) DeleteDoc(ctx context.Context, docId *DocId) (*AffectedCount, error) {
	n := service.Indexer.DeleteDoc(docId.DocId)
	return &AffectedCount{uint32(n)}, nil
}

// 检索，返回文档列表
func (service *IndexServiceWorker) Search(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	documents := service.Indexer.Search(request.Query, request.OnFlag, request.OffFlag, request.OrFlags)
	return &SearchResponse{Documents: documents}, nil
}

func (service *IndexServiceWorker) Count(ctx context.Context, request *CountRequest) (*AffectedCount, error) {
	n := service.Indexer.Count()
	return &AffectedCount{Count: uint32(n)}, nil
}

func (service *IndexServiceWorker) Close() error {
	if service.Hub != nil {
		if err := service.Hub.UnRegister(currentGroup, service.selfAddr); err == nil {
			timeoutCtx, cancel := util.GetDefaultTimeoutContext()
			defer cancel()

			if res, err := service.Hub.client.Get(timeoutCtx, ServiceRootPath+indexName+"/"+currentGroup, etcdv3.WithPrefix()); err == nil {
				if res != nil {
					if len(res.Kvs) <= 0 {
						service.Hub.subIndexGroup()
					}
				}
			}
			service.Hub.Close()
		}
	}
	return service.Indexer.Close()
}
