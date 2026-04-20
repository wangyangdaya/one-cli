package httpx

import (
	"net/http"
	"time"
)

const DefaultTimeout = 30 * time.Second

type ClientOption func(*http.Client)

func NewClient() *http.Client {
	return NewClientWithOptions()
}

func NewClientWithOptions(opts ...ClientOption) *http.Client {
	client := &http.Client{Timeout: DefaultTimeout}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(client *http.Client) {
		client.Timeout = timeout
	}
}
