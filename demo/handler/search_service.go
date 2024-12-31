package handler

import (
	"log"
	"net/http"
	"strings"

	infrastructure "github.com/WlayRay/ElectricSearch/demo/infrastructure"
	"github.com/WlayRay/ElectricSearch/demo/internal"
	"github.com/WlayRay/ElectricSearch/service"
	"github.com/gin-gonic/gin"
)

var Indexer service.IIndexer

func getKeywords(words []string) []string {
	keywords := make([]string, 0, len(words))
	for _, word := range words {
		keyword := strings.TrimSpace(strings.ToLower(word))
		if word != "" {
			keywords = append(keywords, keyword)
		}
	}
	return keywords
}

// 全站搜索接口
func SearchAll(ctx *gin.Context) {
	var searchRequest infrastructure.SearchRequest
	if err := ctx.ShouldBindJSON(&searchRequest); err != nil {
		log.Printf("bind request parameter failed: %s", err)
		ctx.JSON(400, gin.H{
			"error": "invalid request json!",
		})
		return
	}
	searchRequest.Keywords = getKeywords(searchRequest.Keywords)

	if len(searchRequest.Keywords) == 0 && len(searchRequest.Author) == 0 {
		ctx.String(http.StatusBadRequest, "关键词和作者不能同时为空")
		return
	}

	searchCtx := &infrastructure.VideoSearchContext{
		Ctx:     ctx,
		Request: &searchRequest,
		Indexer: Indexer,
	}

	searcher := internal.NewAllVideoSearcher()
	videos := searcher.Search(searchCtx)
	ctx.JSON(http.StatusOK, videos)
}

// UP搜索自己视频的接口
func SearchByAuthor(ctx *gin.Context) {
	var searchRequest infrastructure.SearchRequest
	if err := ctx.ShouldBindJSON(&searchRequest); err != nil {
		log.Printf("bind request parameter failed: %s", err)
		ctx.JSON(400, gin.H{
			"error": "invalid request json!",
		})
		return
	}

	searchRequest.Keywords = getKeywords(searchRequest.Keywords)
	if len(searchRequest.Keywords) == 0 {
		ctx.String(http.StatusBadRequest, "关键词不能为空")
		return
	}

	userName, ok := ctx.Value("user_name").(string)
	if !ok || len(userName) == 0 {
		ctx.String(http.StatusBadRequest, "用户未登录")
		return
	}

	searchCtx := &infrastructure.VideoSearchContext{
		Ctx:     ctx,
		Request: &searchRequest,
		Indexer: Indexer,
	}

	searcher := internal.NewUpVideoSearcher()
	videos := searcher.Search(searchCtx)

	ctx.JSON(http.StatusOK, videos)
}
