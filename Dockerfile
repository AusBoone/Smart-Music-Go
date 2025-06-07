# Simple multi-stage build for Smart-Music-Go
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o smart-music-go cmd/web/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /src/smart-music-go ./
COPY ui ./ui
EXPOSE 4000
CMD ["./smart-music-go"]
