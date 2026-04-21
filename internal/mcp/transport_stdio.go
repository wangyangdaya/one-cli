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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	if err := sendStdioNotification(stdin, rpcRequest{
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
	if err := sendStdioNotification(stdin, request); err != nil {
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

func sendStdioNotification(writer io.Writer, payload rpcRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
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
