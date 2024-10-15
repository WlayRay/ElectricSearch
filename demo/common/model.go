package common

import (
	"context"
	"github.com/WlayRay/ElectricSearch/service"
)

type SearchRequest struct {
	Author       string
	Keywords     []string
	Categories   []string
	MinViewCount int
	MaxViewCount int
}

type VideoSearchContext struct {
	Ctx     context.Context
	Indexer service.IIndexer
	Request *SearchRequest
	Videos  []*BiliBiliVideo
}
