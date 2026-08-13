BINARY := collector
CMD    := ./cmd/collector
DIST   := dist

VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo dev-$$(git rev-parse --short HEAD))
LDFLAGS := -X main.version=$(VERSION)
BUILD   := CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)"

# BUMP selects which part of the last vX.Y.Z tag `make release` increments;
# override on the command line, e.g. `make release BUMP=minor`.
BUMP := patch

.PHONY: all linux mac windows \
	linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 \
	vet test clean help \
	release release-major release-minor release-patch

help:
	@echo "Targets:"
	@echo "  make linux         build linux/amd64 + linux/arm64"
	@echo "  make mac           build darwin/amd64 + darwin/arm64"
	@echo "  make windows       build windows/amd64"
	@echo "  make all           build every platform above"
	@echo "  make vet           go vet ./..."
	@echo "  make test          go test ./..."
	@echo "  make clean         remove $(DIST)/"
	@echo "  make release       tag the next patch version (vX.Y.Z+1) and push it,"
	@echo "                     triggering the release workflow. BUMP=minor|major to"
	@echo "                     bump a different part; release-{major,minor,patch} are"
	@echo "                     shorthands for the same"
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

# release tags the next semantic version after the latest vX.Y.Z tag and
# pushes it, which triggers the tag-triggered build/release workflow
# (.github/workflows/build.yml). Defaults to a patch bump; override with
# BUMP=minor or BUMP=major, or use the release-{major,minor,patch}
# shorthands below.
release:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "release: working tree not clean; commit or stash changes first" >&2; \
		exit 1; \
	fi
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "main" ]; then \
		echo "release: refusing to tag from branch '$$branch' (expected 'main')" >&2; \
		exit 1; \
	fi
	@git fetch origin main --quiet
	@if [ "$$(git rev-parse HEAD)" != "$$(git rev-parse origin/main)" ]; then \
		echo "release: local main is not in sync with origin/main; pull or push first" >&2; \
		exit 1; \
	fi
	@$(MAKE) --no-print-directory vet test
	@current=$$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' | sort -V | tail -1); \
	current=$${current:-v0.0.0}; \
	ver=$${current#v}; \
	major=$$(echo "$$ver" | cut -d. -f1); \
	minor=$$(echo "$$ver" | cut -d. -f2); \
	patch=$$(echo "$$ver" | cut -d. -f3); \
	case "$(BUMP)" in \
		major) major=$$((major + 1)); minor=0; patch=0 ;; \
		minor) minor=$$((minor + 1)); patch=0 ;; \
		patch) patch=$$((patch + 1)) ;; \
		*) echo "release: BUMP must be major, minor, or patch (got '$(BUMP)')" >&2; exit 1 ;; \
	esac; \
	next="v$$major.$$minor.$$patch"; \
	echo "release: $$current -> $$next"; \
	git tag -a "$$next" -m "$$next"; \
	git push origin "$$next"

release-major:
	@$(MAKE) --no-print-directory release BUMP=major

release-minor:
	@$(MAKE) --no-print-directory release BUMP=minor

release-patch:
	@$(MAKE) --no-print-directory release BUMP=patch
