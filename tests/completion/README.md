# Completion Integration Tests

This directory contains integration tests for dirvana's completion system across multiple shells.

## Overview

The tests validate that dirvana's completion system works correctly with various tools (kubectl, terraform, aqua) in different shells (bash, zsh).

## Architecture

```
tests/completion/
├── Dockerfile                    # Multi-stage build with test tools
├── scripts/
│   ├── test_bash_completion.sh  # Expect script for bash
│   └── test_zsh_completion.sh   # Expect script for zsh
├── testdata/
│   └── .dirvana.yml             # Test configuration with aliases
└── shell_integration_test.go    # Go integration tests
```

## Running Tests

### Test all shells

```bash
task test-completion-all
```

### Test specific shell

```bash
# Bash only
TEST_SHELL=bash task test-completion

# Zsh only
TEST_SHELL=zsh task test-completion
```

### Interactive testing

Open an interactive shell in the test container:

```bash
# Bash shell
TEST_SHELL=bash task test-completion-shell

# Zsh shell
TEST_SHELL=zsh task test-completion-shell
```

## Adding New Tools

To add a new tool to test:

### 1. Add tool to Dockerfile

Either install via package manager:

```dockerfile
RUN apk add --no-cache my-tool
```

Or copy from official image:

```dockerfile
FROM official/my-tool:version AS my-tool
# ...
FROM base
COPY --from=my-tool /path/to/binary /usr/local/bin/
```

### 2. Add alias to testdata/.dirvana.yml

```yaml
aliases:
  m: my-tool
```

### 3. Add test case to shell_integration_test.go

```go
{
    name:           "my-tool-in-bash",
    shell:          "bash",
    alias:          "m",
    tool:           "my-tool",
    minCompletions: 10,
    shouldContain:  []string{"command1", "command2"},
},
```

## Test Structure

### Expect Scripts

The expect scripts automate shell interaction:

1. Spawn shell (bash/zsh)
2. Navigate to test directory
3. Load dirvana environment (`eval "$(dirvana export)"`)
4. Trigger completion with TAB
5. Capture and output results

### Go Integration Tests

The Go tests:

1. Call expect scripts for each tool/shell combination
2. Parse completion output
3. Validate:
   - Minimum number of completions
   - Presence of expected commands
   - Proper formatting

## Supported Tools

Currently tested tools:

- **kubectl** (Cobra protocol) - `k` alias
- **terraform** (BashComplete protocol) - `tf` alias
- **aqua** (UrfaveCli protocol) - `a` alias

Each tool represents a different completion protocol, ensuring comprehensive coverage.

## Troubleshooting

### Tests fail in container

Run interactive shell to debug:

```bash
TEST_SHELL=bash task test-completion-shell
```

Then manually test completion:

```bash
cd /app/tests/completion/testdata
eval "$(dirvana export)"
k <TAB><TAB>  # Test completion
```

### Tool not detected

Check completion detection:

```bash
# In container
dirvana completion --alias k -- kubectl
```

### Completions not matching

Compare with native tool:

```bash
# Alias completion
k <TAB><TAB>

# Native tool completion
kubectl <TAB><TAB>
```

They should match exactly.

## CI Integration

The tests run in Docker, making them suitable for CI/CD:

```yaml
# Example GitHub Actions
- name: Test completion
  run: task test-completion-all
```

## Go Version

The Dockerfile uses a configurable Go version via ARG:

```dockerfile
ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-alpine AS base
```

This can be overridden in Taskfile.yml via the `GO_VERSION` variable.
