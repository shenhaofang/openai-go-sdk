package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type RespAIResponse struct {
	ID                 string                     `json:"id"`
	Object             string                     `json:"object"`
	CreatedAt          int64                      `json:"created_at"`
	Status             string                     `json:"status"`
	CompletedAt        int64                      `json:"completed_at,omitempty"`
	Model              string                     `json:"model"`
	Output             []ResponseOutputItem       `json:"output,omitempty"`
	Usage              *ResponseUsage             `json:"usage,omitempty"`
	Error              *AIError                   `json:"error,omitempty"`
	IncompleteDetails  *ResponseIncompleteDetails `json:"incomplete_details,omitempty"`
	PreviousResponseID interface{}                `json:"previous_response_id,omitempty"`
	ParallelToolCalls  bool                       `json:"parallel_tool_calls,omitempty"`
	Store              bool                       `json:"store,omitempty"`
	Temperature        float64                    `json:"temperature,omitempty"`
	TopP               float64                    `json:"top_p,omitempty"`
	Truncation         string                     `json:"truncation,omitempty"`
	Metadata           map[string]interface{}     `json:"metadata,omitempty"`
	Raw                json.RawMessage            `json:"-"`
}

func (r *RespAIResponse) UnmarshalJSON(input []byte) error {
	type respAlias RespAIResponse
	var alias respAlias
	if err := json.Unmarshal(input, &alias); err != nil {
		return err
	}
	*r = RespAIResponse(alias)
	r.Raw = append(r.Raw[:0], input...)
	return nil
}

func (r *RespAIResponse) OutputText() string {
	if r == nil {
		return ""
	}
	var builder strings.Builder
	for _, item := range r.Output {
		contents := item.Content
		if item.Message != nil {
			contents = item.Message.Content
		}
		for _, content := range contents {
			if content.Type == "output_text" {
				builder.WriteString(content.Text)
			}
		}
	}
	return builder.String()
}

type ResponseIncompleteDetails struct {
	Reason string `json:"reason,omitempty"`
}

type ResponseUsage struct {
	InputTokens         int                         `json:"input_tokens"`
	InputTokensDetails  *ResponseInputTokenDetails  `json:"input_tokens_details,omitempty"`
	OutputTokens        int                         `json:"output_tokens"`
	OutputTokensDetails *ResponseOutputTokenDetails `json:"output_tokens_details,omitempty"`
	TotalTokens         int                         `json:"total_tokens"`
}

type ResponseInputTokenDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type ResponseOutputTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type ResponseOutputItem struct {
	Type           string                  `json:"type"`
	ID             string                  `json:"id,omitempty"`
	Status         string                  `json:"status,omitempty"`
	Role           ResponseRole            `json:"role,omitempty"`
	Content        []ResponseOutputContent `json:"content,omitempty"`
	Message        *ResponseOutputMessage  `json:"-"`
	FunctionCall   *ResponseFunctionCall   `json:"-"`
	CustomToolCall *ResponseCustomToolCall `json:"-"`
	Raw            json.RawMessage         `json:"-"`
}

func (i *ResponseOutputItem) UnmarshalJSON(input []byte) error {
	type itemAlias ResponseOutputItem
	var base itemAlias
	if err := json.Unmarshal(input, &base); err != nil {
		return err
	}
	*i = ResponseOutputItem(base)
	i.Raw = append(i.Raw[:0], input...)

	switch i.Type {
	case "message":
		var msg ResponseOutputMessage
		if err := json.Unmarshal(input, &msg); err != nil {
			return err
		}
		i.Message = &msg
		i.ID = msg.ID
		i.Status = msg.Status
		i.Role = msg.Role
		i.Content = msg.Content
	case "function_call":
		var call ResponseFunctionCall
		if err := json.Unmarshal(input, &call); err != nil {
			return err
		}
		i.FunctionCall = &call
	case "custom_tool_call":
		var call ResponseCustomToolCall
		if err := json.Unmarshal(input, &call); err != nil {
			return err
		}
		i.CustomToolCall = &call
	}
	return nil
}

type ResponseOutputMessage struct {
	ID      string                  `json:"id,omitempty"`
	Type    string                  `json:"type"`
	Status  string                  `json:"status,omitempty"`
	Role    ResponseRole            `json:"role,omitempty"`
	Content []ResponseOutputContent `json:"content,omitempty"`
	Phase   string                  `json:"phase,omitempty"`
}

type ResponseOutputContent struct {
	Type        string            `json:"type"`
	Text        string            `json:"text,omitempty"`
	Refusal     string            `json:"refusal,omitempty"`
	Annotations []json.RawMessage `json:"annotations,omitempty"`
	Logprobs    []json.RawMessage `json:"logprobs,omitempty"`
	Raw         json.RawMessage   `json:"-"`
}

func (c *ResponseOutputContent) UnmarshalJSON(input []byte) error {
	type contentAlias ResponseOutputContent
	var alias contentAlias
	if err := json.Unmarshal(input, &alias); err != nil {
		return err
	}
	*c = ResponseOutputContent(alias)
	c.Raw = append(c.Raw[:0], input...)
	return nil
}

