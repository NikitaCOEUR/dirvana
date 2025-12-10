#!/usr/bin/expect -f
# Test fish completion for dirvana-managed aliases
#
# Usage: test_fish_completion.sh <alias> <config_dir>
#
# This script:
# 1. Starts a fish shell
# 2. Sources dirvana export
# 3. Simulates TAB completion for the given alias
# 4. Outputs the completions

set timeout 5
set alias_name [lindex $argv 0]
set config_dir [lindex $argv 1]

if {$alias_name == ""} {
    send_user "Usage: test_fish_completion.sh <alias> <config_dir>\n"
    exit 1
}

# Start fish with no config files
spawn fish --no-config

# Wait for prompt
expect {
    timeout { send_user "TIMEOUT waiting for initial prompt\n"; exit 1 }
    -re "> "
}

# Navigate to config directory if provided
if {$config_dir != ""} {
    send "cd $config_dir\r"
    expect {
        timeout { send_user "TIMEOUT after cd\n"; exit 1 }
        -re "> "
    }
}

# Source dirvana export (fish syntax)
send "eval (dirvana export)\r"
expect {
    timeout { send_user "TIMEOUT after dirvana export\n"; exit 1 }
    -re "> "
}

# Trigger completion by typing the alias and TAB
# Fish shows completions in a different format than bash/zsh
send "$alias_name \t"

# Wait a bit for completions to appear
sleep 0.5

# Capture the output
expect {
    timeout { send_user "TIMEOUT waiting for completions\n"; exit 1 }
    -re "> "
}

# Exit fish
send "exit\r"
expect eof
