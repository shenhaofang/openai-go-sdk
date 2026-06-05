package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMakeResponseReqBytesStringInputMergesExtraAndDefaults(t *testing.T) {
	client := NewAIClient("test-key", "", ClientDefaultParamOption{
		MaxToken: 128,
	})

	reqBytes, err := client.MakeResponseReqBytes(OpenAIResponseParam{
		Model:   "gpt-5.5",
		Input:   "hello",
		Include: []string{"message.output_text.logprobs"},
		Extra: map[string]interface{}{
			"background": true,
		},
	})
	if err != nil {
		t.Fatalf("MakeResponseReqBytes returned error: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(reqBytes, &got); err != nil {
		t.Fatalf("request JSON is invalid: %v", err)
	}

	if got["model"] != "gpt-5.5" {
		t.Fatalf("model = %v, want gpt-5.5", got["model"])
	}
	if got["input"] != "hello" {
		t.Fatalf("input = %v, want hello", got["input"])
	}
	if got["max_output_tokens"] != float64(128) {
		t.Fatalf("max_output_tokens = %v, want 128", got["max_output_tokens"])
	}
	if got["background"] != true {
		t.Fatalf("background = %v, want true", got["background"])
	}

	include, ok := got["include"].([]interface{})
	if !ok || len(include) != 1 || include[0] != "message.output_text.logprobs" {
		t.Fatalf("include = %#v, want message.output_text.logprobs", got["include"])
	}
}

func TestMakeResponseReqBytesMessageInputContentTypes(t *testing.T) {
	client := NewAIClient("test-key", "")

	reqBytes, err := client.MakeResponseReqBytes(OpenAIResponseParam{
		Model: "gpt-5.5",
		Input: []ResponseInputMessage{
			{
				Role: ResponseRoleUser,
				Content: []ResponseInputContent{
					ResponseInputText{Text: "What is in this image?"},
					ResponseInputImage{ImageURL: "https://example.com/cat.png", Detail: "high"},
					ResponseInputFile{FileID: "file_123", Filename: "brief.pdf"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("MakeResponseReqBytes returned error: %v", err)
	}

	var got struct {
		Input []struct {
			Role    string                   `json:"role"`
			Content []map[string]interface{} `json:"content"`
		} `json:"input"`
	}
	if err := json.Unmarshal(reqBytes, &got); err != nil {
		t.Fatalf("request JSON is invalid: %v", err)
	}
	if len(got.Input) != 1 {
		t.Fatalf("input length = %d, want 1", len(got.Input))
	}
	if got.Input[0].Role != "user" {
		t.Fatalf("role = %q, want user", got.Input[0].Role)
	}
	if len(got.Input[0].Content) != 3 {
		t.Fatalf("content length = %d, want 3", len(got.Input[0].Content))
	}
	if got.Input[0].Content[0]["type"] != "input_text" || got.Input[0].Content[0]["text"] != "What is in this image?" {
		t.Fatalf("text content = %#v", got.Input[0].Content[0])
	}
	if got.Input[0].Content[1]["type"] != "input_image" || got.Input[0].Content[1]["image_url"] != "https://example.com/cat.png" || got.Input[0].Content[1]["detail"] != "high" {
		t.Fatalf("image content = %#v", got.Input[0].Content[1])
	}
	if got.Input[0].Content[2]["type"] != "input_file" || got.Input[0].Content[2]["file_id"] != "file_123" || got.Input[0].Content[2]["filename"] != "brief.pdf" {
		t.Fatalf("file content = %#v", got.Input[0].Content[2])
	}
}

func TestMakeResponseReqBytesRejectsInvalidParams(t *testing.T) {
	client := NewAIClient("test-key", "")

	if _, err := client.MakeResponseReqBytes(OpenAIResponseParam{Input: "hello"}); err == nil || !strings.Contains(err.Error(), "model is required") {
		t.Fatalf("empty model error = %v, want model is required", err)
	}

	_, err := client.MakeResponseReqBytes(OpenAIResponseParam{
		Model:              "gpt-5.5",
		Input:              "hello",
		PreviousResponseID: "resp_123",
		Conversation:       "conv_123",
	})
	if err == nil || !strings.Contains(err.Error(), "previous_response_id and conversation") {
		t.Fatalf("conversation conflict error = %v, want conflict", err)
	}

	_, err = client.MakeResponseReqBytes(OpenAIResponseParam{
		Model: "gpt-5.5",
		Input: "hello",
		Extra: map[string]interface{}{
			"model": "override",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "extra field conflicts") {
		t.Fatalf("extra conflict error = %v, want conflict", err)
	}
}

func TestMakeResponseRequestBuildsDefaultPathAndHeaders(t *testing.T) {
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %s, want /responses", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Fatalf("accept = %q, want text/event-stream", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_123","object":"response","status":"completed","output":[]}`))
	}))
	defer server.Close()

	client := NewAIClient("test-key", server.URL)
	req, err := client.MakeResponseRequest("", OpenAIResponseParam{
		Model:  "gpt-5.5",
		Input:  "hello",
		Stream: true,
	})
	if err != nil {
		t.Fatalf("MakeResponseRequest returned error: %v", err)
	}

	resp, err := req.GetResp(context.Background())
	if err != nil {
		t.Fatalf("GetResp returned error: %v", err)
	}
	defer resp.Close()

	if !sawRequest {
		t.Fatal("server did not receive request")
	}
	if !resp.IsStream {
		t.Fatal("response should be marked as stream")
	}
}

func TestMakeResponseRequestUsesCustomMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/responses" {
			t.Fatalf("path = %s, want /custom/responses", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"resp_123","object":"response","status":"completed","output":[]}`))
	}))
	defer server.Close()

	client := NewAIClient("test-key", server.URL)
	req, err := client.MakeResponseRequest("custom/responses", OpenAIResponseParam{
		Model: "gpt-5.5",
		Input: "hello",
	})
	if err != nil {
		t.Fatalf("MakeResponseRequest returned error: %v", err)
	}

	resp, err := req.GetResp(context.Background())
	if err != nil {
		t.Fatalf("GetResp returned error: %v", err)
	}
	defer resp.Close()
}
