---
title: "Quick Start"
weight: 20
---

# Quick Start Guide

Get started with Dirvana in 5 minutes!

---

## Step 1: Create a Project Configuration

Navigate to your project directory:

```bash
cd ~/projects/myproject
```

Initialize Dirvana (creates `.dirvana.yml`):

```bash
dirvana init
```

---

## Step 2: Edit Configuration

Open the configuration file:

```bash
dirvana edit
```

Or edit `.dirvana.yml` manually:

```yaml
# Simple aliases
aliases:
  ll: ls -lah
  gs: git status
  build: go build -o bin/app ./cmd

# Aliases with auto-completion
aliases:
  k:
    command: kubectl
    completion: kubectl  # Now 'k <TAB>' works!

# Functions
functions:
  mkcd: |
    mkdir -p "$1" && cd "$1"

# Environment variables
env:
  PROJECT_NAME: myproject
  LOG_LEVEL: debug

  # Dynamic values
  GIT_BRANCH:
    sh: git rev-parse --abbrev-ref HEAD
```

---

## Step 3: Authorize the Project

For security, you must authorize each project:

```bash
dirvana allow
```

---

## Step 4: Load the Configuration

Reload by changing directories:
```bash
cd .. && cd -
```

Or manually:
```bash
eval "$(dirvana export)"
```

---

## Step 5: Test Your Configuration

```bash
# Test alias
ll

# Test function
mkcd test_dir

# Test auto-completion
k get <TAB>
# Shows: pods, services, deployments, ...

# Check environment variable
echo $PROJECT_NAME
# Output: myproject
```

---

## Common Use Cases

{{< tabs "examples" >}}

{{< tab "Kubernetes" >}}
```yaml
aliases:
  k:
    command: kubectl
    completion: kubectl
  kns: kubens
  kctx: kubectx

env:
  KUBECONFIG:
    sh: echo "$HOME/.kube/$(basename $(pwd))-config"
```
{{< /tab >}}

{{< tab "Docker" >}}
```yaml
aliases:
  dc: docker compose
  up: docker compose up -d
  down: docker compose down
  logs: docker compose logs -f

env:
  COMPOSE_PROJECT_NAME: myproject
```
{{< /tab >}}

{{< tab "Terraform" >}}
```yaml
aliases:
  tf:
    command: terraform
    completion: terraform
  plan: terraform plan
  apply: terraform apply

env:
  TF_LOG: debug
  TF_VAR_environment: dev
```
{{< /tab >}}

{{< tab "Node.js" >}}
```yaml
aliases:
  dev: npm run dev
  build: npm run build
  test: npm test

env:
  NODE_ENV: development
```
{{< /tab >}}
{{< /tabs >}}

---

## Next Steps

{{< button relref="/docs/configuration" >}}Configuration Guide{{< /button >}}
{{< button relref="/docs/advanced" >}}Advanced Features{{< /button >}}
