# ElectricSearch索引框架

纯go语言实现的搜索引擎索引框架，支持[单机](service/test/indexer_test.go)部署，[分布式](service/test/distribute_test.go)部署，分布式部署需要etcd作为服务注册中心，可用使用Docker部署etcd

## 项目架构

### 倒排索引

<img src="asset/倒排索引.png" width="700"/>    

- 倒排索引的整体架构由的[ConcurrentHashMap](util/concurrent_hash_map.go)配合SkipList实现，ConcurrentHashMap较支持并发读写，且较sync.map性能更好。
- IntId是使用[雪花算法](util/snowflake.go)给document生成的自增id，用于SkipList的排序。
- Id是document在业务侧的ID。
- BitsFeature是uint64，可以把document的属性编码成bit流，遍历倒排索引的同时完成部分筛选功能。

### 正排索引

支持badger和bolt两种数据库存储将document存储在磁盘。

### 分布式索引

如果document数量过大，单机容不下时，可以将document分散存储在多台服务器上。各索引服务器之间通过grpc通信，通过etcd实现服务注册与发现。

<img src="asset/分布式索引架构.png" width="700"/>  

- 由多个Group垂直切分整个倒排索引，每个Group内有多台worker做冗余备份
- worker(Group)之间通过etcd感知彼此的存在，并使用grpc通信
- 在[hub_proxy](service/hub_proxy.go)中使用代理模式缓存etcd的中的服务地址，并对etcd做限流保护

## 示例教程

在代码中import依赖包并执行go mod tidy

``` go
import (
    "github.com/WlayRay/ElectricSearch/service"
    "github.com/WlayRay/ElectricSearch/types"
)
```

定义业务文档结构体，这里假设业务文档为Book

```go
type Book struct {
    ISBN    string
    Title   string
    Author  string
    Price   float64
    Content string
}

// 业务侧自行实现doc的序列化和反序列化
func (book *Book) Serialize() []byte {
    // 可用pb、json、gob等方式将Book序列化成字节流
}

func DeserializeBook(v []byte) *Book {
    // 反序列化
}
```

初始化索引

``` go
es := new(service.Indexer)
// 指定预估的文档容量、正排索引使用的存储引擎，以及文档数据存储的位置
if err := es.Init(100, 0, "./data/local_db/book_bolt"); err != nil { // 这里0指代使用boltdb
    fmt.Println(err)
    t.Fail()
    return
}
defer es.Close()
```

初始化文档

```go
book1 := Book{
    ISBN:    "315246546",
    Title:   "计算机系列丛书",
    Author:  "张三",
    Price:   59.0,
    Content: "冰雪奇缘2 中文版电影原声带 (Frozen 2 (Mandarin Original Motion Picture",
}

doc1 := types.Document{
        Id:          book1.ISBN,
        BitsFeature: 0b10101, //二进制
        Keywords:    []*types.Keyword{{Field: "content", Word: "机器学习"}, {Field: "content", Word: "神经网络"}, {Field: "title", Word: book1.Title}},
        Bytes:       book1.Serialize(), //写入索引时需要自行序列化
    }
```

```go
// 添加删除文档
es.AddDoc(doc1) 
es.DeleteDoc(doc1.Id)

// 构造搜索表达式搜索文档
// 支持任意复杂的And和Or的组合。And要求同时命中，Or只要求命中一个
q1 := types.NewTermQuery("title", "生命起源")
q2 := types.NewTermQuery("content", "文物")
q3 := types.NewTermQuery("title", "中国历史")
q4 := types.NewTermQuery("content", "文物")
q5 := types.NewTermQuery("content", "唐朝")
q6 := q1.And(q2)
q7 := q3.And(q4).And(q5)
q8 := q6.Or(q7)

var onFlag uint64 = 0b10000 //要求doc.BitsFeature的对应位必须都是1
var offFlag uint64 = 0b01000 //要求doc.BitsFeature的对应位必须都是0
orFlags := []uint64{uint64(0b00010), uint64(0b00101)} //要求doc.BitsFeature的对应位至少有一个是1
docs := es.Search(q8, onFlag, offFlag, orFlags) //检索
```
