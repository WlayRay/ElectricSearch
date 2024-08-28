package util

import (
	"path"
	"runtime"
)

var (
	RootPath string
)

// 获取项目的根路径
func init() {
	RootPath = path.Dir(GetCurrentPath()+"..") + "/"
}

func GetCurrentPath() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

