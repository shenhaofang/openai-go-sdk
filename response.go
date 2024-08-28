package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type RespAIChat struct {
	ID                  string               `json:"id"`
	Object              string               `json:"object"`
	Created             int64                `json:"created"`
	Model               string               `json:"model"`
	Choices             []ChatChoice         `json:"choices"`
	SystemFingerprint   string               `json:"system_fingerprint"`
	PromptAnnotations   []PromptAnnotation   `json:"prompt_annotations,omitempty"`
	PromptFilterResults []PromptFilterResult `json:"prompt_filter_results,omitempty"`
	Usage               *ChatUsage           `json:"usage,omitempty"`
	Error               *AIError             `json:"error,omitempty"`
}

type AIError struct {
	Code    string      `json:"code"`
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Param   interface{} `json:"param"`
}

func (e *AIError) Error() string {
	return fmt.Sprintf("Error[%s]: %s(type:%s, param:%v)", e.Code, e.Message, e.Type, e.Param)
}

type RespAIChatStream struct {
	ID                  string               `json:"id"`
	Object              string               `json:"object"`
	Created             int64                `json:"created"`
	Model               string               `json:"model"`
	Choices             []StreamChatChoice   `json:"choices"`
	SystemFingerprint   string               `json:"system_fingerprint"`
	PromptAnnotations   []PromptAnnotation   `json:"prompt_annotations,omitempty"`
	PromptFilterResults []PromptFilterResult `json:"prompt_filter_results,omitempty"`
	Usage               *ChatUsage           `json:"usage,omitempty"`
	Error               *AIError             `json:"error,omitempty"`
}

type FinishReason string

// https://platform.openai.com/docs/api-reference/chat/object
type ChatChoice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason"`
}

// https://platform.openai.com/docs/api-reference/chat/streaming
type StreamChatChoice struct {
	Index        int                             `json:"index"`
	FinishReason FinishReason                    `json:"finish_reason"`
	Delta        ChatCompletionStreamChoiceDelta `json:"delta"` // A chat completion delta generated by streamed model responses.
}

type ChatCompletionStreamChoiceDelta struct {
	Content   string     `json:"content,omitempty"`
	Role      string     `json:"role,omitempty"`
	Refusal   string     `json:"refusal,omitempty"` // The refusal message generated by the model.
	ToolCalls []ToolFunc `json:"tool_calls,omitempty"`
}

const (
	// The reason the model stopped generating tokens.
	// This will be stop if the model hit a natural stop point or a provided stop sequence,
	// length if the maximum number of tokens specified in the request was reached,
	// content_filter if content was omitted due to a flag from our content filters,
	// tool_calls if the model called a tool,
	// or function_call (deprecated) if the model called a function.
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonFunctionCall  FinishReason = "function_call"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonNull          FinishReason = "null"
)

type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Hate struct {
	Filtered bool   `json:"filtered"`
	Severity string `json:"severity,omitempty"`
}
type SelfHarm struct {
	Filtered bool   `json:"filtered"`
	Severity string `json:"severity,omitempty"`
}
type Sexual struct {
	Filtered bool   `json:"filtered"`
	Severity string `json:"severity,omitempty"`
}
type Violence struct {
	Filtered bool   `json:"filtered"`
	Severity string `json:"severity,omitempty"`
}

type ContentFilterResults struct {
	Hate     Hate     `json:"hate,omitempty"`
	SelfHarm SelfHarm `json:"self_harm,omitempty"`
	Sexual   Sexual   `json:"sexual,omitempty"`
	Violence Violence `json:"violence,omitempty"`
}

type PromptAnnotation struct {
	PromptIndex          int                  `json:"prompt_index,omitempty"`
	ContentFilterResults ContentFilterResults `json:"content_filter_results,omitempty"`
}

type PromptFilterResult struct {
	Index                int                  `json:"index"`
	ContentFilterResults ContentFilterResults `json:"content_filter_results,omitempty"`
}

type RespOpenAI struct {
	IsStream          bool
	EmptyMsgLineLimit int
	httpResp          *http.Response
	respReader        *bufio.Reader
}

var (
	dataPrefix  = []byte("data:")
	errorPrefix = []byte(`data: {"error":`)
)

func (r *RespOpenAI) Get() (*RespAIChat, error) {
	res := new(RespAIChat)
	if r.IsStream {
		return nil, errors.New("[ai_resp]resp is stream, use Recv instead")
	}
	defer r.Close()
	bodyBytes, err := io.ReadAll(r.httpResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "[ai_resp]read resp body failed")
	}
	err = json.Unmarshal(bodyBytes, res)
	if err != nil {
		return nil, errors.Wrap(err, "[ai_resp]unmarshal resp body failed")
	}
	return res, nil
}

func (r *RespOpenAI) Recv() (*RespAIChatStream, error) {
	res := new(RespAIChatStream)
	if !r.IsStream {
		return nil, errors.New("[ai_resp]resp is not stream")
	}
	emptyLineCount := 0
	for {
		rawLine, err := r.respReader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return nil, errors.Wrap(err, "[ai_resp]read line failed")
		}
		noSpaceLine := bytes.TrimSpace(rawLine)
		if bytes.HasPrefix(noSpaceLine, errorPrefix) {
			if err != nil {
				return nil, errors.Wrap(err, string(noSpaceLine))
			}
			return nil, errors.New(string(noSpaceLine))
		}
		if !bytes.HasPrefix(noSpaceLine, dataPrefix) && err != io.EOF {
			emptyLineCount++
			if emptyLineCount > r.EmptyMsgLineLimit {
				return nil, errors.New("[ai_resp]empty line count exceed limit")
			}
			continue
		}
		noPrefixLine := bytes.TrimSpace(bytes.TrimPrefix(noSpaceLine, dataPrefix))
		if string(noPrefixLine) == "[DONE]" {
			return res, io.EOF
		}

		unmarshalErr := json.Unmarshal(noPrefixLine, res)
		if unmarshalErr != nil && err != io.EOF {
			return nil, errors.Wrap(unmarshalErr, "[ai_resp]unmarshal resp stream line failed")
		}

		if err == io.EOF {
			return res, io.EOF
		}

		return res, nil
	}
}

func (r *RespOpenAI) Close() error {
	return r.httpResp.Body.Close()
}
