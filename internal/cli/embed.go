// Package cli implements the bodies of the dirvana CLI commands.
package cli

import _ "embed"

// sampleConfig is the commented starter configuration written by
// 'dirvana init' and 'dirvana edit' when no config file exists yet.
//
//go:embed templates/sample-config.yml
var sampleConfig string
