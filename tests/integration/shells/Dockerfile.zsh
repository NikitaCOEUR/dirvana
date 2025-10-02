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

# Create test directory
RUN mkdir -p /test/project

# Create test config
RUN cat > /test/project/.dirvana.yml << 'EOF'
# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json
aliases:
  testcmd: echo "Dirvana alias works in zsh"

functions:
  testfunc: |
    echo "Dirvana function works: $1"

env:
  TEST_VAR: zsh-value
  DYNAMIC_VAR:
    sh: echo "dynamic-zsh"
EOF

# Authorize the directory
RUN /usr/local/bin/dirvana allow /test/project

# Install the hook
RUN /usr/local/bin/dirvana setup --shell zsh

# Copy test script
COPY tests/integration/shells/test-zsh.sh /test-zsh.sh
RUN chmod +x /test-zsh.sh

WORKDIR /test/project

CMD ["/test-zsh.sh"]
