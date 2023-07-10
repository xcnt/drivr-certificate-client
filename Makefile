GOFILES := $(shell find . -name '*.go' -not -path "./vendor/*")

drivr-cert-client: $(GOFILES)
	go build ./cmd/drivr-cert-client

build: drivr-cert-client

.PHONY: build
.DEFAULT_GOAL := drivr-cert-client
