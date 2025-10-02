FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Copy source code
WORKDIR /build
COPY . .

# Build static binary
RUN CGO_ENABLED=0 go build -o dirvana ./cmd/dirvana

# Final stage
FROM debian:bookworm-slim

# Install zsh and required tools
RUN apt-get update && \
    apt-get install -y --no-install-recommends zsh git && \
    rm -rf /var/lib/apt/lists/*

# Copy the dirvana binary from builder
COPY --from=builder /build/dirvana /usr/local/bin/dirvana
RUN chmod +x /usr/local/bin/dirvana

# Create test directory and copy config
RUN mkdir -p /test/project
COPY tests/integration/shells/test-config-zsh.yml /test/project/.dirvana.yml

# Authorize the directory
RUN /usr/local/bin/dirvana allow /test/project

# Install the hook
RUN /usr/local/bin/dirvana setup --shell zsh

# Copy test script
COPY tests/integration/shells/test-zsh.sh /test-zsh.sh
RUN chmod +x /test-zsh.sh

WORKDIR /test/project

CMD ["/test-zsh.sh"]
