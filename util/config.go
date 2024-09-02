package util

import (
	"bufio"
	"os"
	"path"
	"runtime"
	"strings"
)

var (
	RootPath       string
	Configurations map[string]string
)

// 获取项目的根路径
func init() {
	RootPath = path.Dir(GetCurrentPath()+"..") + "/"

	initConf := RootPath + "init.conf"
	file, err := os.Open(initConf)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	Configurations = make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			Configurations[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func GetCurrentPath() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}
