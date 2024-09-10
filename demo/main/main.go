package main

import (
	"flag"
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
	port                = flag.Int("port", 0, "server的工作端口")
)

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
	} else {
		totalWorkers, _ = strconv.Atoi(v)
		if totalWorkers < 1 {
			panic("totalWorkers invalid!")
		}
	}

	//当前worker标号（从0开始）
	if v, ok := util.Configurations["worker-index"]; !ok {
		panic("workerIndex not found in configurations!")
	} else {
		workerIndex, _ = strconv.Atoi(v)
		if workerIndex < 0 || workerIndex > totalWorkers {
			panic("workerIndex invalid!")
		}
	}

	//初始化文档所用CSV文件的路径
	if v, ok := util.Configurations["csv-file-path"]; !ok {
		panic("csvFilePath not found in configurations!")
	} else {
		csvFilePath = util.RootPath + strings.Replace(v, "\"", "", -1)
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
}

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
	engine.Run("127.0.0.1" + strconv.Itoa(*port))
}

func main() {
	flag.Parse()

	switch mode {
	case 1, 3:
		WebServerMain(mode)
		startGin()
	case 2:

	}
}

// go run ./demo/main -mode=1 -index=true -port=5678 -dbPath=data/local_db/video_bolt