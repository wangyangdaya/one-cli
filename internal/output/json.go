package output

import "encoding/json"

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessEnvelope struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ErrorEnvelope struct {
	OK      bool      `json:"ok"`
	Command string    `json:"command"`
	Error   ErrorBody `json:"error"`
}

func JSONSuccess(command, message string, data any) (string, error) {
	body, err := json.Marshal(SuccessEnvelope{
		OK:      true,
		Command: command,
		Message: message,
		Data:    data,
	})
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func JSONError(command, code, message string) (string, error) {
	body, err := json.Marshal(ErrorEnvelope{
		OK:      false,
		Command: command,
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
	if err != nil {
		return "", err
	}

	return string(body), nil
}
