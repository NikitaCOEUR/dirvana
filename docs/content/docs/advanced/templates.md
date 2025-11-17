---
title: "Template Variables"
weight: 42
---

# Template Variables

Use Go templates with [Sprig functions](http://masterminds.github.io/sprig/) for dynamic path references and string manipulation.

---

## Available Variables

### `{{.DIRVANA_DIR}}`

Directory containing the `.dirvana.yml` file where the variable is defined.

```yaml
env:
  PROJECT_ROOT: "{{.DIRVANA_DIR}}"
  BUILD_DIR: "{{.DIRVANA_DIR}}/build"
```

### `{{.USER_WORKING_DIR}}`

Directory where you invoked the command.

```yaml
env:
  CURRENT_DIR: "{{.USER_WORKING_DIR}}"
```

---

## Hierarchical Configs

Each `.dirvana.yml` file gets its own `DIRVANA_DIR`:

```yaml
# Parent: /project/.dirvana.yml
env:
  PROJECT_ROOT: "{{.DIRVANA_DIR}}"  # /project

# Child: /project/backend/.dirvana.yml
env:
  BACKEND_ROOT: "{{.DIRVANA_DIR}}"  # /project/backend
  # Inherits PROJECT_ROOT=/project from parent
```

---

## Sprig Functions

Sprig provides 100+ functions for string manipulation:

### Path Manipulation

```yaml
env:
  # Extract directory name
  PROJECT_NAME: "{{.DIRVANA_DIR | base}}"              # myproject

  # Get parent directory
  PARENT_DIR: "{{.DIRVANA_DIR | dir}}"                 # /home/user/projects
```

### String Transformation

```yaml
env:
  # Uppercase
  PROJECT_UPPER: "{{.DIRVANA_DIR | base | upper}}"     # MYPROJECT

  # Lowercase
  PROJECT_LOWER: "{{.DIRVANA_DIR | base | lower}}"     # myproject

  # Replace
  PROJECT_CLEAN: "{{.DIRVANA_DIR | base | replace \"-\" \"_\"}}"
```

### Hash Functions

```yaml
env:
  # Generate unique project ID
  PROJECT_ID: "{{.DIRVANA_DIR | sha256sum | trunc 8}}"

  # Cache key
  CACHE_KEY: "build-{{.DIRVANA_DIR | sha256sum | trunc 8}}"
```

---

## Use Cases

### Monorepos

Each service has its own `DIRVANA_DIR`:

```yaml
# /monorepo/service-a/.dirvana.yml
env:
  SERVICE_ROOT: "{{.DIRVANA_DIR}}"
  SERVICE_NAME: "{{.DIRVANA_DIR | base}}"
  BUILD_DIR: "{{.DIRVANA_DIR}}/dist"

aliases:
  build: "cd {{.DIRVANA_DIR}} && make build"
```

### Build Paths

Reference build directories relative to project root:

```yaml
env:
  PROJECT_ROOT: "{{.DIRVANA_DIR}}"
  BUILD_DIR: "{{.DIRVANA_DIR}}/build"
  CONFIG_FILE: "{{.DIRVANA_DIR}}/config.yml"

aliases:
  # Always build from project root
  build: "cd {{.DIRVANA_DIR}} && make build"
```

### Unique Identifiers

Generate cache keys or Docker tags:

```yaml
env:
  # Unique ID based on project path
  PROJECT_ID: "{{.DIRVANA_DIR | sha256sum | trunc 8}}"

  # Docker tag
  DOCKER_TAG: "{{.DIRVANA_DIR | base}}-{{.DIRVANA_DIR | sha256sum | trunc 8}}"
```

### Dynamic Configs

Create project-specific configuration paths:

```yaml
env:
  # Generate unique kubeconfig per project
  KUBECONFIG: "/tmp/kubeconfig-{{.DIRVANA_DIR | sha256sum | trunc 8}}"

  # Generate unique talosconfig
  TALOSCONFIG: "/tmp/talosconfig-{{.DIRVANA_DIR | sha256sum | trunc 8}}"
```

---

## Templates in Aliases

```yaml
aliases:
  # Always build from project root
  build: "cd {{.DIRVANA_DIR}} && make build"

  # Multi-line with templates
  info: |
    echo "Project: {{.DIRVANA_DIR | base}}"
    echo "Location: {{.DIRVANA_DIR}}"
    echo "Current dir: {{.USER_WORKING_DIR}}"

  # Use in conditionals
  deploy:
    when:
      file: "{{.DIRVANA_DIR}}/deploy.sh"
    command: "{{.DIRVANA_DIR}}/deploy.sh --id={{.DIRVANA_DIR | sha256sum | trunc 8}}"
```

---

## Templates in Functions

```yaml
functions:
  # Jump to project root from anywhere
  goto: "cd {{.DIRVANA_DIR}}"

  # Show project info
  project-info: |
    echo "Project: {{.DIRVANA_DIR | base}}"
    echo "Path: {{.DIRVANA_DIR}}"
```

---

## Available Sprig Functions

See the [Sprig documentation](http://masterminds.github.io/sprig/) for all available functions:

- **String functions**: `trim`, `upper`, `lower`, `replace`, `split`, `join`
- **Path functions**: `base`, `dir`, `clean`, `ext`
- **Hash functions**: `sha256sum`, `sha1sum`, `md5sum`
- **Encoding**: `b64enc`, `b64dec`
- **Regex**: `regexMatch`, `regexFind`, `regexReplaceAll`
- And many more!

---

> [!NOTE]
> See `examples/templating/` in the repository for comprehensive examples.
