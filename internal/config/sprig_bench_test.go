package config

import (
	"testing"
)

func BenchmarkExpandTemplate_SingleCall(b *testing.B) {
	cfg := &Config{ConfigDir: "/tmp/test"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.expandTemplate("hello {{.DIRVANA_DIR}}")
	}
}

func BenchmarkExpandTemplate_Realistic10Calls(b *testing.B) {
	cfg := &Config{ConfigDir: "/tmp/test"}
	templates := []string{
		"git", "git status", "kubectl", "kubecolor",
		"helm", "packer", "task", "terraform",
		"echo hello {{.DIRVANA_DIR}}", "{{.USER_WORKING_DIR}}/bin",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range templates {
			cfg.expandTemplate(t)
		}
	}
}
