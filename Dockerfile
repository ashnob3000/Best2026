FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o panel .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/panel .
COPY static ./static
COPY templates ./templates
EXPOSE 8080
CMD ["./panel"]
