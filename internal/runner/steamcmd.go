package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SteamCompatMode selects how SteamCMD's i386 binary is launched.
type SteamCompatMode string

const (
	SteamCompatAuto   SteamCompatMode = "auto"
	SteamCompatNative SteamCompatMode = "native"
	SteamCompatBox86  SteamCompatMode = "box86"
	SteamCompatBox32  SteamCompatMode = "box32"
)

const steamCMDCompatEnv = "DMP_STEAMCMD_COMPAT"

const steamCMDMaxRestarts = 3

const box32SteamCMDScript = `steam_lib_dir=$1
box64=$2
steamcmd=$3
max_restarts=$4
shift 4
export LD_LIBRARY_PATH="$steam_lib_dir${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
restart_count=0
while :; do
	"$box64" "$steamcmd" "$@"
	rc=$?
	if [ "$rc" -ne 42 ]; then
		exit "$rc"
	fi
	restart_count=$((restart_count + 1))
	if [ "$restart_count" -ge "$max_restarts" ]; then
		echo "SteamCMD Box32 restart limit reached after $restart_count restart requests" >&2
		exit 42
	fi
done`

// SteamCMDConfig configures SteamCMD compatibility resolution.
// Box64Path / Box86Path reuse the same override semantics as Config
// (and DMP_BOX64_PATH / DMP_BOX86_PATH).
type SteamCMDConfig struct {
	HostOS    string
	HostArch  string
	Mode      SteamCompatMode // empty: read DMP_STEAMCMD_COMPAT, default auto
	Box64Path string
	Box86Path string
}

// DefaultSteamCMDConfig returns a SteamCMDConfig for the current host.
func DefaultSteamCMDConfig() SteamCMDConfig {
	c := DefaultConfig()
	return SteamCMDConfig{
		HostOS:   c.HostOS,
		HostArch: c.HostArch,
	}
}

func (c SteamCMDConfig) toRunnerConfig() Config {
	return Config{
		HostOS:    c.HostOS,
		HostArch:  c.HostArch,
		Box64Path: c.Box64Path,
		Box86Path: c.Box86Path,
	}
}

// BuildSteamCMDCommand builds a CommandSpec for the selected SteamCMD backend.
// Native and Box86 modes execute steamcmd.sh; Box32 directly runs the linux32
// binary and handles SteamCMD's exit-42 self-update restart outside the script.
func BuildSteamCMDCommand(script string, args []string, config SteamCMDConfig) (CommandSpec, error) {
	if script == "" {
		return CommandSpec{}, fmt.Errorf("steamcmd script path must not be empty")
	}

	argsCopy := make([]string, len(args))
	copy(argsCopy, args)

	mode, err := resolveSteamCompatMode(config)
	if err != nil {
		return CommandSpec{}, err
	}

	effective := effectiveSteamCompatMode(mode, config.HostOS, config.HostArch)

	spec := CommandSpec{
		Binary: script,
		Args:   argsCopy,
	}

	switch effective {
	case SteamCompatNative:
		// Do not override DEBUGGER so advanced users can customize their environment.
		// Explicit native on linux/arm64 is allowed; resulting exec format errors
		// are the caller's responsibility.
		return spec, nil

	case SteamCompatBox86:
		box86Path, err := resolveBox86Path(config.toRunnerConfig())
		if err != nil {
			if mode == SteamCompatAuto {
				return CommandSpec{}, fmt.Errorf(
					"SteamCMD requires x86 32-bit compatibility on linux/arm64. " +
						"Box86 was not found. " +
						"Install Box86 or set DMP_BOX86_PATH. " +
						"If your Box64 build supports Box32, set DMP_STEAMCMD_COMPAT=box32")
			}
			return CommandSpec{}, fmt.Errorf("SteamCMD box86 mode: %w", err)
		}
		env, err := envMapToSlice(map[string]string{"DEBUGGER": box86Path})
		if err != nil {
			return CommandSpec{}, err
		}
		spec.Env = env
		return spec, nil

	case SteamCompatBox32:
		// Box32 is opt-in only. Presence of a Box64 binary does not prove BOX32
		// support was compiled in; callers must set DMP_STEAMCMD_COMPAT=box32
		// explicitly when their Box64 build has Box32 enabled.
		box64Path, err := resolveBox64Path(config.toRunnerConfig())
		if err != nil {
			return CommandSpec{}, fmt.Errorf(
				"SteamCMD box32 mode requires Box64 with Box32 support enabled: %w", err)
		}
		steamcmdPath := steamCMDLinux32Path(script)
		steamLibDir := filepath.Dir(steamcmdPath)
		box32Args := make([]string, 0, 7+len(argsCopy))
		box32Args = append(box32Args,
			"-c",
			box32SteamCMDScript,
			"dmp-steamcmd-box32",
			steamLibDir,
			box64Path,
			steamcmdPath,
			fmt.Sprint(steamCMDMaxRestarts),
		)
		box32Args = append(box32Args, argsCopy...)
		spec = CommandSpec{
			Binary: "/bin/sh",
			Args:   box32Args,
		}
		return spec, nil
	}

	return CommandSpec{}, fmt.Errorf("unsupported SteamCMD compatibility mode: %s", effective)
}

func steamCMDLinux32Path(script string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(script), "linux32", "steamcmd"))
}

func resolveSteamCompatMode(config SteamCMDConfig) (SteamCompatMode, error) {
	mode := SteamCompatMode(strings.ToLower(strings.TrimSpace(string(config.Mode))))
	if mode == "" {
		mode = SteamCompatMode(strings.ToLower(strings.TrimSpace(os.Getenv(steamCMDCompatEnv))))
	}
	if mode == "" {
		return SteamCompatAuto, nil
	}
	switch mode {
	case SteamCompatAuto, SteamCompatNative, SteamCompatBox86, SteamCompatBox32:
		return mode, nil
	default:
		return "", fmt.Errorf(
			"invalid %s value %q (allowed: auto, native, box86, box32)",
			steamCMDCompatEnv, mode)
	}
}

// effectiveSteamCompatMode resolves auto to a concrete backend.
// On linux/arm64, auto prefers Box86 and never silent-falls-back to Box32.
func effectiveSteamCompatMode(mode SteamCompatMode, hostOS, hostArch string) SteamCompatMode {
	if mode != SteamCompatAuto {
		return mode
	}
	if hostOS == "linux" && hostArch == "arm64" {
		return SteamCompatBox86
	}
	return SteamCompatNative
}
