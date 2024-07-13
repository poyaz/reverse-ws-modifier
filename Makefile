include Makefile.vars.mk

BINARY ?= reverse-ws-modifier
SOURCES = $(shell find . -name '*.go')
VERSION ?= $(shell git describe --tags --always --dirty --match "v*")
BUILD_FLAGS ?= -v
LDFLAGS ?= -X github.com/poyaz/reverse-ws-modifier/boot.Version=$(VERSION) -w -s
ARCH ?= amd64

IMAGE_TAG ?= latest
IMAGE_NAME ?= ghcr.io/poyaz/reverse-ws-modifier/$(BINARY)

build: build/$(BINARY)

build/$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: build-arm64
build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: build-amd64
build-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: build-arm/v7
build-arm/v7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: docker-build
docker-build:
	docker build \
		--tag $(IMAGE_NAME):$(IMAGE_TAG) \
		--label org.opencontainers.image.source=https://github.com/poyaz/reverse-ws-modifier \
		-f Dockerfile \
		.

.PHONY: docker-push
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: test
test: test
	@echo "Execute all unit test files"
	go test --tags=unit ./...
