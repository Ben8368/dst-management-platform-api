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

	spec, err := runner.BuildSteamCMDCommand("./steamcmd.sh", gameUpdateSteamCMDArgs(dstDir), runner.DefaultSteamCMDConfig())
	if err != nil {
		return "", err
	}

	return "cd " + utils.ShellQuote(steamDir) + " && " + runner.ShellRender(spec), nil
}

func gameUpdateSteamCMDArgs(dstDir string) []string {
	return []string{
		"+force_install_dir",
		dstDir,
		"+login",
		"anonymous",
		"+app_update",
		"343050",
		"validate",
		"+quit",
	}
}
