package servicetest

import (
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/WlayRay/ElectricSearch/service"
	"github.com/WlayRay/ElectricSearch/types"

	"google.golang.org/grpc"
)

var (
	workPorts   = []int{49658, 52791, 50660} //在一台机器上启多个worker，实际中是一台机器上启一个worker
	etcdServers = []string{"127.0.0.1:2379"}
	workers     []*service.IndexServiceWorker
)

func StartWorkers() {
	workers = make([]*service.IndexServiceWorker, 0, len(workPorts))
	for i, port := range workPorts {
		// 监听本地端口
		lis, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err != nil {
			panic(err)
		}

		server := grpc.NewServer()
		indexServiceWorker := new(service.IndexServiceWorker)
		indexServiceWorker.Init(i)
		indexServiceWorker.Indexer.LoadFromIndexFile() //从文件中加载索引数据
		// 注册服务的具体实现
		service.RegisterIndexServiceServer(server, indexServiceWorker)
		indexServiceWorker.Register(etcdServers, port, 20)
		go func(port int) {
			// 启动服务
			fmt.Printf("start grpc server on port %d\n", port)
			err = server.Serve(lis) //Serve会一直阻塞，所以放到一个协程里异步执行
			if err != nil {
				indexServiceWorker.Close()
				fmt.Printf("start grpc server on port %d failed: %s\n", port, err)
			} else {
				workers = append(workers, indexServiceWorker)
			}
		}(port)
	}
}

func StopWorkers() {
	for _, worker := range workers {
		worker.Close()
	}
}

func TestIndexCluster(t *testing.T) {
	StartWorkers()
	time.Sleep(3 * time.Second) //等所有worker都启动完毕
	defer StopWorkers()

	sentinel := service.NewSentinel(etcdServers)
	//测试Add接口
	book := Book{
		ISBN:    "436246383",
		Title:   "上下五千年",
		Author:  "李四",
		Price:   39.0,
		Content: "冰雪奇缘2 中文版电影原声带 Frozen 2 Mandarin Original Motion Picture",
	}
	doc := types.Document{
		Id:          book.ISBN,
		BitsFeature: 0b10011, //二进制
		Keywords:    []*types.Keyword{{Field: "content", Word: "唐朝"}, {Field: "content", Word: "文物"}, {Field: "title", Word: book.Title}},
		Bytes:       book.Serialize(),
	}
	n, err := sentinel.AddDoc(doc)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	} else {
		fmt.Printf("添加%d个doc\n", n)
	}
	//测试Search接口
	query := types.NewTermQuery("content", "文物")
	query = query.And(types.NewTermQuery("content", "唐朝"))
	docs := sentinel.Search(query, 0, 0, nil)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	} else {
		docId := ""
		if len(docs) == 0 {
			fmt.Println("无搜索结果")
		} else {
			for _, doc := range docs {
				book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
				if book != nil {
					fmt.Printf("%s %s %s %s %.1f\n", doc.Id, book.ISBN, book.Title, book.Author, book.Price)
					docId = doc.Id
				}
			}
		}
		//测试Delete接口
		if len(docId) > 0 {
			n := sentinel.DeleteDoc(docId)
			fmt.Printf("删除%d个doc\n", n)
		}

		//测试Search接口
		docs := sentinel.Search(query, 0, 0, nil)
		if len(docs) == 0 {
			fmt.Println("无搜索结果")
		} else {
			for _, doc := range docs {
				book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
				if book != nil {
					fmt.Printf("%s %s %s %s %.1f\n", doc.Id, book.ISBN, book.Title, book.Author, book.Price)
				}
			}
		}
	}
}
