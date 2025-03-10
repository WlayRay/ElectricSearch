package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	infrastructure "github.com/WlayRay/ElectricSearch/demo/infrastructure"
)

type SearchRequest struct {
	Author       string   `json:"author"`
	Keywords     []string `json:"keywords"`
	Categories   []string `json:"categories"`
	MinViewCount int      `json:"minViewCount"`
	MaxViewCount int      `json:"maxViewCount"`
}

func TestSearch(t *testing.T) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	request := SearchRequest{
		Keywords:     []string{"go"},
		Categories:   []string{"编程"},
		MinViewCount: 0,
		MaxViewCount: 300000,
	}

	bs, _ := json.Marshal(request)

	req, err := http.NewRequest("POST", "http://0.0.0.0:9000/search", bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var videos []infrastructure.BiliBiliVideo
		json.Unmarshal(content, &videos)
		for _, video := range videos {
			fmt.Printf("%s %d %s %s\n", video.Id, video.ViewCount, video.Title, strings.Join(video.Keywords, "|"))
		}
	} else {
		fmt.Println(resp.Status)
		t.Fail()
	}
}

func TestSearchByAuthor(t *testing.T) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	request := SearchRequest{
		Keywords:     []string{"go"},
		Categories:   []string{"编程"},
		MinViewCount: 0,
		MaxViewCount: 300000,
	}

	bs, _ := json.Marshal(request)

	req, err := http.NewRequest("POST", "http://0.0.0.0:9000/up_search", bytes.NewReader(bs))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-UserName", "七牛云")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		var videos []infrastructure.BiliBiliVideo
		json.Unmarshal(content, &videos)
		for _, video := range videos {
			fmt.Printf("%s %d %s %s\n", video.Id, video.ViewCount, video.Title, strings.Join(video.Keywords, "|"))
		}
	} else {
		fmt.Println(resp.Status)
		t.Fail()
	}
}
