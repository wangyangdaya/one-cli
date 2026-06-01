.PHONY: fmt test build build-host build-darwin-arm64 build-linux-amd64 build-windows-amd64 clean smoke

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X one-cli/internal/app.Version=$(VERSION)"

fmt:
	find . -name '*.go' -not -path './bin/*' -exec gofmt -w {} +

test:
	go test ./... -v

build:
	mkdir -p dist
	$(MAKE) build-host
	$(MAKE) build-darwin-arm64
	$(MAKE) build-linux-amd64
	$(MAKE) build-windows-amd64

build-host:
	mkdir -p dist
	go build $(LDFLAGS) -o dist/opencli ./cmd/opencli

build-darwin-arm64:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/opencli_darwin_arm64 ./cmd/opencli

build-linux-amd64:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/opencli_linux_amd64 ./cmd/opencli

build-windows-amd64:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/opencli_windows_amd64.exe ./cmd/opencli

smoke:
	./scripts/smoke.sh

clean:
	rm -rf bin dist tmp
