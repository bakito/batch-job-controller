FROM golang:1.14 as builder

WORKDIR /build

RUN apt-get update && apt-get install -y upx
COPY . .

ENV GOPROXY=https://goproxy.io \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN if GIT_TAG=$(git describe --tags --abbrev=0 --exact-match 2>/dev/null); then VERSION=${GIT_TAG}; else VERSION=$(git rev-parse --short HEAD); fi && \
    echo Building version ${VERSION} && \
    go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/batch-job-controller/version.Version=${VERSION}" -o batch-job-controller cmd/generic/main.go && \
    echo compress binary && \
    upx --ultra-brute batch-job-controller

# application image

FROM scratch

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080 8090 9153
WORKDIR /opt/go/
USER 1001
ENTRYPOINT ["/opt/go//batch-job-controller"]

COPY --from=builder /build/batch-job-controller /opt/go//batch-job-controller
