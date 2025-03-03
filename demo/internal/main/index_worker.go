package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/WlayRay/ElectricSearch/demo/infrastructure"
	"github.com/WlayRay/ElectricSearch/service"
	"github.com/WlayRay/ElectricSearch/util"
	"google.golang.org/grpc"
)

var indexService *service.IndexServiceWorker

func GrpcIndexerInit() {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	indexService = new(service.IndexServiceWorker)
	indexService.Init(groupIndex)
	if rebuildIndex {
		util.Log.Printf("total workers = %d, worker index = %d", totalShards, groupIndex)
		infrastructure.BuildIndexFromCSVFile(csvFilePath, indexService.Indexer, totalShards, groupIndex)
	} else {
		indexService.Indexer.LoadFromIndexFile() //直接从正排索引中加载
	}

	service.RegisterIndexServiceServer(server, indexService)
	fmt.Printf("Start gprc server on port: %d\n", port)
	indexService.Register(etcdEndpoints, port, heartRate)

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
