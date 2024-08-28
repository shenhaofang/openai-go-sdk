# openai-go-sdk

OpenAI Docs API Reference: https://platform.openai.com/docs/api-reference/introduction

## Installation
Use go get to install SDK：
```shell
go get github.com/shenhaofang/openai-go-sdk
```

## Usage
```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/shenhaofang/openai-go-sdk"
)

const (
	DefaultRequestTimeout      = 30 * time.Second
	DefaultMaxIdleConns        = 100
	DefaultMaxIdleConnsPerHost = 50
	DefaultMaxConnsPerHost     = 200
	DefaultIdleConnTimeout     = 20 * time.Minute

	APIKey  = "your-api-key"
	BaseUrl = "https://dashscope.aliyuncs.com/compatible-mode/v1"
)

func main() {
	// Create a new client
	aiClient := openai.NewAIClient(APIKey, BaseUrl, openai.ClientDefaultParamOption{
		MaxToken: 1500,
	}, openai.ClientHTTPClientOption{
		Client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        DefaultMaxIdleConns,
				MaxIdleConnsPerHost: DefaultMaxIdleConnsPerHost,
				MaxConnsPerHost:     DefaultMaxConnsPerHost,
				IdleConnTimeout:     DefaultIdleConnTimeout,
			},
			Timeout: DefaultRequestTimeout,
		},
	})

	// Create a new context
	ctx := context.Background()

	// text/json
	// Create a new completion request
	chatParam := openai.OpenAIChatParam{
		Model: "qwen-vl-plus",
		Message: []openai.Message{
			{
				Role:    openai.RoleSystem,
				Content: openai.TextContent("你是一个图片解析助手"),
			},
			{
				Role: openai.RoleUser,
				Content: openai.UserArrContent{
					openai.UserImgContent{
						Type: "image_url",
						ImageURL: openai.ImgURL{
							URL: "https://wxls-cms.oss-cn-hangzhou.aliyuncs.com/online/2024-04-18/218da022-f4bf-456a-99af-5cb8e157f7b8.jpg",
						},
					},
					openai.UserImgContent{
						Type: "image_url",
						ImageURL: openai.ImgURL{
							URL: "https://dashscope.oss-cn-beijing.aliyuncs.com/images/dog_and_girl.jpeg",
						},
					},
					openai.UserTextContent{
						Type: "text",
						Text: `请问这些图片里边都是啥？`,
					},
				},
			},
		},
		TopP: 0.1,
	}

	// make chat request
	aiChatReq, err := aiClient.MakeChatRequest("chat/completions", chatParam)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
		return
	}
	// send msg to ai
	aiChatResp, err := aiChatReq.GetResp(ctx)
	if err != nil {
		log.Fatalf("Error get resp: %v", err)
		return
	}

	// get ai response
	resChat, err := aiChatResp.Get()
	if err != nil {
		log.Fatalf("Error get msg from ai resp: %v", err)
		return
	}
	// Print the response content
	fmt.Println(resChat.Choices[0].Message.Content)

	/**
	 * stream request
	 */
	// text/event-stream
	chatParam.Stream = true

	// make chat request
	streamChatReq, err := aiClient.MakeChatRequest("chat/completions", chatParam)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
		return
	}
	// send msg to ai
	streamChatResp, err := streamChatReq.GetResp(ctx)
	if err != nil {
		log.Fatalf("Error get resp: %v", err)
		return
	}

	defer streamChatResp.Close()
	var resGot *openai.RespAIChatStream
	res := ""
	for resGot, err = streamChatResp.Recv(); err == nil && resGot.Error == nil; resGot, err = streamChatResp.Recv() {
		if resGot.Choices[0].Delta.Content == "" {
			continue
		}
		res += resGot.Choices[0].Delta.Content
	}
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Fatalf("Error get msg from ai resp: %v", err)
		return
	}
	if resGot != nil && resGot.Error != nil {
		log.Fatalf("Error get msg from ai resp: %v", resGot.Error)
		return
	}
	// Print the response text
	fmt.Println(res)

	/**
	 * file upload request
	 */
	fileText := bytes.NewReader([]byte(`《侠客行》 - 李白

	赵客缦胡缨，吴钩霜雪明。银鞍照白马，飒沓如流星。

	十步杀一人，千里不留行。事了拂衣去，深藏身与名。

	闲过信陵饮，脱剑膝前横。将炙啖朱亥，持觞劝侯嬴。

	三杯吐然诺，五岳倒为轻。眼花耳热后，意气素霓生。

	救赵挥金锤，邯郸先震惊。千秋二壮士，烜赫大梁城。

	纵死侠骨香，不惭世上英。谁能书阁下，白首太玄经。`))

	// make chat request
	fileInfo, err := aiClient.UpdateFile("files", openai.OpenAIFileCreateParam{
		File:     fileText,
		FileName: "test.txt",
		Purpose:  "file-extract",
	})
	if err != nil {
		log.Fatalf("Error get msg from ai resp: %v", err)
		return
	}
	fmt.Println(fileInfo)

	fileInfo2, err := aiClient.RetrieveFile("files", fileInfo.ID)
	if err != nil {
		log.Fatalf("Error get msg from ai resp: %v", err)
		return
	}
	fmt.Println(fileInfo2)

	// 请求ai大模型
	chatParam = openai.OpenAIChatParam{
		Model: "qwen-long",
		Message: []openai.Message{
			{
				Role:    openai.RoleSystem,
				Content: openai.TextContent("你是一个文档小助手，辅助用户完成文档解读"),
			},
			{
				Role:    openai.RoleSystem,
				Content: openai.TextContent("fileid://" + fileInfo.ID),
			},
			{
				Role:    openai.RoleUser,
				Content: openai.TextContent("这都写了点啥？给说说呗！"),
			},
		},
		TopP: 0.3,
	}

	// make chat request
	aiChatReq, err = aiClient.MakeChatRequest("chat/completions", chatParam)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
		return
	}
	// send msg to ai
	aiChatResp, err = aiChatReq.GetResp(ctx)
	if err != nil {
		log.Fatalf("Error get resp: %v", err)
		return
	}

	// get ai response
	resChat, err = aiChatResp.Get()
	if err != nil {
		log.Fatalf("Error get msg from ai resp: %v", err)
		return
	}
	// Print the response content
	fmt.Println(resChat.Choices[0].Message.Content)
}

```