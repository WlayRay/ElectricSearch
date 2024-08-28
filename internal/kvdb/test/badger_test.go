package kvdbtest

import (
	"testing"

	"MiniES/internal/kvdb"
	"MiniES/util"
)

func TestBadger(t *testing.T) {
	setup = func() {
		var err error
		db, err = kvdb.GetKetValueDB(kvdb.BADGER, util.RootPath+"data/badger_db")
		if err != nil {
			panic(err)
		}
	}

	t.Run("badger_test", testPipeline)
}


