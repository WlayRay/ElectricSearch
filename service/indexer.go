package service

import (
	"MiniES/internal/kvdb"
	"MiniES/internal/reverse_index"
)

type Indexer struct {
	fowrdIndex   kvdb.IKeyValueDB
	reverseIndex reverseindex.IReverseIndex
	maxIntId     uint64
}

func (indexer *Indexer) Init(DocNumEstimate int, dbtype int, Data string) error {
	db, err := kvdb.GetKetValueDB(dbtype, Data)
	if err != nil {
		return err
	}
	indexer.fowrdIndex = db
	indexer.reverseIndex = reverseindex.NewSkipListReverseIndex(DocNumEstimate)
	return nil
}
