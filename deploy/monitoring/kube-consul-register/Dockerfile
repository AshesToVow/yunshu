# 在 kube-consul-register 源码目录构建。
# 需与 go.mod 一致：Go >= 1.23.6。华为云 ddn-k8s 没有 golang:1.23 这个浮动 tag，
# 使用已同步的 1.23-bookworm（Go 1.23.10）。部署请用新 tag，并设置 imagePullPolicy: Always。
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.23-bookworm AS builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0 \
    GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out \
 && go build -ldflags="-s -w" -o /out/kube-consul-register . \
 && go version \
 && ( /out/kube-consul-register -version || true )

FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/gcr.io/iguazio/alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/kube-consul-register /kube-consul-register
WORKDIR /
ENTRYPOINT ["/kube-consul-register"]
