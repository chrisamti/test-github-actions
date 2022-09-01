.PHONY: all lint help

# VERSION=`git describe --tags 2>/dev/null || echo ""`
# BUILDTIME=`date +%FT%T%z`
# BUILDHASH=`git rev-parse --short HEAD`
# BUILDDIRTY=`[ $$(git status --short | wc -c) -ne 0 ] && echo "-dirty"`

# LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.buildtime=${BUILDTIME} -X main.buildhash=${BUILDHASH}${BUILDDIRTY}"

all: lint test ## lint check and test package

lint: ## Lint the project
	golangci-lint run ./...

test:
	go test -race -v ./...

help: ## Print all possible targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
