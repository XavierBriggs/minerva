# Build stage
FROM golang:1.24-alpine AS builder

# Enable auto toolchain to allow Go to download newer versions if dependencies require them
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o minerva ./cmd/minerva

# Final stage - using Debian for full glibc (better TLS compatibility than Alpine's musl)
FROM debian:bookworm-slim

# Install ca-certificates and curl for HTTPS requests (curl needed for ESPN API)
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/minerva .

# Copy migrations and seed data
COPY --from=builder /app/infra/atlas/migrations ./migrations
COPY --from=builder /app/infra/atlas/seed ./seed

# Expose ports
EXPOSE 8080 8081

# Run the binary
CMD ["./minerva"]

