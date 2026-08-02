#!/usr/bin/env fish
# Capture the completions fish offers for a dirvana-managed alias.
#
# Usage: test_fish_completion.sh <alias> <config_dir> [subcommand]
#
# Unlike bash and zsh, fish exposes completion as a plain command:
# `complete -C"<line>"` returns exactly what TAB would offer. No expect,
# no pseudo-terminal, no sleeping - so this reads the same result the
# user gets, deterministically.

set -l alias_name $argv[1]
set -l config_dir $argv[2]
set -l subcommand $argv[3]

if test -z "$alias_name"
    echo "Usage: test_fish_completion.sh <alias> <config_dir> [subcommand]" >&2
    exit 1
end

if test -n "$config_dir"
    cd $config_dir; or exit 1
end

# Load the aliases and their completion registrations
eval (dirvana export 2>/dev/null | string collect)

# Build the line being completed: "alias " or "alias subcommand "
set -l line "$alias_name "
if test -n "$subcommand"
    set line "$alias_name $subcommand "
end

# Keep only the value of each suggestion; the parser splits on whitespace
# and would otherwise take description words for completions
echo "COMPLETIONS_START"
complete -C"$line" | string replace -r '\t.*' ''
echo "COMPLETIONS_END"
