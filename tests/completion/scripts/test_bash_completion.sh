#!/usr/bin/expect -f
# Test bash completion for dirvana-managed aliases
#
# Usage: test_bash_completion.sh <alias> <config_dir>
#
# This script:
# 1. Starts a bash shell
# 2. Sources dirvana export
# 3. Simulates TAB completion for the given alias
# 4. Outputs the completions

set timeout 5
set alias_name [lindex $argv 0]
set config_dir [lindex $argv 1]

if {$alias_name == ""} {
    send_user "Usage: test_bash_completion.sh <alias> <config_dir>\n"
    exit 1
}

# Start bash with no rc files
spawn bash --norc --noprofile

# Wait for prompt (bash-5.2# format)
expect {
    timeout { send_user "TIMEOUT waiting for initial prompt\n"; exit 1 }
    -re "bash.*# "
}

# Disable pagination for completions
send "bind 'set page-completions off'\r"
expect {
    timeout { send_user "TIMEOUT after disabling pagination\n"; exit 1 }
    -re "bash.*# "
}

# Source bash completion to load native tool completions
send "source /etc/bash_completion 2>/dev/null || true\r"
expect {
    timeout { send_user "TIMEOUT after sourcing bash_completion\n"; exit 1 }
    -re "bash.*# "
}

# Source individual completion files from bash_completion.d
send "for f in /etc/bash_completion.d/*; do source \$f 2>/dev/null || true; done\r"
expect {
    timeout { send_user "TIMEOUT after sourcing completion files\n"; exit 1 }
    -re "bash.*# "
}

# Change to config directory
send "cd $config_dir\r"
expect {
    timeout { send_user "TIMEOUT after cd command\n"; exit 1 }
    -re "bash.*# "
}

# Allow directory (suppress authorization prompt)
send "dirvana allow\r"
expect {
    timeout { send_user "TIMEOUT after dirvana allow\n"; exit 1 }
    -re "bash.*# "
}

# Export dirvana environment
send "eval \"\$(dirvana export)\"\r"
expect {
    timeout { send_user "TIMEOUT after dirvana export\n"; exit 1 }
    -re "bash.*# "
}

# Trigger completion (send alias + space + double TAB)
send "$alias_name \t\t"

# Wait a bit for completions to appear and capture them
# We need to wait for either "--More--" (paged output) or prompt
set output ""
expect {
    timeout {
        send_user "TIMEOUT waiting for completions\n"
        exit 1
    }
    -exact "--More--" {
        # Paged output - accumulate what we have so far and send Ctrl+C
        append output $expect_out(buffer)
        send "\x03"
        exp_continue
    }
    -re "bash.*# " {
        # Got back to prompt - accumulate final output
        append output $expect_out(buffer)
    }
}

# If we didn't get back to prompt yet, wait for it after Ctrl+C
if {![string match "*bash*#*" $output]} {
    send "\x03"
    expect {
        timeout { send_user "TIMEOUT after Ctrl+C\n"; exit 1 }
        -re "bash.*# "
    }
}

# Print captured completions to stdout (not to spawned shell)
send_user "COMPLETIONS_START\n"
send_user $output
send_user "COMPLETIONS_END\n"

# Exit shell
send "\025" ;# Ctrl+U to clear line
send "exit\r"
expect eof
