FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates
ENV GOPROXY=https://goproxy.cn,direct \
    GOTOOLCHAIN=auto \
    CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/gorocksdb-server ./cmd/server

FROM alpine:3.21
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo Asia/Shanghai > /etc/timezone
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /out/gorocksdb-server /usr/local/bin/gorocksdb-server
EXPOSE 8080
ENTRYPOINT ["gorocksdb-server"]
