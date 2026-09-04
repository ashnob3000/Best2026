FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o panel .


FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates curl

# Install cloudflared
RUN curl -L \
    https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
    -o /usr/local/bin/cloudflared \
    && chmod +x /usr/local/bin/cloudflared

COPY --from=builder /app/panel .

COPY static ./static
COPY templates ./templates

RUN mkdir -p data

EXPOSE 8080

CMD ["./panel"]
