package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const previewLimit = 12000

func Do(client *http.Client, req *http.Request) (*http.Response, []byte, error) {
	logRequest(req)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[one-cli][http] request_failed method=%s url=%s err=%v", req.Method, req.URL.String(), err)
		return nil, nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		log.Printf("[one-cli][http] response_read_failed method=%s url=%s status=%d err=%v", req.Method, req.URL.String(), resp.StatusCode, err)
		return nil, nil, err
	}
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	log.Printf(
		"[one-cli][http] response method=%s url=%s status=%d body=%s",
		req.Method,
		req.URL.String(),
		resp.StatusCode,
		preview(body),
	)

	return resp, body, nil
}

func logRequest(req *http.Request) {
	if req == nil || req.URL == nil {
		return
	}

	bodyPreview := "<empty>"
	if req.GetBody != nil {
		bodyReader, err := req.GetBody()
		if err == nil {
			body, readErr := io.ReadAll(bodyReader)
			_ = bodyReader.Close()
			if readErr == nil && len(body) > 0 {
				bodyPreview = preview(body)
			}
		}
	}

	log.Printf(
		"[one-cli][http] request method=%s url=%s query=%s headers=%s body=%s",
		req.Method,
		req.URL.String(),
		previewValues(req.URL.Query()),
		previewHeaders(req.Header),
		bodyPreview,
	)
}

func previewValues(values url.Values) string {
	if len(values) == 0 {
		return "<empty>"
	}

	payload := map[string]any{}
	for key, items := range values {
		switch len(items) {
		case 0:
			payload[key] = ""
		case 1:
			payload[key] = items[0]
		default:
			payload[key] = items
		}
	}

	data, err := marshalPreviewJSON(payload)
	if err != nil {
		return values.Encode()
	}
	return preview(data)
}

func previewHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return "<empty>"
	}

	payload := map[string]any{}
	for key, items := range headers {
		if !isSafeTraceHeader(key) {
			payload[key] = redactHeader(items)
			continue
		}

		switch len(items) {
		case 0:
			payload[key] = ""
		case 1:
			payload[key] = items[0]
		default:
			payload[key] = items
		}
	}

	data, err := marshalPreviewJSON(payload)
	if err != nil {
		return "<unavailable>"
	}
	return preview(data)
}

func isSafeTraceHeader(name string) bool {
	switch http.CanonicalHeaderKey(strings.TrimSpace(name)) {
	case "Accept", "Accept-Encoding", "Content-Length", "Content-Type", "Mcp-Protocol-Version", "User-Agent":
		return true
	default:
		return false
	}
}

func redactHeader(values []string) any {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return redactToken(values[0])
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, redactToken(value))
	}
	return out
}

func redactToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return "<redacted>"
}

func preview(body []byte) string {
	if len(body) == 0 {
		return "<empty>"
	}

	if json.Valid(body) {
		var compact bytes.Buffer
		if err := json.Compact(&compact, body); err == nil {
			text := compact.String()
			if len(text) > previewLimit {
				return fmt.Sprintf("%s...(truncated,len=%d)", text[:previewLimit], len(text))
			}
			return text
		}
	}

	text := strings.ReplaceAll(string(body), "\n", "\\n")
	if len(text) > previewLimit {
		return fmt.Sprintf("%s...(truncated,len=%d)", text[:previewLimit], len(text))
	}
	return text
}

func marshalPreviewJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}
