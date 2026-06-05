package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

type AIResponseRequest struct {
	IsStream bool
	httpReq  *http.Request
	client   *http.Client
}

type ResponseRole string

const (
	ResponseRoleUser      ResponseRole = "user"
	ResponseRoleAssistant ResponseRole = "assistant"
	ResponseRoleSystem    ResponseRole = "system"
	ResponseRoleDeveloper ResponseRole = "developer"
)

type OpenAIResponseParam struct {
	Model              string                 `json:"model,omitempty"`
	Input              interface{}            `json:"input,omitempty"`
	Instructions       interface{}            `json:"instructions,omitempty"`
	PreviousResponseID string                 `json:"previous_response_id,omitempty"`
	Conversation       interface{}            `json:"conversation,omitempty"`
	Temperature        float64                `json:"temperature,omitempty"`
	TopP               float64                `json:"top_p,omitempty"`
	MaxOutputTokens    int64                  `json:"max_output_tokens,omitempty"`
	Stream             bool                   `json:"stream,omitempty"`
	Store              *bool                  `json:"store,omitempty"`
	Include            []string               `json:"include,omitempty"`
	Text               interface{}            `json:"text,omitempty"`
	Tools              []interface{}          `json:"tools,omitempty"`
	ToolChoice         interface{}            `json:"tool_choice,omitempty"`
	Reasoning          interface{}            `json:"reasoning,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	Extra              map[string]interface{} `json:"-"`
}

type responseParamJSON struct {
	Model              string                 `json:"model,omitempty"`
	Input              interface{}            `json:"input,omitempty"`
	Instructions       interface{}            `json:"instructions,omitempty"`
	PreviousResponseID string                 `json:"previous_response_id,omitempty"`
	Conversation       interface{}            `json:"conversation,omitempty"`
	Temperature        float64                `json:"temperature,omitempty"`
	TopP               float64                `json:"top_p,omitempty"`
	MaxOutputTokens    int64                  `json:"max_output_tokens,omitempty"`
	Stream             bool                   `json:"stream,omitempty"`
	Store              *bool                  `json:"store,omitempty"`
	Include            []string               `json:"include,omitempty"`
	Text               interface{}            `json:"text,omitempty"`
	Tools              []interface{}          `json:"tools,omitempty"`
	ToolChoice         interface{}            `json:"tool_choice,omitempty"`
	Reasoning          interface{}            `json:"reasoning,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

func (p OpenAIResponseParam) MarshalJSON() ([]byte, error) {
	body := responseParamJSON{
		Model:              p.Model,
		Input:              p.Input,
		Instructions:       p.Instructions,
		PreviousResponseID: p.PreviousResponseID,
		Conversation:       p.Conversation,
		Temperature:        p.Temperature,
		TopP:               p.TopP,
		MaxOutputTokens:    p.MaxOutputTokens,
		Stream:             p.Stream,
		Store:              p.Store,
		Include:            p.Include,
		Text:               p.Text,
		Tools:              p.Tools,
		ToolChoice:         p.ToolChoice,
		Reasoning:          p.Reasoning,
		Metadata:           p.Metadata,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyMap := make(map[string]interface{})
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return nil, err
	}
	for key, value := range p.Extra {
		if _, exists := bodyMap[key]; exists {
			return nil, errors.Errorf("[ai_client]extra field conflicts with typed field: %s", key)
		}
		bodyMap[key] = value
	}
	return json.Marshal(bodyMap)
}

type ResponseInputContent interface {
	responseInputContent()
}

type ResponseInputMessage struct {
	Type    string       `json:"type,omitempty"`
	Role    ResponseRole `json:"role"`
	Content interface{}  `json:"content"`
	Phase   string       `json:"phase,omitempty"`
	Status  string       `json:"status,omitempty"`
}

type ResponseInputText struct {
	Text string `json:"text"`
}

func (ResponseInputText) responseInputContent() {}

func (c ResponseInputText) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		Type: "input_text",
		Text: c.Text,
	})
}

