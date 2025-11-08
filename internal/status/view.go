package status

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors and styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14"))

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// Render renders the status data to a string
func Render(data *Data) string {
	var b strings.Builder

	// Header
	b.WriteString(renderHeader(data))
	b.WriteString("\n")

	// System & Installation
	b.WriteString(renderSystemInfo(data))
	b.WriteString("\n")

	// Authorization (only if there are configs)
	if data.HasAnyConfig {
		b.WriteString(renderAuthInfo(data))
		b.WriteString("\n")
	}

	// Configuration hierarchy
	b.WriteString(renderConfigHierarchy(data))
	b.WriteString("\n")

	// Aliases
	if len(data.Aliases) > 0 {
		b.WriteString(renderAliases(data))
		b.WriteString("\n")
	}

	// Functions
	if len(data.Functions) > 0 {
		b.WriteString(renderFunctions(data))
		b.WriteString("\n")
	}

	// Environment variables
	if len(data.EnvStatic) > 0 || len(data.EnvShell) > 0 {
		b.WriteString(renderEnvVars(data))
		b.WriteString("\n")
	}

	// Flags
	if len(data.Flags) > 0 {
		b.WriteString(renderFlags(data))
		b.WriteString("\n")
	}

	// Cache
	b.WriteString(renderCacheInfo(data))
	b.WriteString("\n")

	// Completion
	b.WriteString(renderCompletionInfo(data))

	return b.String()
}

func renderHeader(data *Data) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📂 Current directory: ") + valueStyle.Render(data.CurrentDir) + "\n")
	b.WriteString(titleStyle.Render("📦 Version: ") + valueStyle.Render(data.Version))
	return b.String()
}

func renderSystemInfo(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("⚙️  System & Installation:") + "\n")

	b.WriteString("   " + keyStyle.Render("Shell: ") + valueStyle.Render(data.Shell) + "\n")

	if data.HookInstalled {
		b.WriteString("   " + keyStyle.Render("Hook: ") + successStyle.Render("✓ Installed") + "\n")
		if data.RCFile != "" {
			b.WriteString("   " + keyStyle.Render("RC file: ") + subtleStyle.Render(data.RCFile) + "\n")
		}
	} else {
		b.WriteString("   " + keyStyle.Render("Hook: ") + errorStyle.Render("✗ Not installed") + "\n")
		if data.Shell != "unknown" {
			b.WriteString("   " + warningStyle.Render(fmt.Sprintf("Run 'dirvana setup %s' to install", data.Shell)) + "\n")
		}
	}

	b.WriteString("   " + keyStyle.Render("Cache path: ") + subtleStyle.Render(data.CachePath) + "\n")
	b.WriteString("   " + keyStyle.Render("Auth path: ") + subtleStyle.Render(data.AuthPath))

	return b.String()
}

func renderAuthInfo(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("🔒 Authorization:") + "\n")

	if data.Authorized {
		b.WriteString("   " + successStyle.Render("✓ Authorized"))
	} else {
		b.WriteString("   " + errorStyle.Render("✗ Not authorized") + "\n")
		b.WriteString("   " + warningStyle.Render(fmt.Sprintf("Run 'dirvana allow %s' to authorize", data.CurrentDir)))
	}

	return b.String()
}

func renderConfigHierarchy(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("📝 Configuration hierarchy:") + "\n")

	hasGlobal := data.GlobalConfig != nil && data.GlobalConfig.Exists
	if len(data.LocalConfigs) == 0 && !hasGlobal {
		b.WriteString("   " + subtleStyle.Render("No configuration files found"))
		return b.String()
	}

	idx := 1
	if hasGlobal {
		status := successStyle.Render("✓")
		note := ""
		if !data.GlobalConfig.Loaded {
			status = errorStyle.Render("✗")
			note = subtleStyle.Render(" (ignored)")
		}
		b.WriteString(fmt.Sprintf("   %d. %s %s%s\n",
			idx,
			subtleStyle.Render(data.GlobalConfig.Path+" (global)"),
			status,
			note))
		idx++
	}

	for _, cfg := range data.LocalConfigs {
		status := successStyle.Render("✓")
		statusText := ""
		if !cfg.Authorized {
			status = errorStyle.Render("✗")
			statusText = subtleStyle.Render(" (not authorized)")
		} else if !cfg.Loaded {
			status = errorStyle.Render("✗")
			statusText = subtleStyle.Render(" (not loaded)")
		} else if cfg.LocalOnly {
			statusText = subtleStyle.Render(" (local only)")
		}

		b.WriteString(fmt.Sprintf("   %d. %s %s%s\n",
			idx,
			valueStyle.Render(cfg.Path),
			status,
			statusText))
		idx++
	}

	// Remove trailing newline
	return strings.TrimSuffix(b.String(), "\n")
}

