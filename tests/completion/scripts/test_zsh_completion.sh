#!/usr/bin/env zsh
# Capture the completions zsh offers for a dirvana-managed alias.
#
# Usage: test_zsh_completion.sh <alias> <config_dir> [subcommand]
#
# Zsh has no equivalent to fish's `complete -C`, and driving a terminal
# through expect meant scraping a rendered screen - pagination, escape
# codes and prompt guessing included, which is why these tests were
# disabled. Instead we call the completion system the way zsh itself
# does: look up the function registered for the alias, set the `words`
# and `CURRENT` variables a real completion would see, and intercept the
# builtins the function reports its matches through.

emulate -L zsh
setopt no_unset

local alias_name="${1:-}"
local config_dir="${2:-}"
local subcommand="${3:-}"

if [[ -z "$alias_name" ]]; then
    print -u2 "Usage: test_zsh_completion.sh <alias> <config_dir> [subcommand]"
    exit 1
fi

[[ -n "$config_dir" ]] && cd "$config_dir"

autoload -Uz compinit && compinit -u 2>/dev/null

# Load the aliases and their completion registrations. DIRVANA_SHELL is
# explicit because this script is not run from an interactive zsh, so
# shell auto-detection would look at the wrong parent process.
eval "$(DIRVANA_SHELL=zsh dirvana export 2>/dev/null)"

# Completion functions report matches through _describe or compadd.
# Replacing both captures what the user would be offered, whether the
# function is dirvana's own or the tool's native one.
_describe() {
    local arrname=${@[-1]}
    local entry
    for entry in ${(P)arrname}; do
        # entries are "value:description"
        print -- "${entry%%:*}"
    done
    return 0
}

compadd() {
    local -a values
    local skip_next=0 arg
    for arg in "$@"; do
        if (( skip_next )); then
            skip_next=0
            continue
        fi
        case "$arg" in
            # options carrying a value
            -d|-X|-J|-V|-P|-S|-p|-s|-M|-W|-F|-i|-a) skip_next=1 ;;
            -*) ;;
            --) ;;
            *) values+=("$arg") ;;
        esac
    done
    print -l -- "${values[@]}"
    return 0
}

# Never let a miss fall through to listing the filesystem
_files() { return 1 }
_message() { return 0 }

# Build the line being completed, as the completion system would see it:
# words holds the tokens, CURRENT is the 1-based index of the one being
# completed - an empty trailing token when completing a fresh word.
local -a words
if [[ -n "$subcommand" ]]; then
    words=("$alias_name" "$subcommand" "")
else
    words=("$alias_name" "")
fi
local CURRENT=${#words}

# Use whatever zsh registered for this alias: dirvana's function, or the
# tool's native completion when dirvana delegated to it
local comp_func=${_comps[$alias_name]:-}
if [[ -z "$comp_func" ]]; then
    print -u2 "no completion registered for '$alias_name'"
    exit 1
fi

print "COMPLETIONS_START"
$comp_func 2>/dev/null
print "COMPLETIONS_END"
