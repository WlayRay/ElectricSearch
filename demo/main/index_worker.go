package main

import (
	"fmt"
	"github.com/WlayRay/ElectricSearch/demo/common"
	"github.com/WlayRay/ElectricSearch/service"
	"github.com/WlayRay/ElectricSearch/util"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var indexService *service.IndexServiceWorker

func GrpcIndexerInit() {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	indexService = new(service.IndexServiceWorker)
	// 初始化索引
	indexService.Init(workerIndex)
	if rebuildIndex {
		util.Log.Printf("totalworkers = %d, workerindex = %d", totalWorkers, workerIndex)
		common.BuildIndexFromCSVFile(csvFilePath, indexService.Indexer, totalWorkers, workerIndex)
	} else {
		indexService.Indexer.LoadFromIndexFile() //直接从正排索引中加载
	}

	service.RegisterIndexServiceServer(server, indexService)
	fmt.Printf("Start gprc server on port: %d\n", port)
	indexService.Regist(etcdEndpoints, port, hearRate)

	err = server.Serve(lis)
	if err != nil {
		indexService.Close()
		fmt.Printf("Failed to start grpc server: %v\n", err)
	}
}

func GrpcIndexerTeardown() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
	indexService.Close()
	os.Exit(0)
}

func GrpcIndexerMain() {
	go GrpcIndexerTeardown()
	GrpcIndexerInit()
}
