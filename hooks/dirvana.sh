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

    # Skip if Dirvana is explicitly disabled
    if [[ "${DIRVANA_ENABLED:-true}" == "false" ]]; then
        return 0
    fi

    # Critical: Don't run if stdin is not the terminal
    # This prevents interference with TUI apps (fzf, vim, etc.)
    if [[ ! -t 0 ]]; then
        return 0
    fi

    # Export shell code from dirvana
    # Note: stderr is preserved to show warnings (like unauthorized directories)
    # dirvana auto-detects the shell from the parent process
    local shell_code
    shell_code=$(dirvana export)
    local exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        # Evaluate the generated shell code
        if [[ -n "$shell_code" ]]; then
            eval "$shell_code" 2>/dev/null
        fi
    fi
}

# Hook into PROMPT_COMMAND for Bash
if [[ -n "$BASH_VERSION" ]]; then
    # Only run if directory changed
    __dirvana_hook_wrapper() {
        if [[ "$PWD" != "${DIRVANA_PREV_DIR:-}" ]]; then
            export DIRVANA_PREV_DIR="$PWD"
            __dirvana_hook
        fi
    }

    # Add to PROMPT_COMMAND (runs before prompt is displayed)
    if [[ -z "${PROMPT_COMMAND}" ]]; then
        PROMPT_COMMAND="__dirvana_hook_wrapper"
    elif [[ ! "${PROMPT_COMMAND}" =~ __dirvana_hook ]]; then
        PROMPT_COMMAND="__dirvana_hook_wrapper;${PROMPT_COMMAND}"
    fi
fi

# Hook into chpwd for Zsh
if [[ -n "$ZSH_VERSION" ]]; then
    autoload -U add-zsh-hook
    add-zsh-hook chpwd __dirvana_hook
fi

# Run on shell startup only if we're in an interactive top-level shell
# Skip if stdin is not a terminal (prevents issues with TUI apps launching subshells)
if [[ $- =~ i ]] && [[ -t 0 ]]; then
    __dirvana_hook
fi
