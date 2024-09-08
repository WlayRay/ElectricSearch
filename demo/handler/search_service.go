package handler

import (
	// "context"
	"log"
	"net/http"
	"strings"

	"github.com/WlayRay/ElectricSearch/demo/common"
	"github.com/WlayRay/ElectricSearch/service"
	"github.com/WlayRay/ElectricSearch/types"
	"github.com/WlayRay/ElectricSearch/util"
	"github.com/gin-gonic/gin"
	"github.com/gogo/protobuf/proto"
)

var Indexer service.IIndexer

func getKeywords(words []string) []string {
	keywords := make([]string, 0, len(words))
	for _, keyword := range keywords {
		word := strings.TrimSpace(strings.ToLower(keyword))
		if word != "" {
			keywords = append(keywords, word)
		}
	}
	return keywords
}

// 搜索接口
func Search(ctx *gin.Context) {
	var searchRequest common.SearchRequest
	if err := ctx.ShouldBindBodyWithJSON(&searchRequest); err != nil {
		log.Printf("search request bind error: %v", err)
		ctx.JSON(400, gin.H{
			"error": "invalid request json!",
		})
		return
	}

	keywords := getKeywords(searchRequest.Keywords)
	if len(keywords) == 0 || len(searchRequest.Author) == 0 {
		ctx.String(http.StatusBadRequest, "关键词和作者不能同时为空")
		return
	}

	query := new(types.TermQuery)
	for _, keyword := range keywords {
		query = query.And(types.NewTermQuery("content", keyword))
	}
	query = query.And(types.NewTermQuery("author", searchRequest.Author))
	orFlags := []uint64{common.GetCategoriesBits(searchRequest.Categories)}
	docs := Indexer.Search(query, 0, 0, orFlags)
	videos := make([]common.BiliBiliVideo, 0, len(docs))
	for _, doc := range docs {
		var video common.BiliBiliVideo
		if err := proto.Unmarshal(doc.Bytes, &video); err == nil {
			if video.ViewCount >= int32(searchRequest.MinViewCount) && video.ViewCount <= int32(searchRequest.MaxViewCount) {
				videos = append(videos, video)
			}
		}
	}
	util.Log.Printf("return %d videos", len(videos))
	ctx.JSON(http.StatusOK, videos)
}

func SearchAll(ctx *gin.Context) {
	var searchRequest common.SearchRequest
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

	// searchCtx := common.VideoSearchContext{
	// 	Ctx:     context.Background(),
	// 	Request: &searchRequest,
	// 	Indexer: Indexer,
	// }


}
