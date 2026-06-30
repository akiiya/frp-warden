SHELL := /bin/bash
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
ifeq ($(strip $(VERSION)),)
VERSION := dev
endif
LDFLAGS := -s -w -X github.com/fengheasia/frp-warden/internal/version.Version=$(VERSION)

.PHONY: build test vet release

build:
	@mkdir -p dist
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/frp-warden ./cmd/frp-warden

test:
	go test ./...

vet:
	go vet ./...

release:
	bash ./scripts/release.sh
