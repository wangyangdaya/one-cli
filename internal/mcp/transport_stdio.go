package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func discoverStdioTools(name string, server ServerConfig) ([]Tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, server.Command, server.Args...)
	cmd.Env = append(os.Environ(), envPairs(server.Env)...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("server %q stdin pipe: %w", name, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("server %q stdout pipe: %w", name, err)
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("server %q start: %w", name, err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	reader := bufio.NewReader(stdout)
	if _, err := doStdioRPC(stdin, reader, rpcRequest{
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
	}); err != nil {
		return nil, fmt.Errorf("server %q initialize failed: %w", name, err)
	}

	if err := sendStdioMessage(stdin, rpcRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}); err != nil {
		return nil, fmt.Errorf("server %q initialized notification failed: %w", name, err)
	}

	result, err := doStdioRPC(stdin, reader, rpcRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	})
	if err != nil {
		return nil, fmt.Errorf("server %q tools/list failed: %w", name, err)
	}

	return parseToolsResult(result)
}

func doStdioRPC(stdin io.Writer, reader *bufio.Reader, request rpcRequest) (map[string]any, error) {
	if err := sendStdioMessage(stdin, request); err != nil {
		return nil, err
	}
	responseBytes, err := readFrame(reader)
	if err != nil {
		return nil, err
	}
	var response rpcResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(response.Error.Message))
	}
	return response.Result, nil
}

// sendStdioMessage writes a JSON-RPC message as a newline-terminated JSON line
// (NDJSON format). This is compatible with both NDJSON-only servers and most
// LSP-style servers that also accept bare JSON input.
func sendStdioMessage(writer io.Writer, payload rpcRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	_, err = writer.Write(body)
	return err
}

// readFrame reads a single MCP response frame from the reader.
// It auto-detects two formats:
//   - NDJSON (bare JSON lines): "{...}\n"
//   - LSP-style framing: "Content-Length: N\r\n\r\n{...}"
//
// Detection: peek at the first non-blank byte. '{' means NDJSON, otherwise
// assume LSP Content-Length headers.
func readFrame(reader *bufio.Reader) ([]byte, error) {
	// Skip blank lines that some servers emit between messages.
	for {
		first, err := reader.Peek(1)
		if err != nil {
			return nil, err
		}
		if first[0] == '\n' || first[0] == '\r' {
			_, _ = reader.ReadByte()
			continue
		}
		break
	}

	first, err := reader.Peek(1)
	if err != nil {
		return nil, err
	}

	// NDJSON: line starts with '{', read until newline.
	if first[0] == '{' {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		return bytes.TrimSpace(line), nil
	}

	// LSP-style: read Content-Length headers, then fixed-size body.
	length := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			length = parsed
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(payload), nil
}

func envPairs(values map[string]string) []string {
	env := make([]string, 0, len(values))
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	return env
}
