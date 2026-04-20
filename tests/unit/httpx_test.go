package unit_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"one-cli/internal/httpx"
	runtimehttpx "one-cli/internal/runtime/httpx"
)

func TestJSONHeaders(t *testing.T) {
	headers := httpx.JSONHeaders("Basic token")

	if got := headers.Get("Authorization"); got != "Basic token" {
		t.Fatalf("expected Authorization header, got %q", got)
	}

	if got := headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}
}

func TestNewClientUsesDefaultTimeout(t *testing.T) {
	client := httpx.NewClient()
	if client.Timeout != httpx.DefaultTimeout {
		t.Fatalf("expected default timeout %v, got %v", httpx.DefaultTimeout, client.Timeout)
	}
}

func TestNewClientWithTimeoutOption(t *testing.T) {
	client := httpx.NewClientWithOptions(httpx.WithTimeout(5 * time.Second))
	if client.Timeout != 5*time.Second {
		t.Fatalf("expected custom timeout, got %v", client.Timeout)
	}
}

func TestDecodeJSONResponseClosesBody(t *testing.T) {
	closed := false
	resp := &http.Response{
		Body: &trackingReadCloser{
			Reader: bytes.NewReader([]byte(`{"name":"alice"}`)),
			closed: &closed,
		},
	}

	got, err := httpx.DecodeJSONResponse[map[string]string](resp)
	if err != nil {
		t.Fatalf("expected response decode to succeed, got %v", err)
	}
	if got["name"] != "alice" {
		t.Fatalf("unexpected decoded value: %#v", got)
	}
	if !closed {
		t.Fatal("expected response body to be closed")
	}
}

type trackingReadCloser struct {
	*bytes.Reader
	closed *bool
}

func (r *trackingReadCloser) Close() error {
	*r.closed = true
	return nil
}

func (r *trackingReadCloser) Read(p []byte) (int, error) {
	return r.Reader.Read(p)
}

var _ io.ReadCloser = (*trackingReadCloser)(nil)

func TestRuntimeHTTPTraceDisabledDoesNotLog(t *testing.T) {
	var logs bytes.Buffer
	runtimehttpx.SetTraceLogger(log.New(&logs, "", 0))
	runtimehttpx.SetTraceEnabled(false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/login", strings.NewReader(`{"email":"a"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret-token")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(`{"email":"a"}`)), nil
	}

	resp, err := runtimehttpx.NewClient().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	_ = resp.Body.Close()

	if logs.Len() != 0 {
		t.Fatalf("expected no trace logs, got %s", logs.String())
	}
}

func TestRuntimeHTTPTraceEnabledLogsAndRedactsAuthorization(t *testing.T) {
	var logs bytes.Buffer
	runtimehttpx.SetTraceLogger(log.New(&logs, "", 0))
	runtimehttpx.SetTraceEnabled(true)
	defer runtimehttpx.SetTraceEnabled(false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/login?verbose=true", strings.NewReader(`{"email":"a"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(`{"email":"a"}`)), nil
	}

	resp, err := runtimehttpx.NewClient().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	_ = resp.Body.Close()

	output := logs.String()
	for _, want := range []string{"[opencli][http] request", "method: POST", "[opencli][http] response", `"verbose": "true"`, `"Content-Type": "application/json"`} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing trace fragment %q in %s", want, output)
		}
	}
	if strings.Contains(output, "secret-token") {
		t.Fatalf("expected authorization to be redacted, got %s", output)
	}
}
