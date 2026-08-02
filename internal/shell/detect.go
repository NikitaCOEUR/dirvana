package shell

import (
	"fmt"
	"os"
	"strings"
)

// Supported shells
const (
	// Bash represents the bash shell
	Bash = "bash"
	// Zsh represents the zsh shell
	Zsh = "zsh"
	// Fish represents the fish shell
	Fish = "fish"
)

// Detect determines the shell type based on the flag or environment,
// defaulting to bash when nothing can be detected.
func Detect(shellFlag string) string {
	if s := DetectRaw(shellFlag); s != "" {
		return s
	}
	return Bash
}

// DetectRaw determines the shell type based on the flag or environment.
// Unlike Detect it returns "" when no shell could be identified, letting
// callers distinguish a real detection from the bash fallback.
// Detection priority:
// 1. Explicit shell flag (if not "auto")
// 2. DIRVANA_SHELL env var (set by hook, most reliable)
// 3. Shell-specific version variables (FISH_VERSION, ZSH_VERSION, BASH_VERSION)
// 4. Parent process detection (Linux via /proc)
// 5. SHELL env var (login shell, less reliable)
func DetectRaw(shellFlag string) string {
	if shellFlag != "auto" && shellFlag != "" {
		return shellFlag
	}

	// Try DIRVANA_SHELL env var (set by hook)
	if dirvanaShell := os.Getenv("DIRVANA_SHELL"); dirvanaShell != "" {
		return dirvanaShell
	}

	// Try shell-specific version variables (most reliable runtime detection)
	if os.Getenv("FISH_VERSION") != "" {
		return Fish
	}
	if os.Getenv("ZSH_VERSION") != "" {
		return Zsh
	}
	if os.Getenv("BASH_VERSION") != "" {
		return Bash
	}

	// Try to detect from parent process (works on Linux)
	if parentShell := detectShellFromParentProcess(); parentShell != "" {
		return parentShell
	}

	// Try SHELL env var (usually set to login shell, less reliable)
	if sh := os.Getenv("SHELL"); sh != "" {
		return parseShellName(sh)
	}

	return ""
}

// parseShellName detects the shell type from any string naming it: a path
// like "/usr/bin/fish" or a process command line
func parseShellName(s string) string {
	s = strings.ToLower(s)
	if strings.Contains(s, "fish") {
		return Fish
	}
	if strings.Contains(s, "zsh") {
		return Zsh
	}
	if strings.Contains(s, "bash") {
		return Bash
	}
	return ""
}

// detectShellFromParentProcess tries to detect the shell by reading the parent process name
func detectShellFromParentProcess() string {
	// Try to read /proc/$PPID/cmdline (Linux)
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getppid()))
	if err == nil {
		return parseShellName(string(cmdline))
	}

	// On macOS, we could use ps, but that's more complex
	// For now, return empty to fall back to other detection methods
	return ""
}

// BinaryPath returns the path of the running dirvana binary so that
// generated hooks work even when dirvana is not on $PATH. Falls back to
// the bare name if the executable path cannot be resolved.
func BinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "dirvana"
	}
	return exe
}
