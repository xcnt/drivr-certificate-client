GOFILES := $(shell find . -name '*.go' -not -path "./vendor/*")

drivr-cert-client: lint $(GOFILES)
	go build ./cmd/drivr-cert-client

build: drivr-cert-client

format:
	go fmt ./...

lint: format
	golangci-lint run

.PHONY: build format lint
.DEFAULT_GOAL := drivr-cert-client
