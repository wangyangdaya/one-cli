package mcputil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Result holds the decoded text output from an MCP tool call.
type Result struct {
	Message string
}

// DecodeResponse parses a raw JSON-RPC response payload from an MCP server
// and extracts the text content from the result.
func DecodeResponse(payload []byte) (Result, error) {
	var response struct {
		Result map[string]any `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return Result{}, fmt.Errorf("decode MCP response: %w", err)
	}
	if response.Error != nil {
		return Result{}, fmt.Errorf("%s", strings.TrimSpace(response.Error.Message))
	}
	if content, ok := response.Result["content"].([]any); ok {
		for _, item := range content {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := entry["text"].(string); ok && strings.TrimSpace(text) != "" {
				return Result{Message: strings.TrimSpace(text)}, nil
			}
		}
	}
	rendered, err := json.Marshal(response.Result)
	if err != nil {
		return Result{}, err
	}
	return Result{Message: strings.TrimSpace(string(rendered))}, nil
}
