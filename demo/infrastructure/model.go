package infrastructure

import (
	"context"

	"github.com/WlayRay/ElectricSearch/service"
)

type SearchRequest struct {
	Author       string   `json:"author"`
	Keywords     []string `json:"keywords"`
	Categories   []string `json:"categories"`
	MinViewCount int      `json:"minViewCount"`
	MaxViewCount int      `json:"maxViewCount"`
}

type VideoSearchContext struct {
	Ctx     context.Context
	Indexer service.IIndexer
	Request *SearchRequest
	Videos  []*BiliBiliVideo
}
