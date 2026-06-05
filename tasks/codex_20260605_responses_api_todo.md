# Responses API Todo

- [x] Review the approved Responses API design and current SDK structure.
- [x] Add failing tests for Responses request marshaling and validation.
- [x] Add failing tests for Responses HTTP request construction.
- [x] Add failing tests for non-streaming Responses parsing and `OutputText`.
- [x] Add failing tests for Responses SSE stream parsing.
- [x] Implement Responses request types and request builders.
- [x] Implement Responses response types and stream reader.
- [x] Run `gofmt` and `go test ./...`.
- [x] Record review and results.

## Review / Results

- Added `responses_request.go` with `OpenAIResponseParam`, typed input content, `Extra` merge conflict protection, request byte generation, and `MakeResponseRequest`.
- Added `responses_response.go` with core Responses response models, `OutputText`, preserved raw output payloads, and semantic SSE event parsing.
- Added request and response tests that first failed on missing Responses API symbols, then passed after implementation.
- Verification: `go test ./...` passes.
