package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Config struct {
	FirstJWTAccess  string `json:"first_jwt_access"`
	FirstJWTRefresh string `json:"first_jwt_refresh"`
	ChatID          string `json:"chat_id"`
	KimiplusID      string `json:"kimiplus_id"`
}

func ReadFile(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close() // 确保文件在最后被关闭
	// 读取文件内容
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	return content
}

type RspToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var auth_token_error = errors.New("auth_token_error")

func GetStreamData(str string, sb *strings.Builder) bool {
	// 定义正则表达式，分别匹配 event 和 text 的值
	textRegex := regexp.MustCompile(`.*"event":"cmpl".*"text":"([^"]+)"`)
	// 提取 event
	eventMatches := textRegex.FindStringSubmatch(str)
	if len(eventMatches) > 1 {
		sb.WriteString(eventMatches[1])
		return true
	} else {
		return false
	}
}

func AddHeader(req *http.Request, jwt string) {
	req.Header.Add("accept", "*/*")
	req.Header.Add("authorization", "Bearer "+jwt)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("origin", "https://kimi.moonshot.cn")
	req.Header.Add("referer", "https://kimi.moonshot.cn/chat/")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")

}

func RefreshToken(jwt string) (*RspToken, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET",
		"https://kimi.moonshot.cn/api/auth/token/refresh", nil)
	if err != nil {
		return nil, err
	}
	AddHeader(req, jwt)
	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	rspToken := &RspToken{}
	err = json.Unmarshal(body, rspToken)
	if err != nil {
		return nil, err
	}
	if rspToken.RefreshToken == "" || rspToken.AccessToken == "" {
		return nil, errors.New("更新token出错")
	}
	return rspToken, nil
}

func GetData(keyword, chatId, jwt, kimiplus_id string) (*strings.Builder, error) {
	url := "https://kimi.moonshot.cn/api/chat/" + chatId + "/completion/stream"
	method := "POST"
	payload := []byte(`{"messages":[{"role":"user","content":"` + keyword +
		`"}],"use_search":true,"extend":{"sidebar":true},"kimiplus_id":"` +
		kimiplus_id + `","use_research":false,"refs":[]}`)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	AddHeader(req, jwt)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	article := &strings.Builder{}
	// 使用 bufio.Scanner 逐行读取响应
	scanner := bufio.NewScanner(res.Body)
	errMsg := ""
	for scanner.Scan() {
		line := scanner.Text()
		if !GetStreamData(line, article) {
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	if article.Len() == 0 || strings.Contains(errMsg, "auth.token.invalid") {
		return article, auth_token_error
	}
	return article, nil
}

func main() {
	config := &Config{}
	json.Unmarshal(ReadFile("config.json"), &config)
	jwt_access := config.FirstJWTAccess
	jwt_refresh := config.FirstJWTRefresh
	kimiplus_id := config.KimiplusID
	chatId := config.ChatID
	keywords := strings.Split(string(ReadFile("keyword.txt")), "\n")
	for _, keyword := range keywords {
		fmt.Printf("正在获取文章:%s\n", keyword)
		sb, err := GetData(keyword, chatId, jwt_access, kimiplus_id)
		if err == auth_token_error {
			rsp, err := RefreshToken(jwt_refresh)
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
			jwt_access = rsp.AccessToken
			jwt_refresh = rsp.RefreshToken
			sb, err = GetData(keyword, chatId, jwt_access, kimiplus_id)
		}
		fmt.Printf("%v\n", sb.String())
	}
}
