package httpx

import "net/http"

func JSONHeaders(authToken string) http.Header {
	headers := http.Header{}
	if authToken != "" {
		headers.Set("Authorization", authToken)
	}
	headers.Set("Content-Type", "application/json")
	return headers
}
