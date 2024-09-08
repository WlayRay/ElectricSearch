package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/WlayRay/ElectricSearch/demo/handler"
	"github.com/WlayRay/ElectricSearch/util"
	"github.com/gin-gonic/gin"
)

var (
	mode         int
	rebuildIndex bool
	port         int
	totalWorkers int
	workerIndex  int
	csvFilePath  string
)

func init() {
	if v, ok := util.Configurations["mode"]; !ok {
		panic("mode not found in configurations!")
	} else {
		mode, _ = strconv.Atoi(v)
		if mode < 1 || mode > 3 {
			panic("mode invalid!")
		}
	}

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

	if v, ok := util.Configurations["port"]; !ok {
		panic("port not found in configurations!")
	} else {
		port, _ = strconv.Atoi(v)
		if port < 1024 || port > 65535 {
			panic("port invalid!")
		}
	}

	if v, ok := util.Configurations["total-workers"]; !ok {
		panic("totalWorkers not found in configurations!")
	} else {
		totalWorkers, _ = strconv.Atoi(v)
		if totalWorkers < 1 {
			panic("totalWorkers invalid!")
		}
	}

	if v, ok := util.Configurations["worker-index"]; !ok {
		panic("workerIndex not found in configurations!")
	} else {
		workerIndex, _ = strconv.Atoi(v)
		if workerIndex < 0 || workerIndex > totalWorkers {
			panic("workerIndex invalid!")
		}
	}

	if v, ok := util.Configurations["csv-file-path"]; !ok {
		panic("csvFilePath not found in configurations!")
	} else {
		csvFilePath = util.RootPath + strings.Replace(v, "\"", "", -1)
	}
}

func startGin() {
	engine := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	engine.Static("js", "demo/frontend/js")
	engine.Static("css", "demo/frontend/css")
	engine.Static("img", "demo/frontend/img")
	engine.LoadHTMLFiles("demo/frontend/index.html")

	engine.Use(handler.GetUserInfo)
	categoriesBits := [...]string{"鬼畜", "记录", "科技", "美食", "音乐", "影视", "娱乐", "游戏", "综艺", "知识", "资讯", "番剧", "舞蹈", "游记"}
	engine.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "search.html", categoriesBits)
	})
	
	engine.POST("/search", handler.SearchAll)
}
