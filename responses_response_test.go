package openai

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRespAIResponseGetAndOutputText(t *testing.T) {
	resp := &RespOpenAIResponse{
		httpResp: &http.Response{
			Body: io.NopCloser(strings.NewReader(`{
				"id": "resp_123",
				"object": "response",
				"created_at": 1741476542,
				"status": "completed",
				"model": "gpt-5.5",
				"output": [
					{
						"type": "message",
						"id": "msg_123",
						"status": "completed",
						"role": "assistant",
						"content": [
							{"type": "output_text", "text": "hello", "annotations": []},
							{"type": "refusal", "refusal": "blocked"}
						]
					},
					{
						"type": "message",
						"id": "msg_456",
						"status": "completed",
						"role": "assistant",
						"content": [
							{"type": "output_text", "text": " world", "annotations": []}
						]
					},
					{
						"type": "custom_tool_call",
						"id": "call_123",
						"call_id": "call_abc",
						"name": "lookup",
						"input": "{\"q\":\"go\"}"
					},
					{
						"type": "unknown_output",
						"id": "item_unknown",
						"payload": true
					}
				],
				"usage": {
					"input_tokens": 3,
					"output_tokens": 4,
					"total_tokens": 7
				}
			}`)),
		},
	}

	got, err := resp.Get()
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if got.ID != "resp_123" {
		t.Fatalf("id = %q, want resp_123", got.ID)
	}
	if got.OutputText() != "hello world" {
		t.Fatalf("OutputText = %q, want hello world", got.OutputText())
	}
	if got.Usage == nil || got.Usage.TotalTokens != 7 {
		t.Fatalf("usage = %#v, want total tokens 7", got.Usage)
	}
	if len(got.Output) != 4 {
		t.Fatalf("output length = %d, want 4", len(got.Output))
	}
	if got.Output[2].CustomToolCall == nil || got.Output[2].CustomToolCall.CallID != "call_abc" {
		t.Fatalf("custom tool call = %#v", got.Output[2].CustomToolCall)
	}
	if len(got.Output[3].Raw) == 0 {
		t.Fatal("unknown output raw payload should be preserved")
	}
}

func TestRespOpenAIResponseRecvParsesSemanticEvents(t *testing.T) {
	stream := strings.Join([]string{
		"event: response.created",
		`data: {"type":"response.created","sequence_number":0,"response":{"id":"resp_123","object":"response","status":"in_progress","output":[]}}`,
		"",
		`data: {"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"hel","sequence_number":1}`,
		"",
		"event: response.output_text.done",
		`data: {"type":"response.output_text.done","item_id":"msg_123","output_index":0,"content_index":0,"text":"hello","sequence_number":2}`,
		"",
		"event: response.completed",
		`data: {"type":"response.completed","sequence_number":3,"response":{"id":"resp_123","object":"response","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	resp := &RespOpenAIResponse{
		IsStream:          true,
		EmptyMsgLineLimit: 3,
		respReader:        bufio.NewReader(strings.NewReader(stream)),
	}

	created, err := resp.Recv()
	if err != nil {
		t.Fatalf("created Recv returned error: %v", err)
	}
	if created.Type != "response.created" || created.SequenceNumber != 0 || created.Response == nil || created.Response.ID != "resp_123" {
		t.Fatalf("created event = %#v", created)
	}

	delta, err := resp.Recv()
	if err != nil {
		t.Fatalf("delta Recv returned error: %v", err)
	}
	if delta.Type != "response.output_text.delta" || delta.Delta != "hel" || delta.SequenceNumber != 1 {
		t.Fatalf("delta event = %#v", delta)
	}

	done, err := resp.Recv()
	if err != nil {
		t.Fatalf("done Recv returned error: %v", err)
	}
	if done.Type != "response.output_text.done" || done.Text != "hello" {
		t.Fatalf("done event = %#v", done)
	}

	completed, err := resp.Recv()
	if err != nil {
		t.Fatalf("completed Recv returned error: %v", err)
	}
	if completed.Type != "response.completed" || completed.Response == nil || completed.Response.OutputText() != "hello" {
		t.Fatalf("completed event = %#v", completed)
	}

	if _, err := resp.Recv(); err != io.EOF {
		t.Fatalf("final Recv error = %v, want io.EOF", err)
	}
}

func TestRespOpenAIResponseRecvParsesErrorEvent(t *testing.T) {
	resp := &RespOpenAIResponse{
		IsStream: true,
		respReader: bufio.NewReader(strings.NewReader(strings.Join([]string{
			"event: error",
			`data: {"type":"error","code":"invalid_request","message":"bad request"}`,
			"",
		}, "\n"))),
	}

	event, err := resp.Recv()
	if err != nil {
		t.Fatalf("Recv returned error: %v", err)
	}
	if event.Type != "error" {
		t.Fatalf("event type = %q, want error", event.Type)
	}
	if event.Error == nil || event.Error.Message != "bad request" || event.Error.Code != "invalid_request" {
		t.Fatalf("event error = %#v", event.Error)
	}
}
