package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/WlayRay/ElectricSearch/demo/common"
	"github.com/WlayRay/ElectricSearch/demo/handler"
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
			common.BuildIndexFromCSVFile(csvFilePath, standaloneIndexer, 0, 0)
		} else {
			standaloneIndexer.LoadFromIndexFile()
		}
		handler.Indexer = standaloneIndexer
	case 3:
	default:
	}
}

func WebServerTeardown() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
	handler.Indexer.Close()
	os.Exit(0)
}

func WebServerMain(mode int) {
	go WebServerTeardown()
	WebServerInit(mode)
}