type ResponseInputImage struct {
	Detail   string `json:"detail,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

func (ResponseInputImage) responseInputContent() {}

func (c ResponseInputImage) MarshalJSON() ([]byte, error) {
	type imageJSON struct {
		Type     string `json:"type"`
		Detail   string `json:"detail,omitempty"`
		FileID   string `json:"file_id,omitempty"`
		ImageURL string `json:"image_url,omitempty"`
	}
	return json.Marshal(imageJSON{
		Type:     "input_image",
		Detail:   c.Detail,
		FileID:   c.FileID,
		ImageURL: c.ImageURL,
	})
}

type ResponseInputFile struct {
	FileData string `json:"file_data,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	FileURL  string `json:"file_url,omitempty"`
	Filename string `json:"filename,omitempty"`
}

func (ResponseInputFile) responseInputContent() {}

func (c ResponseInputFile) MarshalJSON() ([]byte, error) {
	type fileJSON struct {
		Type     string `json:"type"`
		FileData string `json:"file_data,omitempty"`
		FileID   string `json:"file_id,omitempty"`
		FileURL  string `json:"file_url,omitempty"`
		Filename string `json:"filename,omitempty"`
	}
	return json.Marshal(fileJSON{
		Type:     "input_file",
		FileData: c.FileData,
		FileID:   c.FileID,
		FileURL:  c.FileURL,
		Filename: c.Filename,
	})
}

type ResponseTextParam struct {
	Format      interface{} `json:"format,omitempty"`
	Verbosity   string      `json:"verbosity,omitempty"`
	TopLogprobs int         `json:"top_logprobs,omitempty"`
}

type ResponseReasoningParam struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

func (c *AIClient) MakeResponseReqBytes(param OpenAIResponseParam) (reqByts []byte, err error) {
	if param.Model == "" {
		return nil, errors.New("[ai_client]model is required")
	}
	if param.PreviousResponseID != "" && hasResponseConversation(param.Conversation) {
		return nil, errors.New("[ai_client]previous_response_id and conversation cannot be used together")
	}
	if param.MaxOutputTokens == 0 {
		param.MaxOutputTokens = c.DefaultMaxToken
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

	reqByts, err = json.Marshal(param)
	if err != nil {
		return nil, errors.Wrap(err, "[ai_client]marshal open ai response request error")
	}
	return reqByts, nil
}

func hasResponseConversation(conversation interface{}) bool {
	if conversation == nil {
		return false
	}
	if conversationID, ok := conversation.(string); ok {
		return conversationID != ""
	}
	return true
}

func (c *AIClient) MakeResponseRequest(method string, param OpenAIResponseParam) (*AIResponseRequest, error) {
	reqByts, err := c.MakeResponseReqBytes(param)
	if err != nil {
		return nil, err
	}

	reqBuffer := bytes.NewBuffer(reqByts)
	if method == "" {
		method = "responses"
	}
	url := c.BaseURL + "/" + method
	httpReq, err := http.NewRequest("POST", url, reqBuffer)
	if err != nil {
		return nil, errors.Wrap(err, "[ai_client]make request to send response error")
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Key)
	httpReq.Header.Set("Content-Type", "application/json")
	if param.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	res := &AIResponseRequest{
		IsStream: param.Stream,
		httpReq:  httpReq,
		client:   c.client,
	}
	return res, err
}

func (r *AIResponseRequest) GetResp(ctx context.Context) (*RespOpenAIResponse, error) {
	resp := &RespOpenAIResponse{
		IsStream:          r.IsStream,
		EmptyMsgLineLimit: 300,
	}
	var err error
	httpReq := r.httpReq
	if ctx != nil {
		httpReq = httpReq.WithContext(ctx)
	}
	resp.httpResp, err = r.client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "send response request to ai error")
	}
	if r.IsStream {
		resp.respReader = bufio.NewReader(resp.httpResp.Body)
	}

	return resp, nil
}
