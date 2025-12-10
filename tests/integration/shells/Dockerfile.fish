FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build static binary
RUN CGO_ENABLED=0 go build -o dirvana ./cmd/dirvana

# Final stage
FROM alpine:3.19

# Install fish shell and required tools
RUN apk add --no-cache fish git bash

# Copy the dirvana binary from builder
COPY --from=builder /build/dirvana /usr/local/bin/dirvana
RUN chmod +x /usr/local/bin/dirvana

# Create test directory and copy config
RUN mkdir -p /test/project
COPY tests/integration/shells/test-config-fish.yml /test/project/.dirvana.yml

# Authorize the directory and auto-approve shell commands
RUN /usr/local/bin/dirvana allow --auto-approve-shell /test/project

# Install the hook
RUN /usr/local/bin/dirvana setup --shell fish

# Copy test scripts
COPY tests/integration/shells/test-fish.sh /test-fish.sh
RUN chmod +x /test-fish.sh

WORKDIR /test/project

# Default to running standard tests
CMD ["/test-fish.sh"]
