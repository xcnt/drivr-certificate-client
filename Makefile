GRAPHQL_SCHEMA=api/schema.graphql
GOFILES := $(shell find . -name '*.go' -not -path "./vendor/*")

$(GRAPHQL_SCHEMA):
	curl localhost:8080/schema > $<

api/generated.go: $(GRAPHQL_SCHEMA) api/genqlient.yaml
	go generate ./api/...

drivr-certificate-client: $(GOFILES)
	go build -o $@ ./cmd/drivr-certificate-client

build: lint drivr-certificate-client

format: api/generated.go
	go fmt ./...

lint: format
	go tool github.com/golangci/golangci-lint/cmd/golangci-lint run -v

release: build
	goreleaser --snapshot --clean

download_mods:
	go get -u ./...
	go mod tidy

update_mods: download_mods build

vulnerability-scan:
	go tool golang.org/x/vuln/cmd/govulncheck ./...

.PHONY: build format lint download_mods vulnerability-scan update_mods release
.DEFAULT_GOAL := build
