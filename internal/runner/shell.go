package runner

import (
	"dst-management-platform-api/utils"
	"strings"
)

// ShellRender converts a CommandSpec to a safely-escaped POSIX shell command string.
// This is needed for compatibility with the current screen/bash-based process management.
func ShellRender(spec CommandSpec) string {
	parts := []string{utils.ShellQuote(spec.Binary)}

	for _, arg := range spec.Args {
		parts = append(parts, utils.ShellQuote(arg))
	}

	return strings.Join(parts, " ")
}
