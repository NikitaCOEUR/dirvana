---
title: "Auto-Completion"
weight: 43
---

# Auto-Completion

Dirvana aliases can inherit auto-completion from existing commands.

---

## Basic Completion

Use the `completion` field to inherit completion from another command:

```yaml
aliases:
  k:
    command: kubecolor
    completion: kubectl # Inherits kubectl completion

  kust: kustomize    # Native Inherits kustomize completion

  kubectl:            # Too lazy to set up completion for the kubectl command?
    command: kubectl  # No problem — just declare the command as is, and Dirvana will take care of the rest!

  tf:
    command: task terraform -- # Use terraform with task as wrapper
    completion: terraform      # And benefit from terraform completion
```

Now when you type:
```bash
k get <TAB>
# Shows: pods, services, deployments, ...
```

---

## How It Works

When you set `completion: kubectl`, Dirvana:
1. Detects your shell (Bash or Zsh)
2. Finds the completion function for `kubectl`
3. Registers the same completion for your alias `k`

---

## Supported Commands

Auto-completion works for commands with built-in completion support (Cobra CLI, urfave/cli, etc.) or those in the Dirvana completion registry.:

- **kubectl** - Kubernetes resources
- **git** - Branches, files, remotes
- **terraform** - Workspaces, resources
- **docker** - Containers, images
- **helm** - Charts, releases
- **And many more!**

---

## Completion Registry

Some commands don't have built-in completion support. Dirvana provides a [completion registry](https://github.com/NikitaCOEUR/dirvana/tree/main/registry) with custom completions.

### Using Registry Completions

The registry is automatically used when you specify a `completion` field:

```yaml
aliases:
  gov: govc # Uses govc completion from registry
```

### Contributing to Registry

You can contribute custom completions to the registry. See the [registry README](https://github.com/NikitaCOEUR/dirvana/tree/main/registry) for details.

---

## Completion for Custom Commands

If your command doesn't have completion support and isn't in the registry, contribute !

Add a completion script to the Dirvana registry. See [registry/README.md](https://github.com/NikitaCOEUR/dirvana/tree/main/registry).

---

## Examples

### Kubernetes Aliases

```yaml
aliases:
  k:
    command: kubecolor
    completion: kubectl

  kubectl: kubectl

  kust: kustomize

  kns: kubens

  kctx: kubectx

  h:
    command: helm
    completion: helm
```

### Git Aliases

```yaml
aliases:
  g:
    command: git
    completion: git

  gco:
    command: git checkout
    completion: git

  gp:
    command: git push
    completion: git
```

### Docker Aliases

```yaml
aliases:
  d:
    command: docker
    completion: docker

  dc:
    command: docker compose
    completion: docker
```

---

## Troubleshooting

### Completion Not Working

1. **Check dirvana status:**
   ```bash
   dirvana status # Installs completion
   ```

2. **Ensure dirvana is setup :**
   ```bash
   dirvana setup
   # Reload your shell or source ~/.bashrc / ~/.zshrc
   ```

3. **Is your dirvana allowed ?**
   ```bash
   dirvana allow
   ```

4. **Test dirvana completion:**
   ```bash
   dirvana completion <command>
   ```

### Completion for Wrapper Commands

If you're wrapping a command, use the wrapped command for completion:

```yaml
aliases:
  # Wrapper around kubectl
  k:
    command: kubecolor  # Colorized kubectl
    completion: kubectl # Use kubectl completion
```

---

> [!NOTE]
> See the [completion registry](https://github.com/NikitaCOEUR/dirvana/tree/main/registry) for available custom completions.