type ResponseFunctionCall struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Status    string `json:"status,omitempty"`
}

type ResponseCustomToolCall struct {
	ID     string `json:"id,omitempty"`
	Type   string `json:"type"`
	CallID string `json:"call_id,omitempty"`
	Name   string `json:"name,omitempty"`
	Input  string `json:"input,omitempty"`
	Status string `json:"status,omitempty"`
}

type RespOpenAIResponse struct {
	IsStream          bool
	EmptyMsgLineLimit int
	httpResp          *http.Response
	respReader        *bufio.Reader
}

func (r *RespOpenAIResponse) Get() (*RespAIResponse, error) {
	res := new(RespAIResponse)
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

func (r *RespOpenAIResponse) HttpStatus() int {
	if r.httpResp == nil {
		return 0
	}
	return r.httpResp.StatusCode
}

type RespAIResponseStreamEvent struct {
	Type           string          `json:"type"`
	SequenceNumber int             `json:"sequence_number,omitempty"`
	ItemID         string          `json:"item_id,omitempty"`
	OutputIndex    int             `json:"output_index,omitempty"`
	ContentIndex   int             `json:"content_index,omitempty"`
	Delta          string          `json:"delta,omitempty"`
	Text           string          `json:"text,omitempty"`
	Response       *RespAIResponse `json:"response,omitempty"`
	Error          *AIError        `json:"error,omitempty"`
	Raw            json.RawMessage `json:"-"`
}

func (e *RespAIResponseStreamEvent) UnmarshalJSON(input []byte) error {
	var base struct {
		Type           string          `json:"type"`
		SequenceNumber int             `json:"sequence_number,omitempty"`
		ItemID         string          `json:"item_id,omitempty"`
		OutputIndex    int             `json:"output_index,omitempty"`
		ContentIndex   int             `json:"content_index,omitempty"`
		Delta          string          `json:"delta,omitempty"`
		Text           string          `json:"text,omitempty"`
		Response       *RespAIResponse `json:"response,omitempty"`
		Error          *AIError        `json:"error,omitempty"`
		Code           string          `json:"code,omitempty"`
		Message        string          `json:"message,omitempty"`
		Param          interface{}     `json:"param,omitempty"`
	}
	if err := json.Unmarshal(input, &base); err != nil {
		return err
	}
	e.Type = base.Type
	e.SequenceNumber = base.SequenceNumber
	e.ItemID = base.ItemID
	e.OutputIndex = base.OutputIndex
	e.ContentIndex = base.ContentIndex
	e.Delta = base.Delta
	e.Text = base.Text
	e.Response = base.Response
	e.Error = base.Error
	e.Raw = append(e.Raw[:0], input...)
	if e.Error == nil && (base.Type == "error" || base.Code != "" || base.Message != "") {
		e.Error = &AIError{
			Code:    base.Code,
			Type:    base.Type,
			Message: base.Message,
			Param:   base.Param,
		}
	}
	return nil
}

func (r *RespOpenAIResponse) Recv() (*RespAIResponseStreamEvent, error) {
	if !r.IsStream {
		return nil, errors.New("[ai_resp]resp is not stream")
	}
	if r.respReader == nil {
		return nil, errors.New("[ai_resp]stream reader is nil")
	}

	eventName, data, err := r.readResponseSSE()
	if err != nil {
		return nil, err
	}
	if bytes.Equal(data, []byte("[DONE]")) {
		return &RespAIResponseStreamEvent{Type: eventName, Raw: data}, io.EOF
	}

	event := new(RespAIResponseStreamEvent)
	if err := json.Unmarshal(data, event); err != nil {
		return nil, errors.Wrap(err, "[ai_resp]unmarshal resp stream event failed")
	}
	if event.Type == "" {
		event.Type = eventName
	}
	if event.Type == "error" && event.Error == nil {
		event.Error = &AIError{Type: event.Type}
	}
	return event, nil
}

func (r *RespOpenAIResponse) readResponseSSE() (string, []byte, error) {
	emptyLineCount := 0
	eventName := ""
	dataLines := make([]string, 0, 1)

	for {
		rawLine, err := r.respReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", nil, errors.Wrap(err, "[ai_resp]read line failed")
		}

		line := strings.TrimRight(rawLine, "\r\n")
		if strings.TrimSpace(line) == "" {
			if len(dataLines) > 0 {
				return eventName, []byte(strings.Join(dataLines, "\n")), nil
			}
			if err == io.EOF {
				return "", nil, io.EOF
			}
			emptyLineCount++
			if emptyLineCount > r.EmptyMsgLineLimit && r.EmptyMsgLineLimit > 0 {
				return "", nil, errors.New("[ai_resp]empty line count exceed limit")
			}
			continue
		}
		emptyLineCount = 0

		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if err == io.EOF {
			if len(dataLines) > 0 {
				return eventName, []byte(strings.Join(dataLines, "\n")), nil
			}
			return "", nil, io.EOF
		}
	}
}

func (r *RespOpenAIResponse) Close() error {
	if r == nil || r.httpResp == nil || r.httpResp.Body == nil {
		return nil
	}
	return r.httpResp.Body.Close()
}
