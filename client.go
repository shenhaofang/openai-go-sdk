package openai

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

type AIClient struct {
	Key                string
	BaseURL            string
	DefaultMaxToken    int64
	DefaultTemperature float64
	client             *http.Client
}

type ClientOption interface {
	Set(c *AIClient)
}

func NewAIClient(key string, baseURL string, opts ...ClientOption) *AIClient {
	defaultClient := &AIClient{
		Key:     key,
		BaseURL: baseURL,
	}
	for _, opt := range opts {
		opt.Set(defaultClient)
	}
	if defaultClient.BaseURL == "" {
		defaultClient.BaseURL = "https://api.openai.com/v1"
	}
	if defaultClient.client == nil {
		defaultClient.client = http.DefaultClient
	}
	return defaultClient
}

func (c *AIClient) WithOptions(opts ...ClientOption) *AIClient {
	for _, opt := range opts {
		opt.Set(c)
	}
	return c
}

type ClientDefaultParamOption struct {
	MaxToken    int64
	Temperature float64
}

func (o ClientDefaultParamOption) Set(c *AIClient) {
	if o.MaxToken > 0 {
		c.DefaultMaxToken = o.MaxToken
	}
	if o.Temperature > 0 {
		c.DefaultTemperature = o.Temperature
	}
}

type ClientHTTPClientOption struct {
	Client *http.Client
}

func (o ClientHTTPClientOption) Set(c *AIClient) {
	if o.Client != nil {
		c.client = o.Client
	}
}

func (c *AIClient) MakeChatRequest(method string, param OpenAIChatParam) (*AIRequest, error) {
	if param.Model == "" {
		param.Model = "gpt-3.5-turbo-0301" //gpt-3.5-turbo or gpt-3.5-turbo-0301
	}
	if param.N < 1 {
		param.N = 1
	}
	if param.N > 5 {
		param.N = 5
	}
	if len(param.Message) == 0 {
		return nil, nil
	}
	if param.MaxTokens == 0 {
		param.MaxTokens = c.DefaultMaxToken
	}
	if param.Temperature < 0 || param.Temperature > 2 {
		param.Temperature = c.DefaultTemperature
	}
	if param.TopP > 0 {
		param.Temperature = 0
		if param.TopP > 1 {
			param.TopP = 1
		}
	}
	if param.Temperature == 0 && param.TopP == 0 {
		param.TopP = 1
	}
	if param.FrequencyPenalty > 2 || param.FrequencyPenalty < (-2) {
		param.FrequencyPenalty = 0
	}
	if param.PresencePenalty > 2 || param.PresencePenalty < (-2) {
		param.PresencePenalty = 0.6
	}
	reqByts, err := json.Marshal(param)
	if err != nil {
		err = errors.Wrap(err, "[ai_client]marshal open ai request error")
		return nil, err
	}

	reqBuffer := bytes.NewBuffer(reqByts)
	if method == "" {
		method = "chat/completions"
	}
	url := c.BaseURL + "/" + method
	// 发送请求
	httpReq, err := http.NewRequest("POST", url, reqBuffer)
	if err != nil {
		err = errors.Wrap(err, "[ai_client]make request to send msg error")
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Key)
	// req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15")
	httpReq.Header.Set("Content-Type", "application/json")
	if param.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}
	res := &AIRequest{
		IsStream: param.Stream,
		httpReq:  httpReq,
		client:   c.client,
	}
	return res, err
}
