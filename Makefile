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
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.57.2 golangci-lint run -v

release: build
	goreleaser --snapshot --clean

download_mods:
	go get -u ./...
	go mod tidy

update_mods: download_mods build

vulnerability-scan:
	docker-compose -f docker-compose.yml build
	docker-compose -f docker-compose.yml up -d
	docker-compose -f docker-compose.yml exec -T -e CGO_ENABLED=0 app go install golang.org/x/vuln/cmd/govulncheck@latest
	docker-compose -f docker-compose.yml exec -T -e CGO_ENABLED=0 app govulncheck ./...
	docker-compose -f docker-compose.yml down

.PHONY: build format lint download_mods vulnerability-scan update_mods release
.DEFAULT_GOAL := build
