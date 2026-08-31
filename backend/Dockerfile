# syntax=docker/dockerfile:1

ARG ALPINE_MIRROR=https://mirror.yandex.ru/mirrors/alpine

FROM golang:1.24-alpine AS builder
ARG ALPINE_MIRROR
WORKDIR /src
RUN sed -i "s|https://dl-cdn.alpinelinux.org/alpine|${ALPINE_MIRROR}|g" /etc/apk/repositories \
    && apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org,https://goproxy.io,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api

FROM alpine:3.21
ARG ALPINE_MIRROR
RUN sed -i "s|https://dl-cdn.alpinelinux.org/alpine|${ALPINE_MIRROR}|g" /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 65532 nonroot
WORKDIR /app
COPY --from=builder /api /app/api
USER 65532:65532
ENV SERVER_PORT=8080
EXPOSE 8080
CMD ["/app/api"]
