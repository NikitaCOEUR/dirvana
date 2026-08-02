package shell

// Executable returns the executable name for the given shell type.
// The shell constants are the executable names, so this only has to
// reject anything unknown.
func Executable(shellType string) string {
	switch shellType {
	case Bash, Zsh, Fish:
		return shellType
	default:
		// Detect always returns at least Bash, but keep bash as fallback for safety
		// Note: sh (dash/busybox) is not supported as it doesn't support required flags
		return Bash
	}
}

// invokeFlags returns the optimization flags for the given shell type
func invokeFlags(shellType string) []string {
	switch shellType {
	case Fish:
		return []string{"--no-config"}
	case Bash:
		return []string{"--norc", "--noprofile"}
	case Zsh:
		return []string{"--no-rcs"}
	default:
		return []string{}
	}
}

// argSyntax returns the positional-argument syntax for the given shell type
func argSyntax(shellType string) string {
	if shellType == Fish {
		return " $argv"
	}
	return ` "$@"`
}

// needsExtraShellArg returns true if the shell needs an extra shell argument in argv
func needsExtraShellArg(shellType string) bool {
	return shellType != Fish
}

// BuildArgs builds the argument list to execute command through the given shell
func BuildArgs(shellExec, shellType, command string, args []string) []string {
	flags := invokeFlags(shellType)
	syntax := argSyntax(shellType)
	needsExtra := needsExtraShellArg(shellType)

	var argv []string
	argv = append(argv, shellExec)
	argv = append(argv, flags...)

	if len(args) > 0 {
		argv = append(argv, "-c", command+syntax)
		if needsExtra {
			argv = append(argv, shellExec) // bash/zsh: $0 separator
		} else {
			argv = append(argv, "--") // fish: end-of-options marker
		}
		argv = append(argv, args...)
	} else {
		argv = append(argv, "-c", command)
	}
	return argv
}
