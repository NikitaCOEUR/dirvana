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
FROM bash:5.2

# Install required tools
RUN apk add --no-cache git

# Copy the dirvana binary from builder
COPY --from=builder /build/dirvana /usr/local/bin/dirvana
RUN chmod +x /usr/local/bin/dirvana

# Create test directory and copy config
RUN mkdir -p /test/project
COPY tests/integration/shells/test-config-bash.yml /test/project/.dirvana.yml

# Authorize the directory and auto-approve shell commands
RUN /usr/local/bin/dirvana allow --auto-approve-shell /test/project

# Install the hook
RUN /usr/local/bin/dirvana setup --shell bash

# Copy test scripts
COPY tests/integration/shells/test-bash.sh /test-bash.sh
RUN chmod +x /test-bash.sh

WORKDIR /test/project

# Default to running standard tests, but allow override via environment variable
CMD ["/test-bash.sh"]
