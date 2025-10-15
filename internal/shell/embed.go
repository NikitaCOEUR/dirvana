package shell

import _ "embed"

// Embedded shell completion templates
// These templates are compiled into the binary at build time

//go:embed templates/bash.tmpl
var bashTemplate string

//go:embed templates/zsh.tmpl
var zshTemplate string
