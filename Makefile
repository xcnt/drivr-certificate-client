GOFILES := $(shell find . -name '*.go' -not -path "./vendor/*")

drivr-certificate-client: lint $(GOFILES)
	go build -o $@ ./cmd/drivr-certificate-client

build: drivr-certificate-client

format:
	go fmt ./...

lint: format
	golangci-lint run

release: build
	goreleaser --snapshot --clean

vulnerability-scan:
	docker-compose -f docker-compose.yml build
	docker-compose -f docker-compose.yml up -d
	docker-compose -f docker-compose.yml exec -T -e CGO_ENABLED=0 app go install golang.org/x/vuln/cmd/govulncheck@latest
	docker-compose -f docker-compose.yml exec -T -e CGO_ENABLED=0 app govulncheck ./...
	docker-compose -f docker-compose.yml down

.PHONY: build format lint
.DEFAULT_GOAL := build
