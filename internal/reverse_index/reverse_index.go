package reverseindex

import (
	"github.com/WlayRay/ElectricSearch/v1.0.0/types"
)

type IReverseIndex interface {
	// TODO: 将接收的Document转换成指针类型，并修改其他调用改接口的地方
	// 添加一个Document
	Add(doc types.Document)

	// 删除Keyword对应的Document
	Delete(IntId uint64, keyword *types.Keyword)

	// 搜索，返回文档ID列表
	Search(q *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []string
}
