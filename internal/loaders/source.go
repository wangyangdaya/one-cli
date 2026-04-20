package loaders

import (
	"fmt"
	"strings"
)

type SourceKind string

const (
	SourceKindFile SourceKind = "file"
	SourceKindURL  SourceKind = "url"
)

func DetectSourceKind(input string) SourceKind {
	switch {
	case strings.HasPrefix(input, "http://"), strings.HasPrefix(input, "https://"):
		return SourceKindURL
	default:
		return SourceKindFile
	}
}

func Load(input string) ([]byte, error) {
	switch DetectSourceKind(input) {
	case SourceKindURL:
		return LoadHTTP(input)
	case SourceKindFile:
		return LoadFile(input)
	default:
		return nil, fmt.Errorf("unsupported source kind for %q", input)
	}
}
