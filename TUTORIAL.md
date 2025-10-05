# 🚀 Dirvana Quick Start Tutorial

**Dirvana** automatically loads shell aliases, functions, and environment variables when you enter a directory. Think of it as having a different shell configuration per project.

## 📋 Table of Contents

- [Installation](#-installation)
- [Setup](#-setup)
- [Basic Usage](#-basic-usage)
- [Common Use Cases](#-common-use-cases)
- [Tips & Troubleshooting](#-tips--troubleshooting)


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

Download the binary from [GitHub Releases](https://github.com/NikitaCOEUR/dirvana/releases).

```bash
# Example for Linux AMD64
curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-linux-amd64 -o dirvana
chmod +x dirvana
sudo mv dirvana /usr/local/bin/
```

### Verify

```bash
dirvana --version
```

---

## ⚙️ Setup

Install the shell hook (only once):

```bash
dirvana setup
source ~/.bashrc  # or ~/.zshrc
```

That's it! Dirvana is now watching for directory changes.

---

## 🎯 Basic Usage

### 1. Initialize a Project

```bash
cd ~/my-project
dirvana init
```

This creates `.dirvana.yml` in your project.

### 2. Configure Aliases and Variables

Edit `.dirvana.yml`:

```yaml
aliases:
  gs: git status
  build: go build -o bin/app ./cmd
  test: go test ./...

env:
  PROJECT_NAME: my-project
  GO_ENV: development
```

### 3. Authorize

For security, explicitly authorize each project:

```bash
dirvana allow
```

### 4. Reload

```bash
cd .. && cd -
# or
eval "$(dirvana export)"
```

### 5. Use Your Aliases!

```bash
gs        # runs: git status
build     # runs: go build -o bin/app ./cmd
echo $PROJECT_NAME  # prints: my-project
```

---

## 💼 Common Use Cases### 1. Terraform project with taskfile

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

### 2. Node.js Project

```yaml
aliases:
  dev: npm run dev
  build: npm run build
  test: npm test

env:
  NODE_ENV: development
  PORT: 3000
```

### 3. Docker Project

```yaml
aliases:
  dc: docker-compose
  up: docker-compose up -d
  down: docker-compose down
  logs: docker-compose logs -f

env:
  COMPOSE_PROJECT_NAME: my-project
```

### 4. Kubernetes Project

```yaml
aliases:
  k:
    command: kubectl
    completion: kubectl  # Auto-completion support
  kgp: kubectl get pods

env:
  KUBECONFIG: ./kubeconfig.yaml
  KUBE_NAMESPACE: production
```

### 5. Python Project

```yaml
aliases:
  py: python3
  test: pytest

functions:
  venv: |
    python3 -m venv .venv
    source .venv/bin/activate

env:
  PYTHONPATH: ./src
```

### 6. Global Configuration

Create `~/.config/dirvana/global.yml` for aliases available everywhere:

```yaml
aliases:
  ll: ls -lah
  g:
    command: git
    completion: git

functions:
  mkcd: |
    mkdir -p "$1" && cd "$1"

env:
  EDITOR: vim
```

---

## 🔥 Tips & Troubleshooting

### Configuration Hierarchy

Configs merge from global → parent → current:

```
~/.config/dirvana/global.yml    (all projects)
    ↓
~/projects/.dirvana.yml         (all subfolders)
    ↓
~/projects/backend/.dirvana.yml (this folder)
```

Child configurations override parent values.

### Useful Commands

```bash
dirvana status       # Check current config
dirvana edit         # Edit local config
dirvana edit --global # Edit global config
dirvana validate     # Check config syntax
dirvana allow        # Authorize directory
dirvana clean        # Clear cache
```

### Common Issues

**Aliases don't load?**
1. Check authorization: `dirvana status`
2. If "Not authorized": `dirvana allow`
3. Reload: `cd .. && cd -`

**"directory not authorized" error?**
```bash
dirvana allow
```

**Hook doesn't work after update?**
```bash
dirvana setup
source ~/.bashrc
```

**Debug mode:**
```bash
DIRVANA_LOG_LEVEL=debug cd your-project
```

### Auto-completion

For aliases that delegate to other commands, enable completion:

```yaml
aliases:
  k:
    command: kubecolor
    completion: kubectl  # Inherits kubectl's completion

  g: git
```

Now `k get <TAB>` and `g log <TAB>` work as expected!

---

## � More Information

- [Complete README](./README.md) - Full documentation
- [Examples](./examples/) - Real-world configurations
- [JSON Schema](./schema/dirvana.schema.json) - Config reference

---

## 🤝 Contributing

Found a bug? Want a feature? [Open an issue](https://github.com/NikitaCOEUR/dirvana/issues)!

---

**Happy coding with Dirvana! 🎉**
