# Templating Example

This example demonstrates Dirvana's template variable system, inspired by [Taskfile](https://taskfile.dev).

## 🎯 Template Variables

Dirvana provides two special variables that are always available in templates:

- **`{{.DIRVANA_DIR}}`** - Directory containing the `.dirvana.yml` file where the variable is used
- **`{{.USER_WORKING_DIR}}`** - Directory where you invoked the command

## 🔧 Sprig Functions

All [Sprig functions](http://masterminds.github.io/sprig/) are available, including:

### Path Functions
```yaml
env:
  PROJECT_NAME: "{{.DIRVANA_DIR | base}}"           # Extract directory name
  PARENT_DIR: "{{.DIRVANA_DIR | dir}}"              # Get parent directory
  CLEAN_PATH: "{{.DIRVANA_DIR | clean}}"            # Clean/normalize path
```

### String Functions
```yaml
env:
  PROJECT_UPPER: "{{.DIRVANA_DIR | base | upper}}"  # MYPROJECT
  PROJECT_LOWER: "{{.DIRVANA_DIR | base | lower}}"  # myproject
```

### Hash Functions
```yaml
env:
  # Generate unique project ID
  PROJECT_ID: "{{.DIRVANA_DIR | sha256sum | trunc 8}}"

  # Full SHA256 hash
  PROJECT_HASH: "{{.DIRVANA_DIR | sha256sum}}"

  # Other hash functions
  MD5_HASH: "{{.DIRVANA_DIR | md5sum}}"
  SHA1_HASH: "{{.DIRVANA_DIR | sha1sum}}"
```

## 📁 Hierarchical Configs

When using nested `.dirvana.yml` files, each one gets its own `DIRVANA_DIR`:

```
/project/
  .dirvana.yml              ← DIRVANA_DIR = /project
  backend/
    .dirvana.yml            ← DIRVANA_DIR = /project/backend
  frontend/
    .dirvana.yml            ← DIRVANA_DIR = /project/frontend
```

### Parent Config (`/project/.dirvana.yml`)
```yaml
env:
  PROJECT_ROOT: "{{.DIRVANA_DIR}}"  # /project

aliases:
  build: "cd {{.DIRVANA_DIR}} && make build-all"
```

### Child Config (`/project/backend/.dirvana.yml`)
```yaml
env:
  BACKEND_ROOT: "{{.DIRVANA_DIR}}"  # /project/backend
  # Inherits PROJECT_ROOT=/project from parent

aliases:
  build: "cd {{.DIRVANA_DIR}} && go build"  # Uses /project/backend
```

## 💡 Use Cases

### 1. Always Build from Project Root

```yaml
aliases:
  build: "cd {{.DIRVANA_DIR}} && make build"
  test: "cd {{.DIRVANA_DIR}} && go test ./..."
```

No matter where you run `build` or `test`, it will execute from the project root.

### 2. Version/Cache Keys

```yaml
env:
  # Generate unique cache key based on project location
  CACHE_KEY: "{{.DIRVANA_DIR | sha256sum | trunc 8}}"

  # Use in Docker tags
  DOCKER_TAG: "myapp:{{.DIRVANA_DIR | sha256sum | trunc 8}}"
```

### 3. Dynamic Paths

```yaml
env:
  BUILD_DIR: "{{.DIRVANA_DIR}}/build"
  DIST_DIR: "{{.DIRVANA_DIR}}/dist"
  CONFIG_FILE: "{{.DIRVANA_DIR}}/config/production.yml"

aliases:
  clean: "rm -rf {{.DIRVANA_DIR}}/build {{.DIRVANA_DIR}}/dist"
```

### 4. Conditional Paths

```yaml
aliases:
  test:
    command: "pytest {{.DIRVANA_DIR}}/tests"
    when:
      dir: "{{.DIRVANA_DIR}}/tests"
    else: "echo 'No tests directory found'"
```

### 5. Return to Original Directory

```yaml
functions:
  # Jump to project root
  goto: "cd {{.DIRVANA_DIR}}"

  # Return to where you were
  back: "cd {{.USER_WORKING_DIR}}"
```

## 🚀 Try It Out

```bash
# Copy this example to your project
cp examples/templating/.dirvana.yml /path/to/your/project/

# Authorize the directory
cd /path/to/your/project
dirvana allow

# Source the configuration
eval "$(dirvana export bash)"

# Try the info alias to see templates in action
info

# Output:
# Project: your-project
# Location: /path/to/your/project
# You ran this from: /path/to/your/project
# Project ID: a1b2c3d4
```

## 📚 More Information

- [Sprig Function Reference](http://masterminds.github.io/sprig/)
- [Go Template Documentation](https://pkg.go.dev/text/template)
- [Taskfile Documentation](https://taskfile.dev/usage/) (inspiration)

## 🎓 Advanced Examples

### Monorepo Setup

```yaml
# Root: /monorepo/.dirvana.yml
env:
  MONOREPO_ROOT: "{{.DIRVANA_DIR}}"

functions:
  goto_root: "cd {{.DIRVANA_DIR}}"

---

# Service: /monorepo/services/api/.dirvana.yml
env:
  SERVICE_ROOT: "{{.DIRVANA_DIR}}"
  SERVICE_NAME: "{{.DIRVANA_DIR | base}}"

aliases:
  build: "cd {{.DIRVANA_DIR}} && docker build -t {{.DIRVANA_DIR | base}}:latest ."
  logs: "docker logs {{.DIRVANA_DIR | base}}"
```

### Multi-Environment Deployment

```yaml
env:
  PROJECT_ID: "{{.DIRVANA_DIR | sha256sum | trunc 8}}"

aliases:
  deploy_dev: "./deploy.sh dev {{.DIRVANA_DIR | base}}-{{.PROJECT_ID}}"
  deploy_prod: "./deploy.sh prod {{.DIRVANA_DIR | base}}-{{.PROJECT_ID}}"
```
