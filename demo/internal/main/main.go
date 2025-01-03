package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/WlayRay/ElectricSearch/demo/handler"
	"github.com/WlayRay/ElectricSearch/internal/kvdb"
	"github.com/WlayRay/ElectricSearch/util"
	"github.com/gin-gonic/gin"
)

var (
	mode                int
	documentEstimateNum int
	dbType              int
	dbPath              string
	rebuildIndex        bool
	totalWorkers        int
	workerIndex         int
	csvFilePath         string
	port                int
	etcdEndpoints       []string
	heartRate           int
)

func startGin() {
	engine := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	engine.Use(handler.GetUserInfo)
	// categoriesBits := [...]string{"鬼畜", "记录", "科技", "美食", "音乐", "影视", "娱乐", "游戏", "综艺", "知识", "资讯", "番剧", "舞蹈", "游记"}

	engine.POST("/search", handler.SearchAll)
	engine.POST("/up_search", handler.SearchByAuthor)
	
	if err := engine.Run("127.0.0.1:" + strconv.Itoa(port)); err != nil {
		util.Log.Println("Server failed to start:", err)
		return
	}
	util.Log.Println("Server started succeed at port:", port)
}

func main() {
	switch mode {
	case 1, 3:
		WebServerMain(mode)
		startGin()
	case 2:
		GrpcIndexerMain()
	}
	// 模式1为单机部署，模式2为启动分布式部署下的每个索引服务节点，相当于一个grpc server
	// 模式3为启动分布式部署下的etcd代理（Sentinel），后续的添加、搜索、删除文档等都通过代理操作
	// 在分布式部署时，需要先通过模式2启动多个索引服务节点，然后再通过模式3启动etcd代理和web server
}

//go run ./demo/main

func init() {
	// 配置文件校验
	// 初始化部署模式
	if v, ok := util.ConfigMap["mode"]; !ok {
		panic("mode not found in ConfigMap!")
	} else {
		mode, _ = strconv.Atoi(fmt.Sprintf("%v", v))
		if mode < 1 || mode > 3 {
			panic("mode invalid!")
		}
	}

	// 读取 server 配置
	serverConfig, ok := util.ConfigMap["server"].(map[string]any)
	if !ok {
		panic("server configuration not found!")
	}

	if v, ok := serverConfig["port"]; !ok {
		panic("port not found in ConfigMap!")
	} else {
		port, _ = strconv.Atoi(fmt.Sprintf("%v", v))
	}

	if v, ok := serverConfig["rebuild_index"]; !ok {
		panic("rebuildIndex not found in ConfigMap!")
	} else {
		switch v := v.(type) {
		case bool:
			rebuildIndex = v
		case string:
			rebuildIndex = v == "true"
		default:
			panic("rebuildIndex invalid!")
		}
	}

	// 读取 distributed 配置
	distributedConfig, ok := util.ConfigMap["distributed"].(map[string]any)
	if mode == 2 && !ok {
		panic("distributed configuration not found!")
	}

	if mode == 2 {
		if v, ok := distributedConfig["total_workers"]; !ok {
			panic("totalWorkers not found in ConfigMap!")
		} else {
			totalWorkers, _ = strconv.Atoi(fmt.Sprintf("%v", v))
			if totalWorkers < 1 {
				panic("totalWorkers invalid!")
			}
		}

		if v, ok := distributedConfig["worker_index"]; !ok {
			panic("workerIndex not found in ConfigMap!")
		} else {
			workerIndex, _ = strconv.Atoi(fmt.Sprintf("%v", v))
			if workerIndex < 0 || workerIndex >= totalWorkers {
				panic("workerIndex invalid!")
			}
		}

		if v, ok := distributedConfig["heart_rate"]; ok {
			heartRate, _ = strconv.Atoi(fmt.Sprintf("%v", v))
		}
	}

	// 读取 index 配置
	indexConfig, ok := util.ConfigMap["index"].(map[string]any)
	if !ok {
		panic("index configuration not found!")
	}

	if v, ok := indexConfig["csv_file"]; !ok {
		panic("csvFilePath not found in ConfigMap!")
	} else {
		csvFilePath = util.RootPath + strings.Replace(fmt.Sprintf("%v", v), "\"", "", -1)
		util.Log.Printf("csvFilePath: %s", csvFilePath)
	}

	if v, ok := indexConfig["db_path"]; !ok {
		panic("dbPath not found in ConfigMap!")
	} else {
		dbPath = util.RootPath + strings.Replace(fmt.Sprintf("%v", v), "\"", "", -1)
	}

	if v, ok := indexConfig["db_type"]; !ok {
		panic("dbType not found in ConfigMap!")
	} else {
		switch fmt.Sprintf("%v", v) {
		case "badger":
			dbType = kvdb.BADGER
			dbPath += "badger_db"
		default:
			dbType = kvdb.BOLT
			dbPath += "bolt_db/bolt"
		}
	}

	if v, ok := indexConfig["document_estimate_num"]; !ok {
		panic("documentEstimateNum not found in ConfigMap!")
	} else {
		documentEstimateNum, _ = strconv.Atoi(fmt.Sprintf("%v", v))
	}

	// 读取 etcd 配置
	etcdConfig, ok := util.ConfigMap["etcd"].(map[string]any)
	if !ok {
		panic("etcd configuration not found!")
	}

	if v, ok := etcdConfig["servers"]; ok {
		servers, ok := v.([]any)
		if !ok {
			panic("etcd servers configuration invalid!")
		}
		for _, server := range servers {
			endpoint := strings.TrimSpace(fmt.Sprintf("%v", server))
			endpoint = strings.Replace(endpoint, "\"", "", -1)
			etcdEndpoints = append(etcdEndpoints, endpoint)
		}
		util.Log.Printf("etcdEndpoints: %v", etcdEndpoints)
	}
}
