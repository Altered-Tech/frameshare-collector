BINARY := collector
CMD    := ./cmd/collector
DIST   := dist

VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo dev-$$(git rev-parse --short HEAD))
LDFLAGS := -X main.version=$(VERSION)
BUILD   := CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)"

# BUMP forces which part of the last vX.Y.Z tag `make release` increments,
# e.g. `make release BUMP=minor`. Leave unset (the default) to have it
# decide on its own from Conventional Commits since the last tag: any
# "type!:" or "BREAKING CHANGE:" -> major, else any "feat:" -> minor, else
# any "fix:" -> patch, else patch.
BUMP :=

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
	@echo "  make release       tag the next semantic version and push it, triggering"
	@echo "                     the release workflow. Bump level (major/minor/patch) is"
	@echo "                     decided from Conventional Commits since the last tag;"
	@echo "                     override with BUMP=minor|major or the"
	@echo "                     release-{major,minor,patch} shorthands"
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
# (.github/workflows/build.yml). With BUMP unset, it decides major vs.
# minor vs. patch itself from Conventional Commits since the last tag (see
# the BUMP comment above); pass BUMP=minor/major, or use the
# release-{major,minor,patch} shorthands, to force a level instead.
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
	if [ "$$current" != "v0.0.0" ]; then \
		range="$$current..HEAD"; \
		if [ "$$(git rev-list $$range --count)" -eq 0 ]; then \
			echo "release: no commits since $$current; nothing to release" >&2; \
			exit 1; \
		fi; \
	else \
		range="HEAD"; \
	fi; \
	bump="$(BUMP)"; \
	if [ -z "$$bump" ]; then \
		log=$$(git log $$range --format='%s%n%b'); \
		if echo "$$log" | grep -Eq '^[a-zA-Z]+(\([^)]*\))?!:' || echo "$$log" | grep -q 'BREAKING CHANGE:'; then \
			bump=major; \
		elif echo "$$log" | grep -Eq '^feat(\([^)]*\))?:'; then \
			bump=minor; \
		elif echo "$$log" | grep -Eq '^fix(\([^)]*\))?:'; then \
			bump=patch; \
		else \
			echo "release: no Conventional Commits (feat:/fix:/BREAKING CHANGE) found since $$current; defaulting to patch" >&2; \
			bump=patch; \
		fi; \
	fi; \
	ver=$${current#v}; \
	major=$$(echo "$$ver" | cut -d. -f1); \
	minor=$$(echo "$$ver" | cut -d. -f2); \
	patch=$$(echo "$$ver" | cut -d. -f3); \
	case "$$bump" in \
		major) major=$$((major + 1)); minor=0; patch=0 ;; \
		minor) minor=$$((minor + 1)); patch=0 ;; \
		patch) patch=$$((patch + 1)) ;; \
		*) echo "release: BUMP must be major, minor, or patch (got '$$bump')" >&2; exit 1 ;; \
	esac; \
	next="v$$major.$$minor.$$patch"; \
	echo "release: $$current -> $$next ($$bump)"; \
	git tag -a "$$next" -m "$$next"; \
	git push origin "$$next"

release-major:
	@$(MAKE) --no-print-directory release BUMP=major

release-minor:
	@$(MAKE) --no-print-directory release BUMP=minor

release-patch:
	@$(MAKE) --no-print-directory release BUMP=patch
