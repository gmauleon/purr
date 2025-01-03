SHELL := /bin/bash

DISTDIR := dist
BINDIR := bin
PATH := $(BINDIR):$(PATH)

GOLANGCI_LINT_VERSION := v1.63.3

all: lint build

golangci-lint:
	@[[ -x $(BINDIR)/golangci-lint ]] || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)

.PHONY: clean
clean:
	rm -rf $(BINDIR)
	rm -rf $(DISTDIR)

.PHONY: lint
lint: golangci-lint
	golangci-lint run

.PHONY: build
build: lint
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags=-static" -o $(DISTDIR)/purr
	chmod +x ${DISTDIR}/purr

.PHONY: image
image:
	docker build . -t gmauleon/purr