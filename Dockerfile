# Dockerfile
FROM golang:1.22 AS builder

# Set working dir inside container
WORKDIR /app

# Copy everything
COPY . .

# Download deps
RUN go mod download

# Build (but we won’t use the binary for dev, just ensure deps are ready)
RUN go build ./...

# Default command (overridden by docker-compose `command:`)
CMD ["go", "run", "cmd/bootstrap/main.go"]
