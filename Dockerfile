FROM golang:1.24-alpine AS builder
WORKDIR /build

RUN apk update && apk add upx

COPY . .

ARG VERSION=main
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

RUN go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/batch-job-controller/version.Version=${VERSION}" -o batch-job-controller cmd/generic/main.go && \
    upx -q batch-job-controller

# application image

FROM scratch

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080 8090 9153
WORKDIR /opt/go/
USER 1001
ENTRYPOINT ["/opt/go//batch-job-controller"]

COPY --from=builder /build/batch-job-controller /opt/go//batch-job-controller
