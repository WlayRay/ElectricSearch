package main

import (
	"github.com/WlayRay/ElectricSearch/demo/infrastructure"
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
	if err := indexService.Init(etcdEndpoints, currentGroup, heartRate); err != nil {
		panic(err)
	}

	service.RegisterIndexServiceServer(server, indexService)
	if err := indexService.Register(port); err != nil {
		_ = indexService.Close()
		util.Log.Fatalf("failed to register: %v", err)
	}

	if rebuildIndex {
		infrastructure.BuildIndexFromCSVFile(csvFilePath, indexService.Indexer, indexService.Hub.CountIndexGroup(), currentGroup)
	} else {
		indexService.Indexer.LoadFromIndexFile() //直接从正排索引中加载
	}

	err = server.Serve(lis)
	if err != nil {
		util.Log.Fatalf("failed to serve: %v", err)
		indexService.Close()
	}
}

func GrpcIndexerTeardown() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
	_ = indexService.Close()
	os.Exit(0)
}

func GrpcIndexerMain() {
	go GrpcIndexerTeardown()
	GrpcIndexerInit()
}
