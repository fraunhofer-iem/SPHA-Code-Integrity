# Build stage
FROM golang:1.24.4-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all executables
RUN mkdir -p bin && \
    for cmd in singleRepo multiRepo groupResult transformResult; do \
        go build -v -ldflags="-s -w" -o "bin/spha-${cmd}" "./cmd/$cmd"; \
    done

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and git for cloning repositories
RUN apk --no-cache add ca-certificates git

# Set working directory
WORKDIR /app

# Copy built executables from builder stage
COPY --from=builder /app/bin/ ./bin/

# Create a non-root user
RUN adduser -D -s /bin/sh appuser
USER appuser

# Set PATH to include our binaries
ENV PATH="/app/bin:${PATH}"

# Default command - show available commands
CMD ["sh", "-c", "echo 'Available commands: spha-singleRepo, spha-multiRepo, spha-groupResult, spha-transformResult' && echo 'Usage: docker run <image> <command> [args]' && echo 'Example: docker run <image> spha-singleRepo -ownerAndRepo owner/repo -token your_token'"]