package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/WlayRay/ElectricSearch/internal/kvdb"
	"github.com/WlayRay/ElectricSearch/types"
	"github.com/WlayRay/ElectricSearch/util"
)

var indexService string

func init() {
	var (
		ok   bool
		mode int
	)
	mode = util.ConfigMap["mode"].(int)
	if mode < 2 || mode > 3 {
		indexService = "standalone"
		return
	}
	distributedConfig, ok := util.ConfigMap["distributed"].(map[string]any)
	if !ok {
		panic("distributed configuration not found!")
	}

	if indexService, ok = distributedConfig["index-service"].(string); !ok {
		panic("indexService not found in config")
	}
}

// IndexServiceWorker是一个grpc服务，用于索引文档
type IndexServiceWorker struct {
	Indexer  *Indexer
	hub      *ServiceHub
	selfAddr string
}

func (service *IndexServiceWorker) Init(workerIndex int) error {
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
		util.Log.Println("db path:", dbPath)
		dbPath += "_" + strconv.Itoa(workerIndex)
	}
	return service.Indexer.Init(docNumEstimate, dbType, dbPath)
}

func (service *IndexServiceWorker) Register(etcdEndpoint []string, servicePort, heartRate int) error {
	// 向注册中心注册自己
	if len(etcdEndpoint) > 0 {
		if servicePort < 1024 {
			return fmt.Errorf("invalid listen port %d, should more than 1024", servicePort)
		}
		/*selfLocalIp, err := util.GetLocalIP()
		if err != nil {
			panic(err)
		}*/
		selfLocalIp := "127.0.0.1" // 仅在本机器模拟分布式部署用
		service.selfAddr = fmt.Sprintf("%s:%d", selfLocalIp, servicePort)
		hub := GetServiceHub(etcdEndpoint, int64(heartRate))
		leaseId, err := hub.Register(indexService, service.selfAddr, 0)
		if err != nil {
			panic(err)
		}
		service.hub = hub
		go func() {
			for {
				hub.Register(indexService, service.selfAddr, leaseId)
				time.Sleep(time.Duration(heartRate)*time.Second - 100*time.Millisecond)
			}
		}()
	}
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
	if service.hub != nil {
		service.hub.UnRegister(indexService, service.selfAddr)
		service.hub.Close()
	}
	return service.Indexer.Close()
}
