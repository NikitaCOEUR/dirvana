package shell

import _ "embed"

// Embedded shell completion templates
// These templates are compiled into the binary at build time

//go:embed templates/completion/bash.tmpl
var bashTemplate string

//go:embed templates/completion/zsh.tmpl
var zshTemplate string

// Embedded shell hook templates
//
//go:embed templates/hook/bash.tmpl
var bashHookTemplate string

//go:embed templates/hook/zsh.tmpl
var zshHookTemplate string
