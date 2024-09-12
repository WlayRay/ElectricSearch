package internal

import (
	"reflect"
	"sync"
	"time"

	"github.com/WlayRay/ElectricSearch/demo/common"
	"github.com/WlayRay/ElectricSearch/demo/internal/filter"
	"github.com/WlayRay/ElectricSearch/demo/internal/recaller"
	"github.com/WlayRay/ElectricSearch/util"
	"golang.org/x/exp/maps"
)

type Recaller interface {
	Recall(ctx *common.VideoSearchContext) []*common.BiliBiliVideo
}

type Filter interface {
	Apply(ctx *common.VideoSearchContext)
}

// 模版方法模式，超类
type VideoSearcher struct {
	Recallers []Recaller
	Filters   []Filter
}

func (search *VideoSearcher) WithRecaller(recallers ...Recaller) {
	search.Recallers = append(search.Recallers, recallers...)
}

func (search *VideoSearcher) WithFilter(filters ...Filter) {
	search.Filters = append(search.Filters, filters...)
}

func (search *VideoSearcher) Recall(searchCtx *common.VideoSearchContext) {
	if len(search.Recallers) == 0 {
		return
	}
	// 并行执行多路召回
	colletion := make(chan *common.BiliBiliVideo, 1000)
	wg := sync.WaitGroup{}
	wg.Add(len(search.Recallers))

	for _, recaller := range search.Recallers {
		go func(recaller Recaller) {
			defer wg.Done()
			rule := reflect.TypeOf(recaller).Name()
			result := recaller.Recall(searchCtx)
			util.Log.Printf("recall %d docs by %s", len(result), rule)
			for _, video := range result {
				colletion <- video
			}
		}(recaller)
	}

	videoMap := make(map[string]*common.BiliBiliVideo, 1000)
	receiveDone := make(chan struct{})
	go func() {
		for {
			video, ok := <-colletion
			if !ok {
				break
			}
			videoMap[video.Id] = video
		}
		receiveDone <- struct{}{}
	}()
	wg.Wait()
	close(colletion)
	<-receiveDone
	searchCtx.Videos = maps.Values(videoMap)
}

func (search *VideoSearcher) Filter(searchCtx *common.VideoSearchContext) {
	if len(search.Filters) == 0 {
		return
	}
	for _, filter := range search.Filters {
		filter.Apply(searchCtx)
	}
}

func (search *VideoSearcher) Search(searchCtx *common.VideoSearchContext) []*common.BiliBiliVideo {
	t1 := time.Now()
	search.Recall(searchCtx)
	t2 := time.Now()
	util.Log.Printf("recall %d docs in %d ms", len(searchCtx.Videos), t2.Sub(t1).Milliseconds())

	search.Filter(searchCtx)
	t3 := time.Now()
	util.Log.Printf("after filter remain %d docs in %d ms", len(searchCtx.Videos), t3.Sub(t2).Milliseconds())

	return searchCtx.Videos
}

type AllVideoSearcher struct {
	VideoSearcher
}

func NewAllVideoSearcher() *AllVideoSearcher {
	searcher := new(AllVideoSearcher)
	searcher.WithRecaller(recaller.KeywordRecaller{})
	searcher.WithFilter(filter.ViewFilter{})
	return searcher
}

type UpVideoSearcher struct {
	VideoSearcher
}

func NewUpVideoSearcher() *UpVideoSearcher {
	searcher := new(UpVideoSearcher)
	searcher.WithRecaller(recaller.KeywordAuthorRecaller{})
	searcher.WithFilter(filter.ViewFilter{})
	return searcher
}
