package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/WlayRay/ElectricSearch/demo/handler"
	"github.com/WlayRay/ElectricSearch/demo/infrastructure"
	"github.com/WlayRay/ElectricSearch/service"
)

func WebServerInit(mode int) {
	switch mode {
	case 1:
		standaloneIndexer := new(service.Indexer)
		if err := standaloneIndexer.Init(documentEstimateNum, dbType, dbPath); err != nil {
			panic(err)
		}
		if rebuildIndex {
			infrastructure.BuildIndexFromCSVFile(csvFilePath, standaloneIndexer, 0, 0)
		} else {
			standaloneIndexer.LoadFromIndexFile()
		}
		handler.Indexer = standaloneIndexer
	case 3:
		handler.Indexer = service.NewSentinel(etcdEndpoints)
	default:
		panic("Unsupported mode")
	}
}

func WebServerTeardown(signalCh chan os.Signal) {
	<-signalCh
	_ = handler.Indexer.Close()
	os.Exit(0)
}

func WebServerMain(mode int) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go WebServerTeardown(signalCh)
	WebServerInit(mode)
}
