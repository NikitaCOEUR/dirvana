# Completion Scripts Registry

This directory contains the registry of external completion scripts for tools that don't provide dynamic completion.

## Design Philosophy

**Dirvana uses bash completion scripts for ALL shells (bash, zsh, fish).**

### Why Bash-Only?

The completion system architecture is:

```
User Shell (bash/zsh/fish)
    ↓ (calls shell-specific completion function)
__dirvana_complete() or __dirvana_complete_zsh()
    ↓ (calls dirvana CLI)
dirvana completion (Go binary - shell-agnostic)
    ↓ (uses completion engine)
internal/completion/engine.go
    ↓ (detects and uses appropriate completer)
ScriptCompleter (always executes via "bash -c")
    ↓ (sources the bash script)
/usr/share/bash-completion/completions/tool
    ↓ (returns suggestions)
Suggestions → value\tdescription format
    ↓ (parsed by shell-specific formatter)
User Shell displays completions
```

**Key insight:** The completion backend executes scripts via `bash -c`, parsing the output into shell-agnostic suggestions (`value\tdescription` format). Each shell then formats these suggestions natively using their own completion system.

This means:
- ✅ **One script per tool** (bash)
- ✅ **Works for all shells** (bash, zsh, fish users get same suggestions)
- ✅ **Simpler maintenance** (no need for shell-specific variants)
- ✅ **Consistent behavior** across shells

## Registry Format

### Current Format (v1)

```yaml
version: "1"
description: "Registry of bash completion scripts"

tools:
  tool-name:
    description: "Brief description"
    homepage: "https://github.com/..."
    script:
      url: "https://raw.githubusercontent.com/.../completion.sh"
      sha256: "abc123..."  # Optional checksum
```

## Adding a Tool to the Registry

### Criteria for Inclusion

A tool should be added to the registry **ONLY IF**:

1. ✅ **No dynamic completion** - Tool does NOT support `--completion`, `__complete`, or env-based completion
2. ✅ **No system script** - Script is NOT available in standard locations like `/usr/share/bash-completion/completions/`
3. ✅ **Stable script available** - Completion script is available online and relatively stable

### Criteria for Exclusion

Do NOT add tools that:

- ❌ Have dynamic completion (kubectl, helm, terraform, etc.)
- ❌ Have system-installed scripts (git, docker, systemctl, npm, etc.)
- ❌ Can generate their own completion scripts (most modern Go CLIs)

### Examples

**Good candidates:**
- ✅ `govc` (VMware vSphere CLI) - Niche tool, no system script, no dynamic completion
- ✅ `gh` (GitHub CLI) - Popular but not always in bash-completion packages
- ✅ `k9s` (Kubernetes TUI) - Popular tool without system script
- ✅ `flux` (GitOps) - Cloud-native tool without system distribution

**Bad candidates:**
- ❌ `kubectl` - Has dynamic completion via `kubectl completion bash`
- ❌ `git` - Script available in `/usr/share/bash-completion/completions/git`
- ❌ `docker` - Script available in system packages
- ❌ `npm` - Installed with the package manager

**Tips:**
- When in doubt, try it with dirvana first. If it works without the registry, no need to add it.

## How It Works

### Script Discovery Order

When completing a command, `ScriptCompleter` searches for scripts in this order:

1. System locations:
   - `/usr/share/bash-completion/completions/tool`
   - `/usr/local/share/bash-completion/completions/tool`
   - `/etc/bash_completion.d/tool`
   - `/opt/homebrew/etc/bash_completion.d/tool` (macOS)

2. Dirvana cache:
   - `$CACHE_DIR/completion-scripts/bash/tool`

3. Registry download:
   - If tool is in registry and not found locally, download automatically

### Caching

Downloaded scripts are cached in `$HOME/.cache/dirvana/completion-scripts/bash/` by default.

The registry itself is cached for 7 days to reduce network calls.

## Contributing

To add a new tool to the registry:

1. Verify it meets the inclusion criteria above
2. Test the completion script manually
3. Add entry to `v1/completion-scripts.yml`
4. (Optional) Add SHA256 checksum for security
5. Submit PR with justification for inclusion

### Testing a Script Locally

```bash
# 1. Download the script
curl -o /tmp/test-completion.sh https://example.com/completion.sh

# 2. Source it in bash
source /tmp/test-completion.sh

# 3. Test completion (adjust tool name)
complete -p tool-name  # Should show completion is registered
tool-name <TAB>        # Should show suggestions
```
