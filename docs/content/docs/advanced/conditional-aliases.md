---
title: "Conditional Aliases"
weight: 41
---

# Conditional Aliases

Execute different commands based on runtime conditions.

---

## Why Use Conditional Aliases?

Conditional aliases are useful for:
- Checking if required files or directories exist before running commands
- Verifying environment variables are set
- Ensuring required tools are installed
- Providing fallback commands when conditions aren't met

---

## Basic Conditions

### File Condition

Check if a file exists:

```yaml
aliases:
  dev:
    when:
      file: "package.json"
    command: npm run dev
    else: "echo 'Error: package.json not found'"
```

### Variable Condition

Check if environment variable is set:

```yaml
aliases:
  aws-deploy:
    when:
      var: "AWS_PROFILE"
    command: aws deploy push
    else: "echo 'Error: AWS_PROFILE not set. Run: export AWS_PROFILE=...'"
```

### Directory Condition

Check if directory exists:

```yaml
aliases:
  test:
    when:
      dir: "node_modules"
    command: npm test
    else: "echo 'Run npm install first'"
```

### Command Condition

Check if command exists in PATH:

```yaml
aliases:
  dc:
    when:
      command: "docker"
    command: docker compose
    else: "echo 'Docker not installed'"
```

---

## Multiple Conditions

### All Conditions (AND)

All conditions must be true:

```yaml
aliases:
  k:
    when:
      all:
        - var: "KUBECONFIG"       # Env var is set
        - file: "$KUBECONFIG"     # File exists
    command: kubectl --kubeconfig "$KUBECONFIG"
    else: kubectl  # Fallback
```

### Any Condition (OR)

At least one condition must be true:

```yaml
aliases:
  config-edit:
    when:
      any:
        - file: ".env.local"
        - file: ".env"
        - file: ".env.example"
    command: vim $(ls .env.local .env .env.example 2>/dev/null | head -1)
    else: "echo 'No config file found'"
```

### Nested Conditions

Complex logic with nested conditions:

```yaml
aliases:
  deploy:
    when:
      all:
        - var: "AWS_PROFILE"
        - command: "aws"
        - any:
            - file: ".env.production"
            - file: ".env"
    command: ./deploy.sh
    else: "echo 'Prerequisites not met'"
```

---

## Reusing Conditions with YAML Anchors

Define conditions once and reuse them:

```yaml
# Define reusable conditions
conditions:
  kubeconfig:
    when: &kubeconfig
      all:
        - var: "KUBECONFIG"
        - file: "$KUBECONFIG"

  talosconfig:
    when: &talosconfig
      file: "$TALOSCONFIG"

# Reuse across aliases
aliases:
  k:
    when: *kubeconfig
    command: kubectl
    completion: kubectl

  h:
    when: *kubeconfig
    command: helm

  t:
    when: *talosconfig
    command: talosctl
    completion: talosctl
```

---

## Environment Variable Expansion

File and directory paths support environment variable expansion:

```yaml
aliases:
  kconfig:
    when:
      file: "$HOME/.kube/config"  # $HOME is expanded
    command: kubectl --kubeconfig "$HOME/.kube/config"
```

---

## Error Messages

If conditions fail and no `else` is specified, a descriptive error is shown:

```bash
$ k get pods
Error: condition not met for alias 'k':
All conditions must be met:
  ✓ environment variable 'PROJECT_KUBECONFIG' is set
  ✗ file '/home/user/.kube/myproject-config' does not exist
  ✓ command 'kubectl' exists in PATH
```

---

## Real-World Example

```yaml
# Dynamic environment variables
env:
  PROJECT_KUBECONFIG:
    sh: echo "$HOME/.kube/$(basename $(pwd))-config"

# Define conditions
conditions:
  kube_ready:
    when: &kube_ready
      all:
        - var: "PROJECT_KUBECONFIG"
        - file: "$PROJECT_KUBECONFIG"
        - command: "kubectl"

# Use in aliases
aliases:
  k:
    when: *kube_ready
    command: kubectl --kubeconfig "$PROJECT_KUBECONFIG"
    else: "echo 'Generate kubeconfig first: task bootstrap'"
    completion: kubectl

  deploy:
    when: *kube_ready
    command: task deploy
    else: "echo 'Kubernetes config not ready'"
```

---

> [!NOTE]
> See `examples/conditional-aliases/` in the repository for more examples.
