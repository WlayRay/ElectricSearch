# ElectricSearch索引框架

纯go语言实现的搜索引擎索引框架，支持[单机](service/test/indexer_test.go)部署和[分布式](service/test/distribute_test.go)部署，分布式部署需要etcd作为服务注册中心，可用使用Docker部署etcd

本地一键模拟分布式部署命令（需安装 Docker 和 docker-compose ）： ```docker compose up --build -d```
本地单机部署：
1. 首先需要将[init.yml](./init.yml)的 "mode" 配置改为1
2. 然后运行```go run .\demo\internal\main\```

## 项目架构

### 倒排索引

<img src="asset/倒排索引.png" width="700"/>    

- 倒排索引的整体架构由的[ConcurrentHashMap](util/concurrent_hash_map.go)配合SkipList实现，ConcurrentHashMap较支持并发读写，且较sync.map性能更好。
- IntId是使用[雪花算法](util/snowflake.go)给document生成的自增id，用于SkipList的排序。
- Id是document在业务侧的ID。
- BitsFeature是uint64，可以把document的属性编码成bit流，遍历倒排索引的同时完成部分筛选功能。

### 正排索引

支持badger和bolt两种存储引擎将documents存储在磁盘。

### 分布式索引

如果document数量过大，单机容不下时，可以将document分散存储在多台服务器上。各索引服务器之间通过grpc通信，通过etcd实现服务注册与发现。

<img src="asset/分布式索引架构.png" width="700"/>  

- 由多个Group垂直切分整个倒排索引，每个Group内有多台worker做冗余备份
- worker(Group)之间通过etcd感知彼此的存在，并使用grpc通信
- 在[hub_proxy](service/hub_proxy.go)中使用代理模式缓存etcd的中的服务地址，并对etcd做限流保护

## 示例教程

可以参考项目中的[demo](demo)目录，该目录为此框架的示例demo，可搜索CSV文件中的BiliBili视频信息，并提供http接口查询。

也可以查看[service/test/](service/test)这个目录下的测试函数来学习如何使用此框架。

下面大致介绍demo中的代码流程：

首先在代码中import依赖包并执行go mod tidy

``` go
import (
    "github.com/WlayRay/ElectricSearch/service"
    "github.com/WlayRay/ElectricSearch/types"
    //其他属于此项目的包
)
```

定义业务文档结构体，这里用protobuf定义一个[BiliBili视频信息的结构体](demo/infrastructure/video.proto)

```
message BiliBiliVideo {
  string Id = 1;
  string Title = 2;
  int64 PostTime = 3;
  string Author = 4;
  int32 ViewCount = 5;
  int32 LikeCount = 6;
  int32 CoinCount = 7;
  int32 FavoriteCount = 8;
  int32 shareCount = 9;
  repeated string Keywords = 10;
}
```

初始化索引服务，在[demo/internal/main/web_server.go](demo/internal/main/web_server.go)和[demo/internal/main/index_worker.go](demo/internal/main/index_worker.go)这两个文件中

``` go
// 单机部署
standaloneIndexer := new(service.Indexer)
if err := standaloneIndexer.Init(documentEstimateNum, dbType, dbPath); err != nil {
    panic(err)
}

// 分布式部署
indexService = new(service.IndexServiceWorker)
indexService.Init(currentGroup)
```

分布式部署还要在[demo/internal/main/web_server.go](demo/internal/main/web_server.go)中创建代理

``` go
handler.Indexer = service.NewSentinel(etcdEndpoints)
```

从CSV文件中初始化文档，代码在[demo/infrastructure/build_index.go](demo/infrastructure/build_index.go)中

```go
if rebuildIndex {
    infrastructure.BuildIndexFromCSVFile(csvFilePath, standaloneIndexer, 0, 0)
} else {
    standaloneIndexer.LoadFromIndexFile()
}
```

查询视频

```go
keywords := request.Keywords
query := new(types.TermQuery)
if len(keywords) > 0 {
    for _, keyword := range keywords {
    query = query.And(types.NewTermQuery("content", keyword))
    }
}
if len(request.Author) > 0 {
    query = query.And(types.NewTermQuery("author", strings.ToLower(request.Author)))
}
orFlags := []uint64{(infrastructure.GetCategoriesBits(request.Categories))}
docs := indexer.Search(query, 0, 0, orFlags)

videos := make([]*infrastructure.BiliBiliVideo, 0, len(docs))
for _, doc := range docs {
    var video infrastructure.BiliBiliVideo
    if err := proto.Unmarshal(doc.Bytes, &video); err == nil {
        videos = append(videos, &video)
    }
}
return videos
```
