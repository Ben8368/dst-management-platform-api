package dst

import (
	"dst-management-platform-api/internal/runner"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"path/filepath"
)

// BuildGameUpdateShellCmd builds the shell command used to update the DST
// dedicated server via SteamCMD (app 343050).
func BuildGameUpdateShellCmd() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	steamDir := filepath.Join(home, "steamcmd")
	dstDir := filepath.Join(home, "dst")

	spec, err := runner.BuildSteamCMDCommand("./steamcmd.sh", []string{
		"+login",
		"anonymous",
		"+force_install_dir",
		dstDir,
		"+app_update",
		"343050",
		"validate",
		"+quit",
	}, runner.DefaultSteamCMDConfig())
	if err != nil {
		return "", err
	}

	return "cd " + utils.ShellQuote(steamDir) + " && " + runner.ShellRender(spec), nil
}
