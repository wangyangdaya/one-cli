package loaders

import "os"

func LoadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
