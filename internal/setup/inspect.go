package setup

import (
	"os"
	"os/exec"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/shell"
)

// HookState describes the dirvana hook a shell would actually run.
type HookState struct {
	// Installed is true as soon as any strategy's hook is in place, not just
	// the one this dirvana would pick today.
	Installed bool
	// File is where the hook code lives - the file to open to read or remove
	// it, which is not always the shell's RC file.
	File string
	// RCFile is the shell config that pulls the hook in.
	RCFile string
	// Outdated marks hook code that no longer matches what this dirvana
	// generates. The path baked into it is deliberately not part of the
	// comparison: a hook installed from another location still works.
	Outdated bool
	// BinaryPath is the dirvana the hook invokes, and Broken says that binary
	// can no longer be found - the one case where the hook is silently dead.
	BinaryPath string
	Broken     bool
}

// InspectHook reports on the hook installed for a shell, whichever strategy
// put it there.
//
// Asking a single strategy is not enough: SelectInstallStrategy returns the
// one that would be used *today*, and a perfectly working hook installed the
// other way then reads as missing.
func InspectHook(shellName string) (HookState, error) {
	strategies, err := allStrategies(shellName)
	if err != nil {
		return HookState{}, err
	}

	// The preferred strategy comes first, so it wins when several match
	for _, strategy := range strategies {
		if !strategy.IsInstalled() {
			continue
		}

		state := HookState{
			Installed: true,
			File:      strategy.HookFile(),
			RCFile:    strategy.GetRCFile(),
		}

		installed, err := os.ReadFile(state.File)
		if err != nil {
			// Registered but unreadable: reinstalling is the way out
			state.Outdated = true
			return state, nil
		}

		binaryPath, matches := matchHook(shellName, string(installed))
		state.Outdated = !matches
		state.BinaryPath = binaryPath
		state.Broken = binaryPath != "" && !binaryExists(binaryPath)

		return state, nil
	}

	// Nothing installed: report where the preferred strategy would put it
	return HookState{
		File:   strategies[0].HookFile(),
		RCFile: strategies[0].GetRCFile(),
	}, nil
}

// allStrategies lists every way a hook could have been installed for a shell,
// preferred one first.
func allStrategies(shellName string) ([]InstallStrategy, error) {
	preferred, err := SelectInstallStrategy(shellName)
	if err != nil {
		return nil, err
	}
	strategies := []InstallStrategy{preferred}

	if shellName == shell.Fish {
		return strategies, nil
	}

	if _, ok := preferred.(*DropInStrategy); !ok {
		if dropIn, err := NewDropInStrategy(shellName); err == nil {
			strategies = append(strategies, dropIn)
		}
	}
	if _, ok := preferred.(*ExternalHookStrategy); !ok {
		if external, err := NewExternalHookStrategy(shellName); err == nil {
			strategies = append(strategies, external)
		}
	}

	return strategies, nil
}

// hookBinarySentinel stands in for the dirvana path while the hook template is
// being matched. NUL bytes cannot occur in a real path, so it can never be
// confused with hook code.
const hookBinarySentinel = "\x00dirvana-binary\x00"

// matchHook reports the dirvana path an installed hook invokes, and whether
// the rest of its code is what this dirvana generates.
//
// A verbatim comparison against the freshly generated hook would call every
// hook installed from another location outdated - which is most of them, since
// the path is baked in at install time and dirvana gets moved, upgraded in
// place, or run from a download.
func matchHook(shellName, installed string) (binaryPath string, matches bool) {
	pattern, err := shell.GenerateHookCode(shellName, hookBinarySentinel)
	if err != nil {
		return "", false
	}

	prefix, rest, found := strings.Cut(pattern, hookBinarySentinel)
	if !found {
		return "", installed == pattern
	}
	if !strings.HasPrefix(installed, prefix) {
		return "", false
	}

	// The path runs until the literal text the template puts after it
	separator := rest
	if i := strings.Index(rest, hookBinarySentinel); i >= 0 {
		separator = rest[:i]
	}

	after := installed[len(prefix):]
	candidate := after
	if separator != "" {
		end := strings.Index(after, separator)
		if end < 0 {
			return "", false
		}
		candidate = after[:end]
	}

	// Rebuilding with the candidate turns the rest into an exact comparison,
	// however many times the template repeats the path
	expected, err := shell.GenerateHookCode(shellName, candidate)
	if err != nil {
		return "", false
	}
	return candidate, expected == installed
}

// binaryExists reports whether the hook's dirvana can still be run. LookPath
// answers for both shapes a hook can hold - an absolute path, or a bare name
// to resolve through PATH - and checks the file is actually executable.
func binaryExists(path string) bool {
	_, err := exec.LookPath(path)
	return err == nil
}
