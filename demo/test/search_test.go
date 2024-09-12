package test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/WlayRay/ElectricSearch/demo/common"
	"github.com/bytedance/sonic"
)

type SearchRequest struct {
	Author       string
	Keywords     []string
	Categories   []string
	MinViewCount int
	MaxViewCount int
}

func TestSearch(t *testing.T) {
	client := http.Client{
		Timeout: 100 * time.Millisecond,
	}

	request := SearchRequest{
		Keywords:     []string{"go", "项目"},
		Categories:   []string{"编程"},
		MinViewCount: 0,
		MaxViewCount: 300000,
	}

	bs, _ := sonic.Marshal(request)

	req, err := http.NewRequest("POST", "http://localhost:7887/search", bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/sonic")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var datas []common.BiliBiliVideo
		sonic.Unmarshal(content, &datas)
		for _, data := range datas {
			fmt.Printf("%s %d %s %s\n", data.Id, data.ViewCount, data.Title, strings.Join(data.Keywords, "|"))
		}
	} else {
		fmt.Println(resp.Status)
		t.Fail()
	}
}
