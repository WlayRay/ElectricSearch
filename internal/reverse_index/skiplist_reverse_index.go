package reverseindex

import (
	"runtime"
	"sync"

	"github.com/WlayRay/ElectricSearch/v1.0.0/types"
	"github.com/WlayRay/ElectricSearch/v1.0.0/util"

	"github.com/dgryski/go-farm"
	"github.com/huandu/skiplist"
)

// 倒排索引整体上是个Map，key是关键词KeyWord，value是倒排索引的列表（跳表实现）
type SkipListReverseIndex struct {
	table *util.ConcurrentHashMap
	locks []sync.RWMutex
}

// DocNumEstimate 预估的文档数量
func NewSkipListReverseIndex(DocNumEstimate int) *SkipListReverseIndex {
	return &SkipListReverseIndex{
		table: util.NewConcurrentHashMap(runtime.NumCPU(), DocNumEstimate),
		locks: make([]sync.RWMutex, 1000),
	}
}

func (idx SkipListReverseIndex) getLock(key string) *sync.RWMutex {
	n := int(farm.Hash32WithSeed([]byte(key), 0))
	return &idx.locks[n%len(idx.locks)]
}

type SkipListValue struct {
	Id          string
	BitsFeature uint64
}

func (idx SkipListReverseIndex) Add(doc types.Document) {
	for _, keyword := range doc.Keywords {
		key := keyword.ToString()
		lock := idx.getLock(key)
		lock.Lock()

		skipListValue := SkipListValue{
			Id:          doc.Id,
			BitsFeature: doc.BitsFeature,
		}
		if value, exists := idx.table.Get(key); exists {
			list := value.(*skiplist.SkipList)
			list.Set(doc.IntId, skipListValue)
		} else {
			list := skiplist.New(skiplist.Uint64)
			list.Set(doc.IntId, skipListValue)
			idx.table.Set(key, list)
		}

		lock.Unlock()
	}
}

func (idx SkipListReverseIndex) Delete(IntId uint64, keyword *types.Keyword) {
	key := keyword.ToString()
	lock := idx.getLock(key)
	lock.Lock()

	if value, exists := idx.table.Get(key); exists {
		list := value.(*skiplist.SkipList)
		list.Remove(IntId)
	}

	lock.Unlock()
}

func IntersectionOfSkipList(lists ...*skiplist.SkipList) (res *skiplist.SkipList) {
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		return lists[0]
	}

	res = skiplist.New(skiplist.Uint64)
	nodes := make([]*skiplist.Element, len(lists))
	for i, list := range lists {
		if list == nil || list.Len() == 0 {
			return nil
		}
		nodes[i] = list.Front()
	}

	for {
		var maxList map[int]struct{} // 用于存储当前值最大的节点（可能有多个，所以用map来模仿集合）
		var maxValue uint64 = 0
		for i, node := range nodes {
			if node.Key().(uint64) > maxValue {
				maxValue = node.Key().(uint64)
				maxList = map[int]struct{}{i: {}}
			} else if node.Key().(uint64) == maxValue {
				maxList[i] = struct{}{}
			}
		}
		if len(maxList) == len(lists) { // 所有node节点都指向了最大值，可以添加到交集
			res.Set(nodes[0].Key(), nodes[0].Value)
			for i, node := range nodes { // 所有node节点往后移
				nodes[i] = node.Next()
				if nodes[i] == nil {
					return
				}
			}
		} else {
			for i, node := range nodes {
				if _, exists := maxList[i]; !exists {
					nodes[i] = node.Next()
					if nodes[i] == nil {
						return
					}
				}
			}
		}
	}
}

func UnionOfSkipList(lists ...*skiplist.SkipList) (res *skiplist.SkipList) {
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		return lists[0]
	}

	res = skiplist.New(skiplist.Uint64)
	keySet := make(map[any]struct{}, 1000)
	for _, list := range lists {
		if list == nil {
			continue
		}
		node := list.Front()
		for node != nil {
			if _, exist := keySet[node.Key()]; !exist {
				res.Set(node.Key(), node.Value)
				keySet[node.Key()] = struct{}{}
			}
			node = node.Next()
		}
	}
	return
}

func (idx SkipListReverseIndex) FilterByBits(bits uint64, onFlag uint64, offFlag uint64, orFlags []uint64) bool {
	if bits&onFlag != onFlag {
		return false
	}
	if bits&offFlag != 0 {
		return false
	}
	//多个orFlags必须全部命中
	for _, orFlag := range orFlags {
		if orFlag > 0 && bits&orFlag <= 0 { //单个orFlag只有一个bit命中即可
			return false
		}
	}
	return true
}

func (idx SkipListReverseIndex) search(tq *types.TermQuery, onFlag, offFlag uint64, orFlags []uint64) *skiplist.SkipList {
	if tq.Keyword != nil {
		keyword := tq.Keyword.ToString()
		if value, exists := idx.table.Get(keyword); exists {
			result := skiplist.New(skiplist.Uint64)
			list := value.(*skiplist.SkipList)
			for node := list.Front(); node != nil; node = node.Next() {
				intId := node.Key().(uint64)
				skiplistValue := node.Value.(SkipListValue)
				if idx.FilterByBits(skiplistValue.BitsFeature, onFlag, offFlag, orFlags) {
					result.Set(intId, skiplistValue)
				}
			}
			return result
		}
	} else if len(tq.Must) > 0 {
		results := make([]*skiplist.SkipList, 0, len(tq.Must))
		for _, subQuery := range tq.Must {
			results = append(results, idx.search(subQuery, onFlag, offFlag, orFlags))
		}
		return IntersectionOfSkipList(results...)
	} else if len(tq.Should) > 0 {
		results := make([]*skiplist.SkipList, 0, len(tq.Should))
		for _, subQuery := range tq.Should {
			results = append(results, idx.search(subQuery, onFlag, offFlag, orFlags))
		}
		return UnionOfSkipList(results...)
	}
	return nil
}

func (idx SkipListReverseIndex) Search(tq *types.TermQuery, onFlag, offFlag uint64, orFlags []uint64) []string {
	skp := idx.search(tq, onFlag, offFlag, orFlags)
	if skp == nil {
		return nil
	}
	result := make([]string, 0, skp.Len())
	for node := skp.Front(); node != nil; node = node.Next() {
		skiplistValue := node.Value.(SkipListValue)
		result = append(result, skiplistValue.Id)
	}
	return result
}
