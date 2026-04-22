package config

import "os"

func Lookup(key string) string {
	return os.Getenv(key)
}