func renderAliases(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("🔗 Aliases:") + "\n")

	for name, info := range data.Aliases {
		b.WriteString(fmt.Sprintf("   %s → %s",
			keyStyle.Render(name),
			valueStyle.Render(info.Command)))

		// Show conditional information if present
		if info.HasWhen {
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("      %s %s",
				subtleStyle.Render("when:"),
				subtleStyle.Render(info.WhenSummary)))

			if info.Else != "" {
				b.WriteString("\n")
				b.WriteString(fmt.Sprintf("      %s %s",
					subtleStyle.Render("else:"),
					subtleStyle.Render(info.Else)))
			}
		}

		b.WriteString("\n")
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func renderFunctions(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("⚙️  Functions:") + "\n")

	for _, fn := range data.Functions {
		b.WriteString("   " + valueStyle.Render(fn+"()") + "\n")
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func renderEnvVars(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("🌍 Environment variables:") + "\n")

	if len(data.EnvStatic) > 0 {
		b.WriteString("   " + keyStyle.Render("Static:") + "\n")
		for name, value := range data.EnvStatic {
			displayValue := truncateString(value, 50)
			b.WriteString(fmt.Sprintf("      %s=%s\n",
				keyStyle.Render(name),
				subtleStyle.Render(displayValue)))
		}
	}

	if len(data.EnvShell) > 0 {
		b.WriteString("   " + keyStyle.Render("Dynamic (shell):") + "\n")
		for name, v := range data.EnvShell {
			displayCmd := truncateString(v.Command, 50)
			status := warningStyle.Render("⏳ not approved")
			if v.Approved {
				status = successStyle.Render("✓ approved")
			}
			b.WriteString(fmt.Sprintf("      %s=$(%s) [%s]\n",
				keyStyle.Render(name),
				subtleStyle.Render(displayCmd),
				status))
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func renderFlags(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("🏴 Flags:") + "\n")
	b.WriteString("   " + valueStyle.Render(strings.Join(data.Flags, ", ")))
	return b.String()
}

func renderCacheInfo(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("💾 Cache:") + "\n")

	b.WriteString("   " + keyStyle.Render("Path: ") + subtleStyle.Render(data.CachePath) + "\n")
	b.WriteString("   " + keyStyle.Render("Size: ") + valueStyle.Render(formatBytes(data.CacheFileSize)) + "\n")
	b.WriteString("   " + keyStyle.Render("Total entries: ") + valueStyle.Render(fmt.Sprintf("%d", data.CacheTotalEntries)))

	// Only show current directory cache status if there's a config
	if data.HasAnyConfig {
		b.WriteString("\n")
		if data.CacheValid {
			b.WriteString("   " + keyStyle.Render("Current directory:") + "\n")
			b.WriteString("      " + keyStyle.Render("Status: ") + successStyle.Render("✓ Valid") + "\n")
			b.WriteString("      " + keyStyle.Render("Updated: ") + valueStyle.Render(data.CacheUpdated.Format("2006-01-02 15:04:05")) + "\n")
			if data.CacheLocalOnly {
				b.WriteString("      " + keyStyle.Render("Local only: ") + valueStyle.Render("yes"))
			}
		} else {
			b.WriteString("   " + keyStyle.Render("Current directory:") + "\n")
			b.WriteString("      " + keyStyle.Render("Status: ") + errorStyle.Render("✗ Invalid") + "\n")
			b.WriteString("      " + subtleStyle.Render("Will regenerate on next cd"))
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func renderCompletionInfo(data *Data) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("🔄 Completion:") + "\n")

	// Detection cache info
	if data.CompletionDetection != nil {
		b.WriteString("   " + keyStyle.Render("Detection cache: ") + subtleStyle.Render(data.CompletionDetection.Path) + "\n")
		b.WriteString("   " + keyStyle.Render("Detection cache size: ") + valueStyle.Render(formatBytes(data.CompletionDetection.Size)) + "\n")

		if len(data.CompletionDetection.Commands) > 0 {
			b.WriteString("   " + keyStyle.Render("Detected commands: ") + valueStyle.Render(fmt.Sprintf("%d", len(data.CompletionDetection.Commands))) + "\n\n")

			// Group commands by source
			sourceGroups := make(map[string][]string)
			for cmd, source := range data.CompletionDetection.Commands {
				sourceGroups[source] = append(sourceGroups[source], cmd)
			}

			b.WriteString("   " + keyStyle.Render("Sources:") + "\n")
			for _, source := range []string{"Cobra", "Flag", "Env", "Script"} {
				if cmds, ok := sourceGroups[source]; ok {
					b.WriteString(fmt.Sprintf("      %s (%s): %s\n",
						keyStyle.Render(source),
						valueStyle.Render(fmt.Sprintf("%d", len(cmds))),
						subtleStyle.Render(strings.Join(cmds, ", "))))
				}
			}
		}
	} else {
		b.WriteString("   " + subtleStyle.Render("Detection cache not created yet"))
	}

	// Registry info
	if data.CompletionRegistry != nil && data.CompletionRegistry.Size > 0 {
		b.WriteString("\n   " + keyStyle.Render("Registry:") + "\n")
		b.WriteString("      " + keyStyle.Render("Path: ") + subtleStyle.Render(data.CompletionRegistry.Path) + "\n")
		b.WriteString("      " + keyStyle.Render("Size: ") + valueStyle.Render(formatBytes(data.CompletionRegistry.Size)) + "\n")
		if data.CompletionRegistry.ToolsCount > 0 {
			b.WriteString("      " + keyStyle.Render("Tools available: ") + valueStyle.Render(fmt.Sprintf("%d", data.CompletionRegistry.ToolsCount)) + "\n")
		}
	}

	// Downloaded scripts
	if len(data.CompletionScripts) > 0 {
		b.WriteString("\n   " + keyStyle.Render("Downloaded scripts:") + "\n")
		for _, script := range data.CompletionScripts {
			b.WriteString(fmt.Sprintf("      %s (%s)\n",
				valueStyle.Render(script.Tool),
				subtleStyle.Render(formatBytes(script.Size))))
		}
	}

	// Completion overrides
	if len(data.CompletionOverrides) > 0 {
		b.WriteString("\n   " + keyStyle.Render("Completion overrides:") + "\n")
		for alias, cmd := range data.CompletionOverrides {
			b.WriteString(fmt.Sprintf("      %s → %s\n",
				keyStyle.Render(alias),
				valueStyle.Render(cmd)))
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
