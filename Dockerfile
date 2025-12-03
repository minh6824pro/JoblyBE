# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /src

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/server ./cmd/JobblyBE

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy config file
COPY configs/config.yaml ./configs/

EXPOSE 8000
EXPOSE 9000

CMD ["./server", "-conf", "./configs"]
