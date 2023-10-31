FROM golang:1.21.3-alpine as builder

ARG VERSION
ARG GITHUB_TOKEN

WORKDIR /app

COPY ./go.mod ./go.sum /app/

RUN apk add --no-cache git && \
	git config --global url."https://$GITHUB_TOKEN@github.com/xcnt".insteadOf https://github.com/xcnt && \
	go env -w GOPRIVATE=github.com/xcnt && \
	go mod download && \
	apk del git

COPY ./ /app/

RUN go build -ldflags="-X main.Version=${VERSION:=dev}" ./cmd/drivr-cert-client

FROM alpine:3.16 as runner

COPY --from=builder /app/drivr-cert-client /usr/local/bin/drivr-cert-client

ENTRYPOINT ["/usr/local/bin/drivr-cert-client"]
