package unit_test

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"one-cli/internal/httpx"
	"one-cli/internal/loaders"
)

func TestDetectSourceKind(t *testing.T) {
	tests := []struct {
		in   string
		want loaders.SourceKind
	}{
		{in: "./examples/petstore.yaml", want: loaders.SourceKindFile},
		{in: "https://example.com/openapi.json", want: loaders.SourceKindURL},
		{in: "http://example.com/openapi.yaml", want: loaders.SourceKindURL},
	}

	for _, tc := range tests {
		got := loaders.DetectSourceKind(tc.in)
		if got != tc.want {
			t.Fatalf("detect %q = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestLoadSourceReadsFileAndURL(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "petstore.yaml")
	want := []byte("openapi: 3.0.0\ninfo:\n  title: Demo\n")
	if err := os.WriteFile(filePath, want, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := loaders.Load(filePath)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("load file = %q want %q", string(got), string(want))
	}

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/openapi.yaml" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       ioNopCloser{Reader: bytes.NewReader([]byte("openapi: 3.0.0\ninfo:\n  title: Remote\n"))},
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	got, err = loaders.Load("https://example.com/openapi.yaml")
	if err != nil {
		t.Fatalf("load url: %v", err)
	}
	if string(got) != "openapi: 3.0.0\ninfo:\n  title: Remote\n" {
		t.Fatalf("load url = %q", string(got))
	}
}

func TestLoadHTTPAppliesDefaultRequestDeadline(t *testing.T) {
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	var hasDeadline bool
	var remaining time.Duration
	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		hasDeadline = ok
		if ok {
			remaining = time.Until(deadline)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       ioNopCloser{Reader: bytes.NewReader([]byte("openapi: 3.0.0\n"))},
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	if _, err := loaders.LoadHTTP("https://example.com/openapi.yaml"); err != nil {
		t.Fatalf("load HTTP: %v", err)
	}
	if !hasDeadline {
		t.Fatal("expected HTTP load request to have a deadline")
	}
	if remaining <= 0 || remaining > httpx.DefaultTimeout {
		t.Fatalf("request deadline remaining = %v, want within %v", remaining, httpx.DefaultTimeout)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type ioNopCloser struct {
	*bytes.Reader
}

func (r ioNopCloser) Close() error { return nil }
