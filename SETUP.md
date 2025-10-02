# Dirvana Setup Guide

## Quick Setup

### 1. Install Dependencies

Task is managed via aqua.yaml. If you have [aqua](https://aquaproj.github.io/) installed:

```bash
aqua install
```

Otherwise, install Task manually:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

### 2. Build and Install Dirvana

```bash
# Install dependencies and build
task build

# Or install directly to $GOPATH/bin
task install
```

### 3. Setup Shell Hook

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
# Load Dirvana hook
source /path/to/dirvana/hooks/dirvana.sh
```

Then reload your shell:

```bash
source ~/.bashrc  # or ~/.zshrc
```

## Verification

### Test the CLI

```bash
# Check version
dirvana --version

# Show help
dirvana --help

# Create sample config
cd ~/myproject
dirvana init

# Authorize the directory
dirvana allow

# Export shell code (for testing)
dirvana export
```

### Test the Hook

```bash
# Navigate to your project directory
cd ~/myproject

# Your aliases, functions, and env vars should now be loaded!
# Test them:
ll  # should execute ls -lah
echo $PROJECT_NAME  # should print myproject
```

## Running Tests

```bash
# Run all tests
task test

# Run tests with coverage
task test-coverage

# Run tests with verbose output
task test-verbose
```

## Development Workflow

```bash
# Format code
task fmt

# Run linter (requires golangci-lint)
task lint

# Clean build artifacts
task clean

# Run all verification (format + test)
task verify
```

## Project Structure

```
dirvana/
├── cmd/dirvana/          # CLI entry point
├── internal/
│   ├── auth/             # Authorization system
│   ├── cache/            # Cache management
│   ├── config/           # Configuration loading (Koanf)
│   ├── logger/           # Logging (Zerolog)
│   └── shell/            # Shell code generation
├── pkg/version/          # Version information
├── hooks/                # Shell hook scripts
├── examples/             # Example configurations
├── Taskfile.yml          # Task automation
└── README.md             # Main documentation
```

## Configuration Files

Dirvana supports multiple configuration formats:

- `.dirvana.yml` (YAML)
- `.dirvana.toml` (TOML)
- `.dirvana.json` (JSON)

See the `examples/` directory for sample configurations.

## Troubleshooting

### Hook not loading configurations

1. Ensure the directory is authorized: `dirvana list`
2. Check if config file exists: `ls -la .dirvana.*`
3. Test export manually: `dirvana export`
4. Check logs: `DIRVANA_LOG_LEVEL=debug dirvana export`

### Build errors

1. Ensure Go 1.21+ is installed: `go version`
2. Clean and rebuild: `task clean && task build`
3. Check dependencies: `go mod tidy`

### Permission errors

Ensure cache and data directories are writable:
- Cache: `$XDG_CACHE_HOME/dirvana/` (default: `~/.cache/dirvana/`)
- Data: `$XDG_DATA_HOME/dirvana/` (default: `~/.local/share/dirvana/`)

## Next Steps

1. Read the main [README.md](README.md) for full documentation
2. Check out [examples/](examples/) for configuration samples
3. Customize your project configurations
4. Enjoy automatic environment loading!
