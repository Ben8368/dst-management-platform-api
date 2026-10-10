package runner

import (
	"fmt"
	"os"
	"strings"
)

// SteamCompatMode selects how SteamCMD's i386 binary is launched via DEBUGGER.
type SteamCompatMode string

const (
	SteamCompatAuto   SteamCompatMode = "auto"
	SteamCompatNative SteamCompatMode = "native"
	SteamCompatBox86  SteamCompatMode = "box86"
	SteamCompatBox32  SteamCompatMode = "box32"
)

const steamCMDCompatEnv = "DMP_STEAMCMD_COMPAT"

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

// BuildSteamCMDCommand builds a CommandSpec that executes the official steamcmd.sh.
// Compatibility is applied via the DEBUGGER environment variable (consumed by
// steamcmd.sh when launching linux32/steamcmd), not by wrapping the script with
// box86/box64.
//
// SteamCMD self-restart preserves DEBUGGER because steamcmd.sh execs itself
// within the same inherited process environment:
//
//	if [ $STATUS -eq $MAGIC_RESTART_EXITCODE ]; then
//	    exec "$0" "$@"
//	fi
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
		env, err := envMapToSlice(map[string]string{"DEBUGGER": box64Path})
		if err != nil {
			return CommandSpec{}, err
		}
		spec.Env = env
		return spec, nil
	}

	return CommandSpec{}, fmt.Errorf("unsupported SteamCMD compatibility mode: %s", effective)
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
