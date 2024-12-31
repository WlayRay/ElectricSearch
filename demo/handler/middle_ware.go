package handler

import (
	"net/url"

	"github.com/gin-gonic/gin"
)

func GetUserInfo(ctx *gin.Context) {
	userName, err := url.QueryUnescape(ctx.Request.Header.Get("UserName"))
	if err == nil {
		ctx.Set("user_name", userName)
	}
}
