package kvdbtest

import (
	"testing"

	"ElectricSearch/internal/kvdb"
	"ElectricSearch/util"
)

func TestBolt(t *testing.T) {
	setup = func() {
		var err error
		db, err = kvdb.GetKetValueDB(kvdb.BOLT, util.RootPath+"data/bolt_db") //使用工厂模式
		if err != nil {
			panic(err)
		}
	}

	t.Run("bolt_test", testPipeline)
}
