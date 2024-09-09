package filter

import "github.com/WlayRay/ElectricSearch/demo/common"

type ViewFilter struct{}

func (ViewFilter) Apply(ctx *common.VideoSearchContext) {
	request := ctx.Request
	if request == nil {
		return
	}
	if request.MinViewCount >= request.MaxViewCount {
		return
	}
	videos := make([]*common.BiliBiliVideo, 0, len(ctx.Videos))
	for _, video := range ctx.Videos {
		if video.ViewCount >= int32(request.MinViewCount) && video.ViewCount <= int32(request.MaxViewCount) {
			videos = append(videos, video)
		}
	}
	ctx.Videos = videos
}
