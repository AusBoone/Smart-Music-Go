# Simple multi-stage build for Smart-Music-Go
FROM golang:1.22-alpine AS builder
WORKDIR /src
# Install build tools and Node for the frontend build
RUN apk add --no-cache build-base sqlite-dev nodejs npm
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build the React frontend
RUN cd ui/frontend && npm install && npm run build
# Build the Go binary
RUN go build -o smart-music-go cmd/web/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /src/smart-music-go ./
# Copy templates and static assets
COPY --from=builder /src/ui/templates ./ui/templates
COPY --from=builder /src/ui/static ./ui/static
# Copy the built frontend
COPY --from=builder /src/ui/frontend/dist ./ui/frontend/dist
EXPOSE 4000
CMD ["./smart-music-go"]
