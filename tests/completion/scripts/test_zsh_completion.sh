#!/usr/bin/expect -f
# Test zsh completion for dirvana-managed aliases
#
# Usage: test_zsh_completion.sh <alias> <config_dir>
#
# This script:
# 1. Starts a zsh shell
# 2. Sources dirvana export
# 3. Simulates TAB completion for the given alias
# 4. Outputs the completions

set timeout 5
set alias_name [lindex $argv 0]
set config_dir [lindex $argv 1]

if {$alias_name == ""} {
    send_user "Usage: test_zsh_completion.sh <alias> <config_dir>\n"
    exit 1
}

# Start zsh with no rc files initially
spawn zsh --no-rcs

# Wait for prompt (can be % or hostname#)
expect {
    timeout { send_user "TIMEOUT waiting for initial prompt\n"; exit 1 }
    -re {(%|#)}
}

# Configure zsh completion system
send "autoload -Uz compinit\r"
expect {
    timeout { send_user "TIMEOUT after autoload compinit\n"; exit 1 }
    -re {(%|#)}
}

send "compinit -i\r"
expect {
    timeout { send_user "TIMEOUT after compinit\n"; exit 1 }
    -re {(%|#)}
}

# Disable the "do you wish to see all N possibilities" prompt
# Set LISTMAX to a very high number so zsh never asks
send "LISTMAX=9999\r"
expect {
    timeout { send_user "TIMEOUT after LISTMAX\n"; exit 1 }
    -re {(%|#)}
}

# Load native tool completions
send "kubectl completion zsh > /tmp/_kubectl && source /tmp/_kubectl 2>/dev/null || true\r"
expect {
    timeout { send_user "TIMEOUT after kubectl completion\n"; exit 1 }
    -re {(%|#)}
}

send "terraform -install-autocomplete 2>/dev/null || true\r"
expect {
    timeout { send_user "TIMEOUT after terraform autocomplete\n"; exit 1 }
    -re {(%|#)}
}

send "aqua completion zsh > /tmp/_aqua && source /tmp/_aqua 2>/dev/null || true\r"
expect {
    timeout { send_user "TIMEOUT after aqua completion\n"; exit 1 }
    -re {(%|#)}
}

# Change to config directory
send "cd $config_dir\r"
expect {
    timeout { send_user "TIMEOUT after cd command\n"; exit 1 }
    -re {(%|#)}
}

# Allow directory (suppress authorization prompt and output)
send "dirvana allow >/dev/null 2>&1\r"
expect {
    timeout { send_user "TIMEOUT after dirvana allow\n"; exit 1 }
    -re {(%|#)}
}

# Export dirvana environment
send "eval \"\$(dirvana export)\"\r"
expect {
    timeout { send_user "TIMEOUT after dirvana export\n"; exit 1 }
    -re {(%|#)}
}

# Trigger completion (send alias + space + double TAB)
send "$alias_name \t\t"

# Wait a bit for completions to appear and capture them
set output ""
expect {
    -timeout 30
    timeout {
        send_user "TIMEOUT waiting for completions after 30s\n"
        exit 1
    }
    -ex {--More--} {
        # Paged output - accumulate what we have so far and send Ctrl+C
        append output $expect_out(buffer)
        send "\x03"
        exp_continue
    }
    -re {(%|#)} {
        # Got back to prompt - accumulate final output
        append output $expect_out(buffer)
    }
}

# If we didn't get back to prompt yet, wait for it after Ctrl+C
if {![string match "*%*" $output] && ![string match "*#*" $output]} {
    send "\x03"
    expect {
        timeout { send_user "TIMEOUT after Ctrl+C\n"; exit 1 }
        -re {(%|#)}
    }
}

# Print captured completions to stdout (not to spawned shell)
send_user "COMPLETIONS_START\n"
send_user -- $output
send_user "\n"
send_user "COMPLETIONS_END\n"

# Exit shell
send "\025" ;# Ctrl+U to clear line
send "exit\r"
expect eof
