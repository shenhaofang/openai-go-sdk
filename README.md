# openai-go-sdk

## Installation
Use go get to install SDK：
```shell
go get github.com/shenhaofang/openai-go-sdk
```

## Usage
```go
package main

import (
	"context"
	"fmt"
	"github.com/shenhaofang/openai-go-sdk"
	"log"
)

const (
	DefaultRequestTimeout      = 30 * time.Second
	DefaultMaxIdleConns        = 100
	DefaultMaxIdleConnsPerHost = 50
	DefaultMaxConnsPerHost     = 200
	DefaultIdleConnTimeout     = 20 * time.Minute
)


// 查询图片描述
func (dao *imgDescDao) GetImgDesc(ctx context.Context, imgURLs map[int]string) (map[int]string, error) {
			subBatchJob.UserMsgContents = append(subBatchJob.UserMsgContents, )

			// 请求ai大模型
			chatParam := openai.OpenAIChatParam{
				Model: "qwen-vl-plus",
				Message: []openai.Message{
					{
						Role:    openai.RoleSystem,
						Content: openai.TextContent("你是一个图片内容描述助手，图文创作者将用你提供的图片描述进行创作"),
					},
					{
						Role:    openai.RoleUser,
						Content: subBatchJob.UserMsgContents,
					},
				},
				TopP: 0.1,
			}
			
			log.WithCtxFields(ctx, log.Fields{
				"param": chatParam,
				"res":   resChat,
			}).Debugln("[AI-qwen-vl-plus]get img desc from ai")
			if resChat != nil && resChat.Error != nil {
				log.WithCtxFields(ctx, log.Fields{
					"error": resChat.Error.Error(),
				}).Errorln("[AI-qwen-vl-plus]get img desc from ai error")
				jobErr = err
				return
			}
			// 解析返回结果
			subBatchJob.Res = dao.ParseAiChatResp(subBatchJob.ImgObjList, resChat)
		}(job)
	}
}

func main() {
	// Create a new client
	aiClient: openai.NewAIClient(conf.GlobalConfig.AiImgDealer.Key, conf.GlobalConfig.AiImgDealer.BaseURL, openai.ClientDefaultParamOption{
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
    }),

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
                Role:    openai.RoleUser,
                Content: openai.UserArrContent{
                    openai.UserImgContent{
                        Type: "image_url",
                        ImageURL: openai.ImgURL{
                            URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg",
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
                    }
                },
            },
        },
        TopP: 0.1,
    }

	// make chat request
    aiChatReq, err := dao.aiClient.MakeChatRequest("chat/completions", chatParam)
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
	fmt.Println(aiChatResp.Choices[0].Message.Content)


    /**
    * stream request
    */
    // text/event-stream
    chatParam.Stream = true

    // make chat request
    streamChatReq, err := dao.aiClient.MakeChatRequest("chat/completions", chatParam)
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
	for resGot, err = aiChatResp.Recv(); err == nil && resGot.Error == nil; resGot, err = aiChatResp.Recv() {
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
```