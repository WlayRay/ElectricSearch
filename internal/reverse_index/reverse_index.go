package reverseindex

import (
	"MiniES/types"
)

type IReverseIndex interface {
	// 添加一个Document
	Add(doc *types.Document)

	// 删除Keyword对应的Document
	Delete(IntId uint64, keyword *types.Keyword)

	// 搜索，返回文档ID列表
	Search(q *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []string
}
