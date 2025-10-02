# Shell Integration Tests

This directory contains Docker-based integration tests that validate Dirvana works correctly in real shell environments.

## Overview

Each shell has its own:
- **Dockerfile** - Creates a minimal environment with the shell installed
- **Test script** - Validates hooks, aliases, functions, and environment variables

## Supported Shells

- **Bash 5.2** (`Dockerfile.bash` + `test-bash.sh`)
- **Zsh 5.9** (`Dockerfile.zsh` + `test-zsh.sh`)
- **Fish** (`Dockerfile.fish` + `test-fish.sh`)

## Running Tests

### Using Task (recommended)

```bash
# Run all integration tests
task test-integration

# Run specific shell test
task test-integration-bash
task test-integration-zsh
task test-integration-fish

# Run all tests (unit + integration)
task test-all
```

### Using the script directly

```bash
# All shells at once
./run-tests.sh
```

### Manual Docker commands

```bash
# Build the binary first
cd ../../..
go build -o bin/dirvana ./cmd/dirvana

# Run specific shell test
docker build -t dirvana-test-bash -f tests/integration/shells/Dockerfile.bash .
docker run --rm dirvana-test-bash
```

## What Gets Tested

Each integration test validates:

1. **Hook Installation** - Verifies the shell hook is correctly installed
2. **Aliases** - Tests that aliases defined in config are available
3. **Functions** - Tests that shell functions work correctly
4. **Static Environment Variables** - Validates static env vars are set
5. **Dynamic Environment Variables** - Tests shell command execution for env vars
6. **Directory Change Trigger** - Ensures hooks trigger on `cd`

## CI/CD Integration

These tests run automatically in GitHub Actions via `.github/workflows/integration.yml`:
- Runs on every push and pull request
- Tests all shells in parallel
- Provides detailed output on failures

## Adding New Shell Tests

To add a new shell (e.g., `ksh`):

1. Create `Dockerfile.ksh`:
   ```dockerfile
   FROM <base-image-with-ksh>
   # Copy binary, create config, install hook
   COPY bin/dirvana /usr/local/bin/dirvana
   RUN dirvana setup --shell ksh
   COPY tests/integration/shells/test-ksh.sh /test-ksh.sh
   CMD ["/test-ksh.sh"]
   ```

2. Create `test-ksh.sh`:
   ```bash
   #!/bin/ksh
   # Test aliases, functions, env vars
   ```

3. Update `run-tests.sh` to include the new shell

4. Update `.github/workflows/integration.yml` with a new job
