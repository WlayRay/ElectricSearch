package utiltest

import (
	"fmt"
	"github.com/huandu/skiplist"
	"testing"
)

func TestSkipList(t *testing.T) {
	list := skiplist.New(skiplist.Int32)
	list.Set(24, 31) //skiplist是一个按key排序好的map
	list.Set(24, 40) //相同的key, value会覆盖前值
	list.Set(12, 40) //添加元素
	list.Set(18, 3)
	list.Remove(12) //删除元素

	if value, ok := list.GetValue(18); ok {
		fmt.Printf("Key %d Value %d\n", 18, value)
	}

	//遍历。自动按key排好序
	fmt.Println("------------------")
	node := list.Front()
	for node != nil {
		fmt.Printf("Key:%d Value%d\n", node.Key(), node.Value)
		node = node.Next() //迭代器模式
	}
}
