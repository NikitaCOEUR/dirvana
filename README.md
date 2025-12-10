# Dirvana - Reach directory nirvana
[![Read the Docs](https://img.shields.io/badge/read-docs-informational)](https://nikitacoeur.github.io/dirvana/)
[![GitHub Release](https://img.shields.io/github/v/release/NikitaCOEUR/dirvana?sort=semver&display_name=release&style=flat&color=%2300ADD8)](https://github.com/NikitaCOEUR/dirvana/releases) [![codecov](https://codecov.io/gh/NikitaCOEUR/dirvana/graph/badge.svg?token=IBRJQQQB3V)](https://codecov.io/gh/NikitaCOEUR/dirvana) [![License](https://img.shields.io/github/license/NikitaCOEUR/dirvana)](LICENSE) ![GitHub repo size](https://img.shields.io/github/repo-size/NikitaCOEUR/dirvana)


![dirvana-logo](docs/static/DirvanaLogo.png)

> [!WARNING]
> **Beta Status - We Need Your Feedback!**
>
> Dirvana is currently in beta. Found a bug? Have a feature request? Please [open an issue](https://github.com/NikitaCOEUR/dirvana/issues)!


## Automatically load shell aliases, functions, and environment variables per directory.

Dirvana is a lightweight CLI tool that manages project-specific shell environments.

When you enter a directory, Dirvana automatically loads the configuration defined in `.dirvana.yml`, giving you instant access to project-specific commands and settings.

When you leave, everything is automatically unloaded.

## The Problem

```bash
$ cd ~/projects/terraform
$ export TF_LOG=debug
$ alias tf="task terraform --"
$ alias plan="task terraform -- plan"
# ... and don't forget to unset everything when leaving!
```

## The Solution

```yaml
# .dirvana.yml
aliases:
  tf:
    command: task terraform --
    completion: terraform  # Auto-completion works!
  plan: task terraform -- plan

env:
  TF_LOG: debug
```

```bash
$ cd ~/projects/terraform
# Everything loads automatically!
$ tf <TAB>          # Auto-completion works!
  apply  console  destroy  init  plan  validate ...

$ cd ..
# Everything unloads automatically!
```

---

## Features

- **Fast**: <10ms overhead with intelligent caching
- **Secure**: Authorization system prevents untrusted configs
- **Hierarchical**: Merge configurations from parent directories
- **Simple**: YAML configuration with JSON Schema validation
- **Compatible**: Works with Bash, Zsh, and Fish
- **Auto-completion**: Inherits completion from aliased commands
- **Conditional Aliases**: Execute commands based on runtime conditions
- **Template Variables**: Go templates with Sprig functions

---

## Quick Start

### 1. Install

```bash
# Using go install
go install github.com/NikitaCOEUR/dirvana/cmd/dirvana@latest

# Or download binary
curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-linux-amd64 -o /usr/local/bin/dirvana
chmod +x /usr/local/bin/dirvana
```

### 2. Setup Shell Hook

```bash
dirvana setup
source ~/.bashrc  # or ~/.zshrc, or ~/.config/fish/config.fish
```

### 3. Create Configuration

```bash
cd your-project
dirvana init
dirvana allow
```

**That's it!** Your environment is now automatically managed.

---

## Configuration Example

```yaml
# Simple aliases
aliases:
  # With auto-completion
  tf:
    command: task terraform -- # Execute a wrapper command that use specific variables
    completion: terraform      # But keep terraform completion

  # Conditional execution
  k:
    when:
      file: "$KUBECONFIG"      # Check if KUBECONFIG file exists
    command: kubecolor         # If exists, use command kubecolor based on $KUBECONFIG
    else: task kubecolor --    # else execute a task which generate your kubeconfig file and call kubecolor afterwards
    completion: kubectl        # Inherit kubectl completion

# Functions
functions:
  mkcd: |
    mkdir -p "$1" && cd "$1"

# Environment variables
env:
  KUBECONFIG: "/tmp/kubeconfig-{{.USER_WORKING_DIR | sha256sum | trunc 8}}"
```

---

## Documentation

**Full documentation is available at [https://nikitacoeur.github.io/dirvana/](https://nikitacoeur.github.io/dirvana/)**

- [Installation Guide](https://nikitacoeur.github.io/dirvana/docs/installation/) - Detailed installation instructions
- [Quick Start](https://nikitacoeur.github.io/dirvana/docs/quick-start/) - Get up and running in 5 minutes
- [Configuration Reference](https://nikitacoeur.github.io/dirvana/docs/configuration/) - Complete configuration guide
- [Conditional Aliases](https://nikitacoeur.github.io/dirvana/docs/advanced/conditional-aliases/) - Runtime condition checks
- [Template Variables](https://nikitacoeur.github.io/dirvana/docs/advanced/templates/) - Go templates with Sprig functions
- [Development Guide](https://nikitacoeur.github.io/dirvana/docs/development/) - Contributing and development

---

## Contributing

Contributions are welcome! Please ensure:
- All commits follow [Conventional Commits](https://www.conventionalcommits.org/)
- All tests pass: `task test`
- Code is formatted: `task fmt`
- Linter passes: `task lint`

---

## License

MIT License - See [LICENSE](LICENSE) file for details

## Author

[Nikita C](https://github.com/NikitaCOEUR)
