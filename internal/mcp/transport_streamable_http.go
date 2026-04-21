package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func discoverHTTPTools(name string, server ServerConfig) ([]Tool, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	_, sessionID, err := doRPCRequest(client, server, rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "opencli",
				"version": "dev",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("server %q initialize failed: %w", name, err)
	}
	if strings.TrimSpace(sessionID) != "" {
		if server.Headers == nil {
			server.Headers = make(map[string]string)
		}
		server.Headers["Mcp-Session-Id"] = strings.TrimSpace(sessionID)
	}

	result, _, err := doRPCRequest(client, server, rpcRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	})
	if err != nil {
		return nil, fmt.Errorf("server %q tools/list failed: %w", name, err)
	}

	return parseToolsResult(result)
}

func doRPCRequest(client *http.Client, server ServerConfig, payload rpcRequest) (map[string]any, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimSpace(server.URL), bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for name, value := range server.Headers {
		req.Header.Set(name, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	payloadBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("unexpected status %s", resp.Status)
	}

	responsePayload, err := parseStreamableHTTPResponse(resp.Header.Get("Content-Type"), payloadBytes)
	if err != nil {
		return nil, "", err
	}

	var response rpcResponse
	if err := json.Unmarshal(responsePayload, &response); err != nil {
		return nil, "", err
	}
	if response.Error != nil {
		return nil, "", fmt.Errorf("%s", strings.TrimSpace(response.Error.Message))
	}
	return response.Result, resp.Header.Get("Mcp-Session-Id"), nil
}

func parseStreamableHTTPResponse(contentType string, payload []byte) ([]byte, error) {
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		return extractSSEMessage(payload)
	}
	return payload, nil
}

func extractSSEMessage(payload []byte) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(dataLines) == 0 {
		return nil, fmt.Errorf("missing SSE data payload")
	}
	return []byte(strings.Join(dataLines, "\n")), nil
}
