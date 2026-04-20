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

	data, err := json.Marshal(payload)
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
		if strings.EqualFold(key, "Authorization") {
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

	data, err := json.Marshal(payload)
	if err != nil {
		return "<unavailable>"
	}
	return preview(data)
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
	if len(trimmed) <= 8 {
		return "***"
	}
	return trimmed[:4] + "***" + trimmed[len(trimmed)-2:]
}

func preview(body []byte) string {
	if len(body) == 0 {
		return "<empty>"
	}

	if json.Valid(body) {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err == nil {
			text := pretty.String()
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
