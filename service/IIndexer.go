package service

import (
	"github.com/WlayRay/ElectricSearch/types"
)

type IIndexer interface {
	AddDoc(doc types.Document) (int, error)
	DeleteDoc(docId string) int
	Search(querys *types.TermQuery, onFlag, offFlag uint64, orFlags []uint64) []*types.Document
	Count() int
	Close() error
}
