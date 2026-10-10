package runner

import (
	"dst-management-platform-api/utils"
	"strings"
)

// ShellRender converts a CommandSpec to a safely-escaped POSIX shell command string.
// This is needed for compatibility with the current screen/bash-based process management.
//
// When spec.Env is set, assignments are rendered as KEY='value' prefixes so that
// BashCMD paths receive the same environment overrides as exec.Command paths.
// Environment values are ShellQuoted; keys must be valid shell variable names.
func ShellRender(spec CommandSpec) string {
	parts := make([]string, 0, 1+len(spec.Env)+len(spec.Args))

	for _, entry := range spec.Env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if err := validateEnvKey(key); err != nil {
			continue
		}
		parts = append(parts, key+"="+utils.ShellQuote(value))
	}

	parts = append(parts, utils.ShellQuote(spec.Binary))
	for _, arg := range spec.Args {
		parts = append(parts, utils.ShellQuote(arg))
	}

	return strings.Join(parts, " ")
}
