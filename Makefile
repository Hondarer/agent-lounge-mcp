VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/hondarer/agent-lounge-mcp/internal/version.Version=$(VERSION)
# CGO_ENABLED=0: the SQLite driver is pure Go, so every target builds on one machine.
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)"

TARGETS := linux-amd64 linux-arm64 windows-amd64 windows-arm64 darwin-amd64 darwin-arm64

# Bundled binaries for standalone operation of the agent-lounge-connect skill
SKILL_DIR := .agents/skills/agent-lounge-connect
SKILL_BIN := $(SKILL_DIR)/bin

.PHONY: all build dist skill-bin test vet clean $(TARGETS)

all: vet test build

# Build for the local host OS/architecture
build:
	$(GOBUILD) -o dist/agent-lounge-mcp$(if $(filter windows,$(shell go env GOOS)),.exe) ./cmd/agent-lounge-mcp

# Cross-compile all target binaries and populate the skill bin/ directory
dist: $(TARGETS)
	$(MAKE) skill-bin

skill-bin:
	mkdir -p $(SKILL_BIN)
	cp dist/agent-lounge-mcp-*-* $(SKILL_BIN)/
	echo $(VERSION) > $(SKILL_BIN)/VERSION

$(TARGETS):
	GOOS=$(word 1,$(subst -, ,$@)) GOARCH=$(word 2,$(subst -, ,$@)) \
		$(GOBUILD) -o dist/agent-lounge-mcp-$@$(if $(findstring windows,$@),.exe) ./cmd/agent-lounge-mcp

test:
	go test ./...

vet:
	go vet ./...
	GOOS=windows go vet ./...
	GOOS=darwin go vet ./...

clean:
	rm -rf dist $(SKILL_BIN)
