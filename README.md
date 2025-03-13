# ElectricSearch索引框架

纯go语言实现的搜索引擎索引框架，支持[单机](service/test/indexer_test.go)部署和[分布式](service/test/distribute_test.go)部署，分布式部署需要etcd作为服务注册中心，可用使用Docker部署etcd

本地一键模拟分布式部署命令（需安装 Docker 和 docker-compose ）： ```docker compose up --build -d```
本地单机部署：
1. 首先需要将[init.yml](./init.yml)的 "mode" 配置改为1
2. 然后运行```go run .\demo\internal\main\```

## 项目架构

```
├── Dockerfile                     # Docker构建文件
├── README.md                      # 项目说明文档
├── docker-compose.yml             # Docker Compose配置文件
├── init.yml                       # 初始化配置文件
├── demo                           # 示例应用
│   ├── handler                    # HTTP接口处理逻辑
│   │   ├── middle_ware.go         # 中间件
│   │   └── search_controller.go   # 搜索控制器
│   ├── infrastructure             # 基础设施层
│   │   ├── bits.go                # Bit操作工具
│   │   ├── build_index.go         # 构建索引逻辑
│   │   ├── model.go               # 数据模型
│   │   ├── video.pb.go            # Protobuf生成的代码
│   │   └── video.proto            # Protobuf定义文件
│   ├── internal                   # 内部实现
│   │   ├── filter                 # 过滤器
│   │   │   └── view_range.go      # 视图范围过滤
│   │   ├── main                   # 主程序入口
│   │   │   ├── index_worker.go    # 索引工作线程
│   │   │   ├── main.go            # 主函数
│   │   │   └── web_server.go      # Web服务器
│   │   ├── recaller               # 召回器
│   │   │   └── keyword.go         # 关键词召回
│   │   └── video_search.go        # 视频搜索逻辑
│   └── test                       # 示例测试
│       ├── build_index_test.go    # 构建索引测试
│       └── search_test.go         # 搜索测试
├── etcd                           # Etcd相关工具
│   ├── etcd_client.go             # Etcd客户端
│   └── etcd_distributed_lock.go   # 分布式锁
├── internal                       # 内部模块
│   ├── kvdb                       # 键值数据库
│   │   ├── badger_db.go           # Badger数据库实现
│   │   ├── bolt_db.go             # Bolt数据库实现
│   │   └── kv_db.go               # 键值数据库接口
│   └── reverse_index              # 倒排索引
│       ├── reverse_index.go       # 倒排索引接口
│       └── skiplist_reverse_index.go # SkipList实现
├── pb                             # Protobuf定义文件
│   ├── doc.proto                  # 文档定义
│   ├── index.proto                # 索引定义
│   └── term_query.proto           # 查询定义
├── service                        # 服务模块
│   ├── IIndexer.go                # 索引接口
│   ├── distribute.go              # 分布式逻辑
│   ├── hub_proxy.go               # Hub代理
│   ├── index.pb.go                # Protobuf生成的代码
│   ├── index_service.go           # 索引服务
│   ├── indexer.go                 # 索引器实现
│   ├── load_balance.go            # 负载均衡
│   └── service_hub.go             # 服务Hub
├── types                          # 类型定义
│   ├── doc.go                     # 文档类型
│   ├── doc.pb.go                  # Protobuf生成的代码
│   ├── term_query.go              # 查询类型
│   └── term_query.pb.go           # Protobuf生成的代码
└── util                           # 工具模块
    ├── common_helper.go           # 通用辅助工具
    ├── concurrent_hash_map.go     # 并发哈希表
    ├── config.go                  # 配置工具
    ├── context_tool.go            # 上下文工具
    ├── log.go                     # 日志工具
    ├── net.go                     # 网络工具
    └── snowflake.go               # 雪花算法
```

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
