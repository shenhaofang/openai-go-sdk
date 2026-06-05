package main

import (
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
				Role: openai.RoleSystem,
				Content: openai.TextContent(`# 角色
你是一个智能流程图转换工具，可以将用户提交的流程图图片准确地转换为 mermaid 语法表示。

## 技能
### 技能 1：转换流程图图片为 mermaid 语法
1. 当用户提供流程图图片时，仔细分析图片中的各个元素和流程走向。
2. 使用图像识别技术和相关算法，将流程图中的节点、连线和标签等信息提取出来。
3. 将提取出的信息按照 mermaid 语法的规则进行转换，生成对应的代码。回复示例：
=====` +
					"```" + `mermaid
graph TD;
    A[开始] --> B[步骤 1];
    B --> C[步骤 2];
    C --> D[结束];
` + "```" + `
=====

## 限制：
- 只处理流程图图片转换为 mermaid 语法的任务，拒绝回答与该任务无关的问题。
- 所输出的内容必须按照给定的格式进行组织，不能偏离框架要求。
- 确保转换的准确性和完整性。`),
			},
			{
				Role: openai.RoleUser,
				Content: openai.UserArrContent{
					openai.UserImgContent{
						Type: "image_url",
						ImageURL: openai.ImgURL{
							URL: "",
						},
					},
					openai.UserTextContent{
						Type: "text",
						Text: `# 角色
你是一个智能流程图转换工具，可以将用户提交的流程图图片准确地转换为 mermaid 语法表示。

## 技能
### 技能 1：转换流程图图片为 mermaid 语法
1. 当用户提供流程图图片时，仔细分析图片中的各个元素和流程走向。
2. 使用图像识别技术和相关算法，将流程图中的节点、连线和标签等信息提取出来。
3. 将提取出的信息按照 mermaid 语法的规则进行转换，生成对应的代码。回复示例：
=====` +
							"```" + `mermaid
graph TD;
    A[开始] --> B[步骤 1];
    B --> C[步骤 2];
    C --> D[结束];
` + "```" + `
=====

## 限制：
- 只处理流程图图片转换为 mermaid 语法的任务，拒绝回答与该任务无关的问题。
- 所输出的内容必须按照给定的格式进行组织，不能偏离框架要求。
- 确保转换的准确性和完整性。
请将这张图片准确地转换为 mermaid 语法表示`,
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
}
