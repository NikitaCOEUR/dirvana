---
title: "Development"
weight: 50
---

# Development Guide

Contributing to Dirvana.

---

## Prerequisites

- **Go** 1.21 or later
- **Task** - Build automation ([install](https://taskfile.dev/))
- **aqua** (optional) - Dependency management ([install](https://aquaproj.github.io/))

---

## Building

```bash
# Build binary
task build

# Build for all platforms
task build-all
```

Binary will be in `bin/dirvana`.

---

## Testing

```bash
# Run unit tests
task test

# Run tests with coverage
task test-coverage

# Run tests with verbose output
task test-verbose

# Run integration tests (Docker required)
task test-integration

# Run specific shell integration test
task test-integration-bash
task test-integration-zsh

# Run all tests (unit + integration)
task test-all
```

---

## Integration Tests

Integration tests validate Dirvana in real shell environments using Docker:

- **Bash** - Tests aliases, functions, env vars
- **Zsh** - Tests hooks and features

Each test:
1. Builds a Docker image with the shell
2. Installs Dirvana and sets up hooks
3. Creates test configuration
4. Validates all features work

---

## Project Structure

```
dirvana/
├── cmd/
│   └── dirvana/          # CLI entry point
├── internal/
│   ├── auth/             # Authorization system
│   ├── cache/            # Cache management
│   ├── cli/              # CLI commands
│   ├── completion/       # Auto-completion engines
│   ├── config/           # Configuration loading
│   ├── context/          # Directory context
│   ├── logger/           # Logging
│   ├── shell/            # Shell code generation
│   └── timing/           # Performance timing
├── pkg/
│   └── version/          # Version information
├── hooks/                # Shell hook scripts
├── schema/               # JSON Schema
├── examples/             # Example configurations
├── tests/                # Integration tests
└── Taskfile.yml          # Build automation
```

---

## Available Tasks

Run `task --list` to see all available tasks:

```bash
task --list
```

---

## Code Quality

```bash
# Format code
task fmt

# Run linter
task lint

# Ensure coverage stays at 90%+
task test-coverage
```

---

## Release Process

This project uses [GoReleaser](https://goreleaser.com/) for automated releases with semantic versioning.

### Conventional Commits

All commits must follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features (minor version bump)
- `fix:` - Bug fixes (patch version bump)
- `perf:` - Performance improvements (patch version bump)
- `refactor:` - Code refactoring (patch version bump)
- `docs:` - Documentation changes (no version bump)
- `test:` - Test changes (no version bump)
- `chore:` - Maintenance tasks (no version bump)

**Breaking changes**: Add `BREAKING CHANGE:` in commit footer or `!` after type.

### Creating a Release

1. Ensure all commits follow conventional commit format
2. Create and push a tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. GitHub Actions automatically:
   - Runs tests
   - Builds binaries for all platforms
   - Creates GitHub release with changelog

---

## Contributing

Contributions are welcome! Please ensure:

- All commits follow [Conventional Commits](https://www.conventionalcommits.org/)
- All tests pass: `task test`
- Code is formatted: `task fmt`
- Linter passes: `task lint`
- Coverage remains at 90%+

---

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests: `task test-all`
5. Commit with conventional commits
6. Push and create a pull request

---

## Getting Help

- [GitHub Issues](https://github.com/NikitaCOEUR/dirvana/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/NikitaCOEUR/dirvana/discussions) - Questions and discussions

---

## License

MIT License - See [LICENSE](https://github.com/NikitaCOEUR/dirvana/blob/main/LICENSE) for details.
