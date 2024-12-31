package filter

import (
	infrastructure "github.com/WlayRay/ElectricSearch/demo/infrastructure"
)

type ViewFilter struct{}

func (ViewFilter) Apply(ctx *infrastructure.VideoSearchContext) {
	request := ctx.Request
	if request == nil {
		return
	}
	if request.MinViewCount >= request.MaxViewCount {
		return
	}
	videos := make([]*infrastructure.BiliBiliVideo, 0, len(ctx.Videos))
	for _, video := range ctx.Videos {
		if video.ViewCount >= int32(request.MinViewCount) && video.ViewCount <= int32(request.MaxViewCount) {
			videos = append(videos, video)
		}
	}
	ctx.Videos = videos
}
