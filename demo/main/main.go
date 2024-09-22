package main

import (
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
	hearRate            int
)

func startGin() {
	engine := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	// engine.Static("js", "demo/frontend/js")
	// engine.Static("css", "demo/frontend/css")
	// engine.Static("img", "demo/frontend/img")
	// engine.LoadHTMLFiles("demo/frontend/search.html")

	engine.Use(handler.GetUserInfo)
	// categoriesBits := [...]string{"鬼畜", "记录", "科技", "美食", "音乐", "影视", "娱乐", "游戏", "综艺", "知识", "资讯", "番剧", "舞蹈", "游记"}
	// engine.GET("/", func(ctx *gin.Context) {
	// 	ctx.HTML(http.StatusOK, "search.html", categoriesBits)
	// })

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
	// 初始化部署模式
	if v, ok := util.Configurations["mode"]; !ok {
		panic("mode not found in configurations!")
	} else {
		mode, _ = strconv.Atoi(v)
		if mode < 1 || mode > 3 {
			panic("mode invalid!")
		}
	}

	if v, ok := util.Configurations["port"]; !ok {
		panic("port not found in configurations!")
	} else {
		port, _ = strconv.Atoi(v)
	}

	//是否重建索引
	if v, ok := util.Configurations["rebuild-index"]; !ok {
		panic("rebuildIndex not found in configurations!")
	} else {
		switch v {
		case "true":
			rebuildIndex = true
		case "false":
			rebuildIndex = false
		default:
			panic("rebuildIndex invalid!")
		}
	}

	//分布式模式下的worker总数
	if v, ok := util.Configurations["total-workers"]; !ok {
		panic("totalWorkers not found in configurations!")
	} else if mode == 2 {
		totalWorkers, _ = strconv.Atoi(v)
		if totalWorkers < 1 {
			panic("totalWorkers invalid!")
		}
	}

	//当前worker标号（从0开始）
	if v, ok := util.Configurations["worker-index"]; !ok {
		panic("workerIndex not found in configurations!")
	} else if mode == 2 {
		workerIndex, _ = strconv.Atoi(v)
		if workerIndex < 0 || workerIndex > totalWorkers {
			panic("workerIndex invalid!")
		}
	}

	//初始化文档所用CSV文件的路径
	if v, ok := util.Configurations["csv-file"]; !ok {
		panic("csvFilePath not found in configurations!")
	} else {
		csvFilePath = util.RootPath + strings.Replace(v, "\"", "", -1)
		util.Log.Printf("csvFilePath: %s", csvFilePath)
	}

	//正排索引存储文件的保存路径
	if v, ok := util.Configurations["db-path"]; !ok {
		panic("dbPath not found in configurations!")
	} else {
		dbPath = util.RootPath + strings.Replace(v, "\"", "", -1)
	}

	//正排索引使用哪种引擎
	if v, ok := util.Configurations["db-type"]; !ok {
		panic("dpType not found int configurations!")
	} else {
		switch v {
		case "badger":
			dbType = kvdb.BADGER
			dbPath += "badger_db"
		default:
			dbType = kvdb.BOLT
			dbPath += "bolt_db/bolt"
		}
	}

	if v, ok := util.Configurations["document-estimate-num"]; !ok {
		panic("documentEstimateNum not found in configurations!")
	} else {
		documentEstimateNum, _ = strconv.Atoi(v)
	}

	//etcd集群endpoints
	if v, ok := util.Configurations["etcd-servers"]; ok {
		addrs := strings.Trim(v, "[]")
		endpoints := strings.Split(addrs, ",")
		for _, endpoint := range endpoints {
			endpoint = strings.TrimSpace(endpoint)
			endpoint = strings.Replace(endpoint, "\"", "", -1)
			etcdEndpoints = append(etcdEndpoints, endpoint)
		}
		util.Log.Printf("etcdEndpoints: %v", etcdEndpoints)
	}

	//分布式部署下，每个服务的心跳周期
	if v, ok := util.Configurations["heart-rate"]; ok {
		hearRate, _ = strconv.Atoi(v)
	}
}
