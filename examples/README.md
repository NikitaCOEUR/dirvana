# Dirvana Examples

This directory contains real-world configuration examples for different development environments and use cases.

## 📁 Available Examples

### General Purpose
- **[.dirvana.yml](/.dirvana.yml)** - Basic example showing all core features
- **[.dirvana.json](/.dirvana.json)** - JSON format example
- **[.dirvana.toml](/.dirvana.toml)** - TOML format example

### Language-Specific

#### [golang/](./golang/)
Go development environment with:
- Build, test, and benchmark shortcuts
- Cross-compilation helpers
- Package management
- Mock generation
- Coverage reporting

#### [nodejs/](./nodejs/)
Node.js/TypeScript environment with:
- npm/yarn/pnpm shortcuts
- Component scaffolding
- Test file runners
- Dependency management
- Environment-specific configs

#### [python/](./python/)
Python development with:
- Virtual environment management
- Django/Flask helpers
- Testing with pytest
- Linting and formatting
- Jupyter notebook support

#### [rust/](./rust/)
Rust development featuring:
- Cargo command shortcuts
- Feature flag testing
- Cross-compilation
- Benchmarking
- Coverage generation

### Infrastructure

#### [docker/](./docker/)
Docker-based projects with:
- Docker Compose shortcuts
- Container management
- Log viewing
- Resource cleanup
- Volume backups

#### [kubernetes/](./kubernetes/)
Kubernetes development including:
- kubectl shortcuts with completion
- Pod log aggregation
- Port forwarding helpers
- Context switching
- Helm integration

## 🚀 Usage

### Copy an Example

```bash
# Copy the example that matches your project
cp examples/golang/.dirvana.yml /path/to/your/project/

# Or create a symlink for testing
ln -s $(pwd)/examples/golang/.dirvana.yml /path/to/your/project/
```

### Customize for Your Project

1. Edit the copied file to match your project structure
2. Add project-specific aliases and functions
3. Update environment variables
4. Authorize the directory:
   ```bash
   cd /path/to/your/project
   dirvana allow
   ```

### Combine Examples

You can combine multiple examples using Dirvana's hierarchy feature:

```bash
# Parent directory (general development tools)
~/projects/.dirvana.yml

# Project directory (language-specific)
~/projects/myapp/.dirvana.yml
```

The configurations will merge automatically, with the child overriding parent values.

## 📝 Format Reference

Dirvana supports three configuration formats:

### YAML (Recommended)
```yaml
aliases:
  k: kubectl

functions:
  deploy: |
    echo "Deploying..."

env:
  DEBUG: "true"
  GIT_BRANCH:
    sh: git branch --show-current
```

### JSON
```json
{
  "aliases": {
    "k": "kubectl"
  },
  "functions": {
    "deploy": "echo \"Deploying...\""
  },
  "env": {
    "DEBUG": "true"
  }
}
```

### TOML
```toml
[aliases]
k = "kubectl"

[functions]
deploy = "echo 'Deploying...'"

[env]
DEBUG = "true"
```

## 🎯 Advanced Features Showcase

### Completion Control

```yaml
aliases:
  # Auto-detect completion from command
  k: kubectl

  # Inherit completion from another command
  mykube:
    command: /usr/local/bin/kubectl-wrapper
    completion: kubectl

  # Disable completion
  myalias:
    command: echo "hello"
    completion: false

  # Custom completion
  deploy:
    command: ./deploy.sh
    completion:
      bash: "complete -W 'dev staging prod' deploy"
      zsh: "compdef '_arguments \"1: :(dev staging prod)\"' deploy"
```

### Dynamic Environment Variables

```yaml
env:
  # Static value
  PROJECT_NAME: myproject

  # Execute shell command
  GIT_BRANCH:
    sh: git rev-parse --abbrev-ref HEAD

  # Compose commands
  VERSION:
    sh: cat VERSION || echo "0.1.0"
```

### Configuration Flags

```yaml
# Prevent merging with parent configs
local_only: true

# Ignore global config (~/.config/dirvana/global.yml)
ignore_global: true
```

## 💡 Tips

1. **Start Simple**: Begin with a few aliases and expand as needed
2. **Use Comments**: Document complex functions and non-obvious shortcuts
3. **Test First**: Try commands manually before adding them to your config
4. **Version Control**: Commit your `.dirvana.yml` to share with your team
5. **IDE Support**: Add `# yaml-language-server: $schema=...` for autocomplete

## 📚 More Resources

- [Main Documentation](../README.md)
- [Configuration Schema](../schema/dirvana.schema.json)
- [Setup Guide](../SETUP.md)

## 🤝 Contributing Examples

Have a great example? Share it!

1. Create a new directory under `examples/`
2. Add a well-documented `.dirvana.yml`
3. Update this README
4. Submit a pull request

Popular examples:
- Ruby/Rails
- PHP/Laravel
- Java/Maven
- Terraform/IaC
- Data Science (Jupyter, R, etc.)
