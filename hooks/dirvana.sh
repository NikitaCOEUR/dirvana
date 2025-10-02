#!/usr/bin/env bash
# Dirvana shell hook for Bash/Zsh
# Source this file in your ~/.bashrc or ~/.zshrc

# Ensure dirvana is in PATH
if ! command -v dirvana &> /dev/null; then
    echo "Warning: dirvana not found in PATH" >&2
    return 1
fi

# Main hook function
__dirvana_hook() {
    # Only run if we're in an interactive shell
    if [[ ! $- =~ i ]]; then
        return 0
    fi

    # Export shell code from dirvana
    local shell_code
    shell_code=$(dirvana export 2>&1)
    local exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        # Evaluate the generated shell code
        if [[ -n "$shell_code" ]]; then
            eval "$shell_code"
        fi
    else
        # Only show errors if log level is debug
        if [[ "${DIRVANA_LOG_LEVEL}" == "debug" ]]; then
            echo "$shell_code" >&2
        fi
    fi
}

# Hook into cd command for Bash
if [[ -n "$BASH_VERSION" ]]; then
    __dirvana_cd() {
        builtin cd "$@" && __dirvana_hook
    }
    alias cd='__dirvana_cd'
fi

# Hook into chpwd for Zsh
if [[ -n "$ZSH_VERSION" ]]; then
    autoload -U add-zsh-hook
    add-zsh-hook chpwd __dirvana_hook
fi

# Run on shell startup
__dirvana_hook
