drivr-cert-client:
	go build ./cmd/drivr-cert-client

build: drivr-cert-client

.PHONY: build
.DEFAULT_GOAL := drivr-cert-client
