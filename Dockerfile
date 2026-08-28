FROM golang:1.26-alpine AS builder
ENV GOTOOLCHAIN=local
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o llm-gateway ./cmd/gateway/

FROM alpine:3.21
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk --no-cache add ca-certificates
RUN adduser -D -u 1000 appuser
WORKDIR /app
COPY --from=builder /app/llm-gateway .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs/openapi.yaml ./docs/openapi.yaml
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/config.docker.yaml .
RUN mkdir -p /app/certs && chown -R appuser:appuser /app
USER appuser
EXPOSE 9090
ENTRYPOINT ["./llm-gateway"]
CMD ["--config", "config.yaml"]
