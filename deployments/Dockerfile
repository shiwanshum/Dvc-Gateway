FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /gateway ./cmd/gateway/main.go

FROM alpine:latest
RUN apk add --no-cache tzdata
WORKDIR /app
COPY --from=builder /gateway /app/gateway
# Copy config files if needed
COPY config /app/config

CMD ["/app/gateway"]
