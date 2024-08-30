package service

import (
	"MiniES/internal/kvdb"
	"MiniES/internal/reverse_index"
	"MiniES/types"
	"MiniES/util"
	"bytes"
	"encoding/gob"
	"strings"
	"sync/atomic"
)

// 外观模式，把正排和倒排索引2个子系统封装在一起
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

func (indexer *Indexer) Close() error {
	return indexer.fowrdIndex.Close()
}

// 倒排索引存储在内存中，系统重启时从正派索引里加载数据
func (indexer *Indexer) LoadFromIndexFile() int {
	n := indexer.fowrdIndex.IterDB(func(k, v []byte) error {
		reader := bytes.NewReader(v)
		decoder := gob.NewDecoder(reader)
		var doc types.Document
		err := decoder.Decode(&doc)
		if err != nil {
			util.Log.Printf("Decode error: %v", err)
			return nil
		}
		indexer.reverseIndex.Add(doc)
		return err
	})
	util.Log.Printf("Load %d datas from forward index: %s", n, indexer.fowrdIndex.GetDbPath())
	return int(n)
}

// 向索引中添加文档，如果文档已存在则会覆盖
func (indexer *Indexer) AddDoc(doc types.Document) (int, error) {
	docId := strings.TrimSpace(doc.Id)
	if len(docId) == 0 {
		return 0, nil
	}

	// 先从正排和倒排索引上删除文档
	indexer.DeleteDoc(docId)

	// TODO: 优化，使用雪花算法生成IntId
	// doc.IntId = util.Snowflake.GetId()
	doc.IntId = atomic.AddUint64(&indexer.maxIntId, 1)
	// 写入正排索引
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value)
	if err := encoder.Encode(doc); err == nil {
		indexer.fowrdIndex.Set([]byte(docId), value.Bytes())
	} else {
		return 0, err
	}

	// 写入倒排序索引
	indexer.reverseIndex.Add(doc)
	return 1, nil
}

func (indexer *Indexer) DeleteDoc(docId string) int {
	n := 0
	forwardKey := []byte(docId)
	docBytes, err := indexer.fowrdIndex.Get(forwardKey) //先读正排索引，得到IntId和Keywords
	if err == nil {
		if len(docBytes) > 0 {
			n = 1
			reader := bytes.NewReader(docBytes)
			decoder := gob.NewDecoder(reader)
			var doc types.Document
			err := decoder.Decode(&doc)
			if err != nil {
				util.Log.Printf("Decode error: %v", err)
			} else {
				// 从倒排索引上删除
				for _, keyword := range doc.Keywords {
					indexer.reverseIndex.Delete(doc.IntId, keyword)
				}
			}
		}
	} else {
		util.Log.Printf("DeleteDoc error: %v", err)
	}
	// 从正排索引上删除
	indexer.fowrdIndex.Delete(forwardKey)
	return n
}

// 检索，返回文档列表
func (indexer *Indexer) Search(querys *types.TermQuery, onFlag, offFlag uint64, orFlags []uint64) []*types.Document {
	docIds := indexer.reverseIndex.Search(querys, onFlag, offFlag, orFlags)
	if len(docIds) == 0 {
		return nil
	}

	keys := make([][]byte, 0, len(docIds))
	for _, docId := range docIds {
		keys = append(keys, []byte(docId))
	}
	docs, err := indexer.fowrdIndex.BatchGet(keys)
	if err != nil {
		util.Log.Printf("Search from forward index error: %v", err)
	}

	result := make([]*types.Document, 0, len(docs))
	reader := bytes.NewReader(nil)
	var doc types.Document
	for _, docBytes := range docs {
		if len(docBytes) > 0 {
			reader.Reset(docBytes)
			decoder := gob.NewDecoder(reader)
			err := decoder.Decode(&doc)
			if err != nil {
				util.Log.Printf("Decode error: %v", err)
				continue
			} else {
				result = append(result, &doc)
			}
		}
	}
	return result
}

func (Indexer *Indexer) Count() int {
	n := 0
	Indexer.fowrdIndex.IterKey(func(k []byte) error {
		n++
		return nil
	})
	return n
}
