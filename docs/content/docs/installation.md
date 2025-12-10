---
title: "Installation"
weight: 10
---

# Installation Guide

## Prerequisites

Dirvana works with:
- **Bash** (4.0+)
- **Zsh** (5.0+)
- **Fish** (3.0+)
- **Linux**, **macOS**, or **WSL**

---

## Method 1: Using Aqua (Recommended)

If you use [aqua](https://aquaproj.github.io/), add this to your `aqua.yaml`:

```yaml
registries:
- type: standard
  ref: v4.223.0 # renovate: depName=aquaproj/aqua-registry

packages:
  - name: NikitaCOEUR/dirvana@vX.Y.Z  # replace X.Y.Z with the desired version
```

Then run:

```bash
aqua install --link-only
```

---

## Method 2: Using go install

If you have Go installed (1.21+):

```bash
go install github.com/NikitaCOEUR/dirvana/cmd/dirvana@latest
```

---

## Method 3: Download Binary

### Linux

```bash
curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-linux-amd64 -o /usr/local/bin/dirvana
chmod +x /usr/local/bin/dirvana
```

For ARM64:
```bash
curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-linux-arm64 -o /usr/local/bin/dirvana
chmod +x /usr/local/bin/dirvana
```

### macOS

Intel:
```bash
curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-darwin-amd64 -o /usr/local/bin/dirvana
chmod +x /usr/local/bin/dirvana
```

Apple Silicon (M1/M2):
```bash
curl -L https://github.com/NikitaCOEUR/dirvana/releases/latest/download/dirvana-darwin-arm64 -o /usr/local/bin/dirvana
chmod +x /usr/local/bin/dirvana
```

---

## Setup Shell Hook

> [!IMPORTANT]
> **This step is required for automatic environment loading!**

Run the setup command:

```bash
dirvana setup
```

This will:
- Detect your shell (Bash, Zsh, or Fish)
- Add a hook to your `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`
- Enable automatic configuration loading on directory changes
- Install shell completion

**Reload your shell:**

```bash
source ~/.bashrc                      # For Bash
source ~/.zshrc                       # For Zsh
source ~/.config/fish/config.fish     # For Fish
```

Or simply restart your terminal.

---

## Verify Installation

Check that Dirvana is installed correctly:

```bash
dirvana version
```

You should see the version information.

---

## Shell Completion

The `dirvana setup` command automatically installs shell completion. If you need to install it manually:

**Bash:**
```bash
# One-time use (current shell only)
source <(dirvana completion bash)

# Permanent
dirvana completion bash > ~/.bash_completion.d/dirvana
```

**Zsh:**
```bash
# One-time use (current shell only)
source <(dirvana completion zsh)

# Permanent
dirvana completion zsh > "${fpath[1]}/_dirvana"
# Then reload: compinit
```

**Fish:**
```bash
# One-time use (current shell only)
dirvana completion fish | source

# Permanent
dirvana completion fish > ~/.config/fish/completions/dirvana.fish
```

---

## Next Steps

{{< button relref="/docs/quick-start" >}}Quick Start →{{< /button >}}
