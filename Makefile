GOFILES := $(shell find . -name '*.go' -not -path "./vendor/*")

drivr-certificate-client: lint $(GOFILES)
	go build -o $@ ./cmd/drivr-cert-client

build: drivr-certificate-client

format:
	go fmt ./...

lint: format
	golangci-lint run

release: build
	goreleaser --snapshot --clean

.PHONY: build format lint
.DEFAULT_GOAL := build
