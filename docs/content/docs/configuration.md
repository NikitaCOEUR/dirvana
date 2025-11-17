---
title: "Configuration"
weight: 30
---

# Configuration Reference

Complete reference for Dirvana configuration files.

---

## Configuration Files

Dirvana supports two types of configuration files:

1. **Global config**: `~/.config/dirvana/global.yml` - Applied to all projects
2. **Local configs**: `.dirvana.yml` in project directories - Project-specific settings

> [!NOTE]
> Configuration files are merged in this order: **global → root → parent → current directory** (child configs override parent values).

---

## Basic Structure

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json

# Aliases
aliases:
  ll: ls -lah
  k:
    command: kubectl
    completion: kubectl

# Functions
functions:
  mkcd: |
    mkdir -p "$1" && cd "$1"

# Environment variables
env:
  PROJECT_NAME: myproject
  GIT_BRANCH:
    sh: git rev-parse --abbrev-ref HEAD

# Flags
local_only: false      # Don't merge with parent configs
ignore_global: false   # Don't merge with global config
```

---

## Aliases

### Simple Aliases

```yaml
aliases:
  ll: ls -lah
  gs: git status
  build: go build -o bin/app ./cmd
```

### Aliases with Auto-Completion

```yaml
aliases:
  k:
    command: kubectl
    completion: kubectl  # Inherits kubectl completion!

  g:
    command: git
    completion: git

  tf:
    command: terraform
    completion: terraform
```

---

## Functions

Define reusable shell functions:

```yaml
functions:
  # Create directory and cd into it
  mkcd: |
    mkdir -p "$1" && cd "$1"

  # Greeting function
  greet: |
    echo "Hello, $1!"

  # Complex function
  backup: |
    timestamp=$(date +%Y%m%d_%H%M%S)
    tar -czf "backup_${timestamp}.tar.gz" "$1"
    echo "Backup created!"
```

---

## Environment Variables

### Static Values

```yaml
env:
  PROJECT_NAME: myproject
  LOG_LEVEL: debug
  TF_LOG: info
```

### Dynamic Values

```yaml
env:
  # Execute shell command
  GIT_BRANCH:
    sh: git rev-parse --abbrev-ref HEAD

  PROJECT_NAME:
    sh: basename $(pwd)

  CURRENT_USER:
    sh: whoami

  GIT_REPOSITORY:
    sh: git remote get-url origin | sed 's/.*github.com:\(.*\)\.git/\1/'
```

> [!WARNING]
> **Security Note:** Dynamic environment variables require explicit user authorization to prevent execution of untrusted code.

### Template Variables

```yaml
env:
  # Directory where .dirvana.yml is located
  PROJECT_ROOT: "{{.DIRVANA_DIR}}"

  # Extract directory name
  PROJECT_NAME: "{{.DIRVANA_DIR | base}}"

  # Build paths
  BUILD_DIR: "{{.DIRVANA_DIR}}/build"

  # Generate unique ID
  PROJECT_ID: "{{.DIRVANA_DIR | sha256sum | trunc 8}}"
```

See [Template Variables](advanced/templates) for more details.

---

## Configuration Flags

### `local_only`

Prevent merging with parent directory configurations:

```yaml
local_only: true
```

### `ignore_global`

Ignore global configuration:

```yaml
ignore_global: true
```

---

## Commands Reference

### dirvana init

Create a sample configuration file:
```bash
dirvana init
```

### dirvana edit

Open/create configuration in your editor:
```bash
dirvana edit
```

### dirvana validate

Validate configuration file:
```bash
dirvana validate
dirvana validate /path/to/.dirvana.yml
```

### dirvana status

Show current configuration status:
```bash
dirvana status
```

### dirvana allow / revoke

Manage authorization:
```bash
dirvana allow                # Authorize current directory
dirvana revoke               # Revoke authorization
dirvana list                 # List authorized projects
```

---

## IDE Integration

Enable auto-completion and validation:

### YAML Language Server

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json
```

### VS Code

Add to `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json": [
      ".dirvana.yml"
    ]
  }
}
```

### Generate Schema Locally

```bash
dirvana schema -o .vscode/dirvana.schema.json
```

---

## Next Steps

{{< button relref="/docs/advanced" >}}Advanced Features{{< /button >}}
