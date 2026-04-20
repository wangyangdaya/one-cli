package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func DecodeJSON[T any](data []byte) (T, error) {
	var out T
	err := json.Unmarshal(data, &out)
	return out, err
}

func DecodeJSONReader[T any](r io.Reader) (T, error) {
	var out T

	if r == nil {
		return out, errors.New("nil JSON reader")
	}

	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return out, err
	}

	return out, nil
}

func DecodeJSONResponse[T any](resp *http.Response) (T, error) {
	var out T

	if resp == nil {
		return out, errors.New("nil HTTP response")
	}
	if resp.Body == nil {
		return out, errors.New("nil HTTP response body")
	}

	defer resp.Body.Close()

	return DecodeJSONReader[T](resp.Body)
}
