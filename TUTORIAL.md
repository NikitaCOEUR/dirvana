# 🚀 Dirvana Quick Start Tutorial

**Dirvana** is a lightweight CLI tool that automatically loads shell aliases, functions, and environment variables per directory. Perfect for managing project-specific shell environments.

## 📋 Table of Contents

- [Installation](#-installation)
- [Initial Setup](#-initial-setup)
- [Your First Project](#-your-first-project)
- [Common Use Cases](#-common-use-cases)
- [Advanced Features](#-advanced-features)
- [Troubleshooting](#-troubleshooting)

---

## 🔧 Installation

### Option 1: Go Install

```bash
go install github.com/NikitaCOEUR/dirvana/cmd/dirvana@latest
```

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Option 2: Direct Download

Download the binary from [GitHub Releases](https://github.com/NikitaCOEUR/dirvana/releases) for your platform.

Example for Linux:
```bash
# Download latest release for Linux AMD64
curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-linux-amd64 -o dirvana

# Make it executable
chmod +x dirvana

# Move to a directory in your PATH
sudo mv dirvana /usr/local/bin/
```

### Verify Installation

```bash
dirvana --version
```

---

## ⚙️ Initial Setup

### 1. Install the Shell Hook

The `setup` command automatically configures your shell:

```bash
dirvana setup
```

This command will:
- ✅ Automatically detect your shell (Bash or Zsh)
- ✅ Add the hook to your config file (`~/.bashrc` or `~/.zshrc`)
- ✅ Explain how to reload your shell

### 2. Reload Your Shell

```bash
# For Bash
source ~/.bashrc

# For Zsh
source ~/.zshrc

# Or simply open a new terminal
```

✨ **That's it!** Dirvana is now active and will automatically watch for directory changes.

---

## 🎯 Your First Project

### Scenario: Go Project with Git Aliases

Let's create an environment for a Go project with Git shortcuts.

#### Step 1: Navigate to Your Project

```bash
cd ~/my-projects/my-go-app
```

#### Step 2: Initialize Dirvana

```bash
dirvana init
```

This creates a `.dirvana.yml` file with JSON Schema validation for IDE auto-completion.

#### Step 3: Edit the Configuration

Open `.dirvana.yml` in your favorite editor:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json

aliases:
  # Quick Git aliases
  gs: git status
  ga: git add
  gc: git commit
  gp: git push

  # Go aliases
  build: go build -o bin/app ./cmd
  run: go run ./cmd
  test: go test ./...

functions:
  # Quick commit with message
  gcm: |
    git commit -m "$1"

  # Create and enter directory
  mkcd: |
    mkdir -p "$1" && cd "$1"

env:
  # Static variables
  PROJECT_NAME: my-go-app
  GO_ENV: development

  # Dynamic variables (evaluated at load time)
  GIT_BRANCH:
    sh: git rev-parse --abbrev-ref HEAD
  PROJECT_ROOT:
    sh: pwd
```

#### Step 4: Authorize the Project

**Important**: For security reasons, you must explicitly authorize each project:

```bash
dirvana allow
```

You'll see:
```
✅ Authorized: /home/user/my-projects/my-go-app
💡 Tip: Run 'eval "$(dirvana export)"' to load the environment
```

#### Step 5: Reload the Environment

```bash
# Exit and return to the directory
cd .. && cd -

# Or execute manually
eval "$(dirvana export)"
```

#### Step 6: Test Your Aliases!

```bash
# Test an alias
gs  # Executes 'git status'

# Test a function
gcm "My first commit with Dirvana"  # Executes 'git commit -m "..."'

# Test an environment variable
echo $PROJECT_NAME  # Displays 'my-go-app'
echo $GIT_BRANCH    # Displays your current Git branch
```

🎉 **Congratulations!** You've configured your first Dirvana project.

---

## 💼 Common Use Cases

### 1. Terraform project with taskfile

`.dirvana.yml`:
```yaml
aliases:
  tf:
    command: task terraform --
    completion: terraform
```

This permit du execute `task terraform` with auto-completion for `terraform` commands.
exemple:
```bash
tf init
tf plan

# instead of
task terraform -- init
task terraform -- plan
```

### 2. Node.js Project with npm/yarn

`.dirvana.yml`:
```yaml
aliases:
  dev: npm run dev
  build: npm run build
  test: npm test
  lint: npm run lint

env:
  NODE_ENV: development
  PORT: 3000
```

### 3. Docker Project with Shortcuts

`.dirvana.yml`:
```yaml
aliases:
  dc: docker-compose
  dcu: docker-compose up -d
  dcd: docker-compose down
  dcl: docker-compose logs -f

functions:
  dcexec: |
    docker-compose exec "$1" bash

env:
  COMPOSE_PROJECT_NAME: my-project
  DOCKER_BUILDKIT: 1
```

### 4. Kubernetes Project

`.dirvana.yml`:
```yaml
aliases:
  k: kubectl
  kgp: kubectl get pods
  kgs: kubectl get services
  kgd: kubectl get deployments
  kdp: kubectl describe pod

env:
  KUBECONFIG: ./kubeconfig.yaml
  KUBE_NAMESPACE: production
```

### 5. Global Configuration

Create `~/.config/dirvana/global.yml` for aliases available everywhere:

```yaml
# Global configuration - available in all projects

aliases:
  # System aliases
  ll: ls -lah
  la: ls -A
  ...: cd ../..

  # Universal Git aliases
  g: git
  gst: git status
  gco: git checkout

functions:
  # Create and enter directory
  mkcd: |
    mkdir -p "$1" && cd "$1"

  # Search in history
  hgrep: |
    history | grep "$1"

env:
  EDITOR: vim
  PAGER: less
  LANG: en_US.UTF-8
```

---

## 🔥 Advanced Features

### 1. Configuration Hierarchy

Configurations are merged hierarchically:

```
~/.config/dirvana/global.yml    (global)
    ↓
~/projects/.dirvana.yml          (project root)
    ↓
~/projects/backend/.dirvana.yml  (sub-directory)
```

**Example**:

Global (`~/.config/dirvana/global.yml`):
```yaml
aliases:
  g: git
env:
  EDITOR: vim
```

Project (`~/projects/.dirvana.yml`):
```yaml
aliases:
  build: make build
env:
  PROJECT: myapp
```

Backend (`~/projects/backend/.dirvana.yml`):
```yaml
aliases:
  g: go  # Override global 'git' alias
  build: go build  # Override parent 'make build' alias
env:
  SERVICE: api
```

When you're in `~/projects/backend/`, you have:
- Alias `g` → `go` (overridden)
- Alias `build` → `go build` (overridden)
- Variable `EDITOR` → `vim` (from global)
- Variable `PROJECT` → `myapp` (from parent)
- Variable `SERVICE` → `api` (local)

### 2. Configuration Isolation

#### Ignore Global Config

```yaml
ignore_global: true  # Don't merge with global config

aliases:
  # Only these aliases will be available
  build: make
```

#### Local Only Configuration

```yaml
local_only: true  # Don't merge with parent directories

aliases:
  # Independent from parent directories
  deploy: ./deploy.sh
```

### 3. Dynamic Environment Variables

Execute shell commands at load time:

```yaml
env:
  # Static
  PROJECT_NAME: myapp

  # Dynamic - evaluated at each load
  GIT_BRANCH:
    sh: git rev-parse --abbrev-ref HEAD

  GIT_COMMIT:
    sh: git rev-parse --short HEAD

  CURRENT_USER:
    sh: whoami

  BUILD_TIME:
    sh: date +%Y-%m-%d_%H:%M:%S

  KUBE_CONTEXT:
    sh: kubectl config current-context
```

### 4. Intelligent Auto-completion

Dirvana provides auto-completion for your aliases that inherit from the original command:

```yaml
aliases:
  # 'k' auto-completion will work like 'kubectl'
  k:
    command: kubecolor
    completion: kubectl

  # 'g' auto-completion will work like 'git'
  g:
    command: git
    completion: git
```

Now `k get <TAB>` will suggest Kubernetes resources!

### 5. IDE Integration with JSON Schema

For auto-completion in your editor:

#### VS Code

Create `.vscode/settings.json` in your project:

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json": [
      ".dirvana.yml",
      ".dirvana.yaml"
    ]
  }
}
```

#### Or Generate Schema Locally

```bash
dirvana schema -o .vscode/dirvana.schema.json
```

Then reference it in your `.dirvana.yml`:

```yaml
# yaml-language-server: $schema=.vscode/dirvana.schema.json
```

---

## 🎓 Useful Commands

### Check Status

```bash
dirvana status
```

Displays:
- 📂 Current directory
- 🔒 Authorization status
- 📝 Configuration hierarchy
- 🔗 Defined aliases
- ⚙️ Available functions
- 🌍 Environment variables
- 💾 Cache status

### Edit Configuration

```bash
dirvana edit          # Open local config
dirvana edit --global # Open global config
```

### Validate Configuration

```bash
dirvana validate
dirvana validate /path/to/.dirvana.yml
```

### Authorize/Revoke Projects

```bash
dirvana allow                    # Authorize current directory
dirvana allow /path/to/project   # Authorize specific path
dirvana deny /path/to/project    # Revoke authorization
```

### View Authorized Directories

```bash
dirvana status
# or look directly at
cat ~/.config/dirvana/authorized.txt
```

### Clean Cache

```bash
dirvana clean        # Clean cache only
dirvana clean --all  # Clean cache + temporary files
```

### Generate Completion Scripts (Automatically done by `dirvana setup`)

```bash
# For Bash
dirvana completion bash > ~/.dirvana-completion.bash
echo "source ~/.dirvana-completion.bash" >> ~/.bashrc

# For Zsh
dirvana completion zsh > ~/.dirvana-completion.zsh
echo "source ~/.dirvana-completion.zsh" >> ~/.zshrc
```

---

## 🐛 Troubleshooting

### Problem: Aliases Don't Load

**Solution 1**: Check that the hook is installed

```bash
# Check for the hook
grep dirvana ~/.bashrc  # or ~/.zshrc

# Reinstall if necessary
dirvana setup
source ~/.bashrc
```

**Solution 2**: Check authorization

```bash
dirvana status
# If "Not authorized", run:
dirvana allow
```

**Solution 3**: Reload environment

```bash
cd .. && cd -
# or
eval "$(dirvana export)"
```

### Problem: "directory not authorized" Error

This is a security measure. Explicitly authorize the directory:

```bash
dirvana allow
```

### Problem: Environment Variables Not Set

**Cause**: Dynamic shell commands may fail.

**Solution**: Test the command manually:

```yaml
env:
  GIT_BRANCH:
    sh: git rev-parse --abbrev-ref HEAD  # Fails if not a Git repo
```

Check:
```bash
git rev-parse --abbrev-ref HEAD  # Manual test
```

### Problem: Alias Conflicts

**Cause**: A local alias overrides a global or system alias.

**Solution 1**: Rename the alias:

```yaml
aliases:
  mygs: git status  # Instead of 'gs' which might conflict
```

**Solution 2**: Use `local_only` or `ignore_global`:

```yaml
local_only: true  # Ignore parent configs
```

### Problem: Slow Performance

**Solution**: Check cache:

```bash
dirvana status  # See "Cache status"
```

If cache is not being used:
```bash
dirvana clean  # Clean and regenerate cache
```

### Problem: Hook Doesn't Work After Update

**Solution**: Reinstall the hook:

```bash
dirvana setup
source ~/.bashrc  # or ~/.zshrc
```

### Enable Debug Mode

To diagnose issues:

```bash
DIRVANA_LOG_LEVEL=debug cd your-project
```

---

## 📚 Complete Examples

Check the [`examples/`](./examples/) directory for complete configurations:

- **[Go](./examples/golang/)**: Go project with build, test, and dependency management
- **[Node.js](./examples/nodejs/)**: Node application with npm/yarn
- **[Python](./examples/python/)**: Python project with virtualenv
- **[Docker](./examples/docker/)**: Multi-container with docker-compose
- **[Kubernetes](./examples/kubernetes/)**: K8s cluster management
- **[Rust](./examples/rust/)**: Rust project with Cargo
- **[Advanced Completion](./examples/advanced-completion/)**: Advanced auto-completion

---

## 🎯 Best Practices

### 1. Start Simple

Don't overload your first configuration. Start with 2-3 aliases:

```yaml
aliases:
  build: make build
  test: make test

env:
  PROJECT_NAME: myapp
```

### 2. Use Global Config for Universal Aliases

Put in `~/.config/dirvana/global.yml`:
- System aliases (`ll`, `la`, etc.)
- Generic Git aliases (`g`, `gst`, etc.)
- Common environment variables (`EDITOR`, `PAGER`)

### 3. Document Complex Aliases

```yaml
aliases:
  # Launch dev server with hot-reload
  dev: npm run dev

functions:
  # Deploy to specified environment
  # Usage: deploy production
  deploy: |
    echo "Deploying to $1..."
    ./scripts/deploy.sh "$1"
```

### 4. Version Your Configurations

Commit `.dirvana.yml` to Git to share with your team:

```bash
git add .dirvana.yml
git commit -m "Add Dirvana configuration"
```

### 5. Test Before Authorizing

```bash
# Check config
dirvana validate

# See what will be loaded
dirvana status

# Only authorize if everything is correct
dirvana allow
```

### 6. Use Short but Clear Alias Names

✅ **Good**:
```yaml
aliases:
  gs: git status
  dc: docker-compose
  k: kubectl
```

❌ **Bad**:
```yaml
aliases:
  x: git status  # Too cryptic
  dockercomposeup: docker-compose up  # Too long
```

---

## 🚀 Going Further

### CI/CD Integration

Use Dirvana in your pipelines:

```yaml
# .gitlab-ci.yml
build:
  script:
    - curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-linux-amd64 -o /usr/local/bin/dirvana
    - chmod +x /usr/local/bin/dirvana
    - eval "$(dirvana export)"
    - build  # Uses alias defined in .dirvana.yml
```

### Custom Scripts

Create complex functions:

```yaml
functions:
  release: |
    #!/bin/bash
    VERSION=$1
    if [ -z "$VERSION" ]; then
      echo "Usage: release <version>"
      return 1
    fi

    echo "Creating release $VERSION..."
    git tag -a "v$VERSION" -m "Release $VERSION"
    git push origin "v$VERSION"
    goreleaser release --clean
```

### Multi-environment

Structure for dev/staging/prod:

```
project/
├── .dirvana.yml           # Base config
├── environments/
│   ├── dev/
│   │   └── .dirvana.yml   # Override for dev
│   ├── staging/
│   │   └── .dirvana.yml   # Override for staging
│   └── prod/
│       └── .dirvana.yml   # Override for prod
```

---

## 🤝 Contributing

Dirvana is open source! Contributions welcome:

1. 🐛 [Report a bug](https://github.com/NikitaCOEUR/dirvana/issues)
2. 💡 [Suggest a feature](https://github.com/NikitaCOEUR/dirvana/issues)
3. 🔧 [Submit a Pull Request](https://github.com/NikitaCOEUR/dirvana/pulls)

---

## 📖 Resources

- [Complete README](./README.md)
- [Detailed Setup Guide](./SETUP.md)
- [Configuration Examples](./examples/)
- [JSON Schema](./schema/dirvana.schema.json)
- [Changelog](./CHANGELOG.md)

---

## 💬 Support

- **GitHub Issues**: [github.com/NikitaCOEUR/dirvana/issues](https://github.com/NikitaCOEUR/dirvana/issues)
- **Discussions**: [github.com/NikitaCOEUR/dirvana/discussions](https://github.com/NikitaCOEUR/dirvana/discussions)

---

**Happy coding with Dirvana! 🎉**
