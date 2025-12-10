---
title: "Dirvana"
type: docs
---

# Dirvana

<p align="center">
  <img src="DirvanaLogo.png" alt="Dirvana Logo" width="300">
</p>

<p align="center">
  <strong>Reach directory nirvana</strong><br>
  Per-project aliases, functions, and env vars that auto-load when you cd. Zero friction, zero pollution.
</p>

<p align="center">
  <a href="https://github.com/NikitaCOEUR/dirvana/releases"><img src="https://img.shields.io/github/v/release/NikitaCOEUR/dirvana?sort=semver&display_name=release&style=flat&labelColor=353c4a&color=ffd66b" alt="Release"></a>
  <a href="https://codecov.io/gh/NikitaCOEUR/dirvana"><img src="https://codecov.io/gh/NikitaCOEUR/dirvana/graph/badge.svg?token=IBRJQQQB3V" alt="Coverage"></a>
  <a href="https://github.com/NikitaCOEUR/dirvana/blob/main/LICENSE"><img src="https://img.shields.io/github/license/NikitaCOEUR/dirvana" alt="License"></a>
</p>

---

> [!WARNING]
> **Beta Status - We Need Your Feedback!**
>
> Dirvana is currently in beta and actively seeking testers to help validate its functionality across different configurations and use cases.
>
> Found a bug? Have a feature request? Please [open an issue](https://github.com/NikitaCOEUR/dirvana/issues)!

## The Problem

Managing different shell environments for different projects is tedious:
- Cluttered `.bashrc` files with project-specific aliases
- Manual sourcing of environment files
- Forgetting to unload configurations when switching projects
- Global shell pollution

## The Solution

**Dirvana automatically loads project-specific configurations when you enter a directory and unloads them when you leave.**

### Before Dirvana

```bash
$ cd ~/projects/terraform
$ export TF_LOG=debug
$ alias tf="task terraform --"
$ alias plan="task terraform -- plan"
$ alias apply="task terraform -- apply"
# ... and don't forget to unset everything when leaving!
```

### With Dirvana

```yaml
# .dirvana.yml
aliases:
  tf:
    command: task terraform --
    completion: terraform  # Auto-completion works!
  plan: task terraform -- plan
  apply: task terraform -- apply

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

## Key Features

- **Fast** - <10ms overhead with intelligent caching
- **Secure** - Authorization system prevents untrusted configs
- **Hierarchical** - Merge configurations from parent directories
- **Simple** - YAML configuration with JSON Schema validation
- **Compatible** - Works with Bash, Zsh, and Fish
- **Auto-completion** - Inherits completion from aliased commands

---

## Quick Start

1. **Install Dirvana**
   ```bash
   go install github.com/NikitaCOEUR/dirvana/cmd/dirvana@latest
   ```

2. **Setup shell hook**
   ```bash
   dirvana setup
   source ~/.bashrc  # or ~/.zshrc, or ~/.config/fish/config.fish
   ```

3. **Create configuration**
   ```bash
   cd your-project
   dirvana init
   dirvana allow
   ```

**That's it!** Your environment is now automatically managed.

---

## Documentation

{{< button relref="/docs/installation" >}}Installation{{< /button >}}
{{< button relref="/docs/quick-start" >}}Quick Start{{< /button >}}
{{< button relref="/docs/configuration" >}}Configuration{{< /button >}}
{{< button relref="/docs/advanced" >}}Advanced Features{{< /button >}}
