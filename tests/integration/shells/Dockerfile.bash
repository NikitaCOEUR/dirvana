FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Copy source code
WORKDIR /build
COPY . .

# Build static binary
RUN CGO_ENABLED=0 go build -o dirvana ./cmd/dirvana

# Final stage
FROM bash:5.2

# Install required tools
RUN apk add --no-cache git

# Copy the dirvana binary from builder
COPY --from=builder /build/dirvana /usr/local/bin/dirvana
RUN chmod +x /usr/local/bin/dirvana

# Create test directory
RUN mkdir -p /test/project

# Create test config
RUN cat > /test/project/.dirvana.yml << 'EOF'
# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json
aliases:
  testcmd: echo "Dirvana alias works in bash"

functions:
  testfunc: |
    echo "Dirvana function works: $1"

env:
  TEST_VAR: bash-value
  DYNAMIC_VAR:
    sh: echo "dynamic-bash"
EOF

# Authorize the directory
RUN /usr/local/bin/dirvana allow /test/project

# Install the hook
RUN /usr/local/bin/dirvana setup --shell bash

# Copy test script
COPY tests/integration/shells/test-bash.sh /test-bash.sh
RUN chmod +x /test-bash.sh

WORKDIR /test/project

CMD ["/test-bash.sh"]
