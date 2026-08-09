BINARY := collector
CMD    := ./cmd/collector
DIST   := dist

VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo dev-$$(git rev-parse --short HEAD))
LDFLAGS := -X main.version=$(VERSION)
BUILD   := CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)"

.PHONY: all linux mac windows \
	linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 \
	vet test clean help

help:
	@echo "Targets:"
	@echo "  make linux    build linux/amd64 + linux/arm64"
	@echo "  make mac      build darwin/amd64 + darwin/arm64"
	@echo "  make windows  build windows/amd64"
	@echo "  make all      build every platform above"
	@echo "  make vet      go vet ./..."
	@echo "  make test     go test ./..."
	@echo "  make clean    remove $(DIST)/"
	@echo "Binaries are written to $(DIST)/, version: $(VERSION)"

all: linux mac windows

linux: linux-amd64 linux-arm64

mac: darwin-amd64 darwin-arm64

windows: windows-amd64

$(DIST):
	mkdir -p $(DIST)

linux-amd64: | $(DIST)
	GOOS=linux GOARCH=amd64 $(BUILD) -o $(DIST)/$(BINARY)-linux-amd64 $(CMD)

linux-arm64: | $(DIST)
	GOOS=linux GOARCH=arm64 $(BUILD) -o $(DIST)/$(BINARY)-linux-arm64 $(CMD)

darwin-amd64: | $(DIST)
	GOOS=darwin GOARCH=amd64 $(BUILD) -o $(DIST)/$(BINARY)-darwin-amd64 $(CMD)

darwin-arm64: | $(DIST)
	GOOS=darwin GOARCH=arm64 $(BUILD) -o $(DIST)/$(BINARY)-darwin-arm64 $(CMD)

windows-amd64: | $(DIST)
	GOOS=windows GOARCH=amd64 $(BUILD) -o $(DIST)/$(BINARY)-windows-amd64.exe $(CMD)

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf $(DIST)
