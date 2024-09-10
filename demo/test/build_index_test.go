package test

import (
	"os"
	"testing"

	"github.com/WlayRay/ElectricSearch/demo/common"
	"github.com/WlayRay/ElectricSearch/internal/kvdb"
	"github.com/WlayRay/ElectricSearch/service"
	"github.com/WlayRay/ElectricSearch/util"
)

var (
	dbType  = kvdb.BOLT
	dbPath  = util.RootPath + "data/local_db/video_bolt"
	indexer *service.Indexer
)

func Init() {
	os.Remove(dbPath) //x先删除原有的索引文件
	indexer = new(service.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
}

func TestBuildIndex(t *testing.T) {
	Init()
	defer indexer.Close()
	csvFile := util.RootPath + "data/bilibili_video.csv"
	common.BuildIndexFromCSVFile(csvFile, indexer, 0, 0)
}
