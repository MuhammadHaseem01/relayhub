FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN GOTOOLCHAIN=local go mod download

COPY . .
RUN GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /relayhub ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /relayhub .

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./relayhub"]
