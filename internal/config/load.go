package config

import (
	"errors"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

func Load() (Env, error) {
	if _, err := BootstrapDotenv(); err != nil {
		return Env{}, err
	}

	return ParseFromLookup(os.LookupEnv)
}

func BootstrapDotenv() (bool, error) {
	if err := godotenv.Load(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func ParseFromLookup(lookup func(string) (string, bool)) (Env, error) {
	env := Env{
		JobNo: lookupValue(lookup, "ONE_AI_JOB_NO"),
		One: OneConfig{
			AuthToken: lookupValue(lookup, "ONE_AI_AUTH_TOKEN"),
		},
		Endpoints: EndpointConfig{
			LeaveListURL:   lookupValue(lookup, "ONE_AI_LEAVE_LIST_URL"),
			LeaveCheckURL:  lookupValue(lookup, "ONE_AI_LEAVE_CHECK_URL"),
			LeaveCreateURL: lookupValue(lookup, "ONE_AI_LEAVE_CREATE_URL"),
		},
	}

	return env, nil
}

func ValidateOneAuth(env Env) error {
	if env.One.AuthToken == "" {
		return errors.New("missing ONE_AI_AUTH_TOKEN")
	}

	return nil
}

func lookupValue(lookup func(string) (string, bool), key string) string {
	if value, ok := lookup(key); ok {
		return value
	}

	return ""
}
