#!/usr/bin/env fish
# Report whether dirvana considers each command already covered by fish.
#
# Usage: test_fish_coverage.sh <config_dir> <command>...
#
# This is the decision that keeps dirvana out of the way: when fish knows a
# command, `complete -w` forwards to its completions and asking dirvana would
# cost a fork and a run of the tool on every keypress - a few hundred
# milliseconds, felt at the prompt.

set -l config_dir $argv[1]
set -l commands $argv[2..-1]

if test -z "$config_dir"
    echo "Usage: test_fish_coverage.sh <config_dir> <command>..." >&2
    exit 1
end

cd $config_dir; or exit 1

eval (dirvana export 2>/dev/null | string collect)

echo "COVERAGE_START"
for cmd in $commands
    if __dirvana_fish_covers $cmd
        echo "$cmd covered"
    else
        echo "$cmd uncovered"
    end
end
echo "COVERAGE_END"
