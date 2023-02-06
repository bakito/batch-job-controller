FROM golang:1.20 as builder
ARG upx_brute="--ultra-brute"
WORKDIR /build

RUN apt-get update && apt-get install -y upx
COPY . .

ARG VERSION=main
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/batch-job-controller/version.Version=${VERSION}" -o batch-job-controller cmd/generic/main.go && \
    upx ${upx_brute} -q batch-job-controller

# application image

FROM scratch

LABEL maintainer="bakito <github@bakito.ch>"
EXPOSE 8080 8090 9153
WORKDIR /opt/go/
USER 1001
ENTRYPOINT ["/opt/go//batch-job-controller"]

COPY --from=builder /build/batch-job-controller /opt/go//batch-job-controller
