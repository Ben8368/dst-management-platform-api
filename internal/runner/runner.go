package runner

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// BinaryArch represents the target architecture of a binary
type BinaryArch string

const (
	Native BinaryArch = "native"
	X8664  BinaryArch = "x86_64"
	X8632  BinaryArch = "x86_32"
)

// CommandSpec represents a structured command with binary, arguments, and environment
type CommandSpec struct {
	Binary string
	Args   []string
	Env    []string
}

// Config holds configuration for the compatibility runner
type Config struct {
	HostOS   string
	HostArch string
	// Box64Path overrides the default Box64 location
	Box64Path string
	// Box86Path overrides the default Box86 location
	Box86Path string
}

// DefaultConfig returns a Config with runtime.GOOS and runtime.GOARCH
func DefaultConfig() Config {
	return Config{
		HostOS:   runtime.GOOS,
		HostArch: runtime.GOARCH,
	}
}

// BuildCompatCommand builds a CommandSpec for running a binary on the current host architecture.
// It automatically adds compatibility layers (Box64/Box86) when cross-architecture execution is needed.
func BuildCompatCommand(binary string, args []string, targetArch BinaryArch, config Config) (CommandSpec, error) {
	// Make a copy of args to avoid modifying the input slice
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)

	// Native binaries always run directly
	if targetArch == Native {
		return CommandSpec{
			Binary: binary,
			Args:   argsCopy,
		}, nil
	}

	// On non-Linux, only native binaries are supported
	if config.HostOS != "linux" {
		return CommandSpec{}, fmt.Errorf("unsupported host OS %s for cross-architecture execution (target: %s)", config.HostOS, targetArch)
	}

	// AMD64 host: all x86 architectures run natively
	if config.HostArch == "amd64" {
		return CommandSpec{
			Binary: binary,
			Args:   argsCopy,
		}, nil
	}

	// ARM64 host: x86 binaries need emulation
	if config.HostArch == "arm64" {
		switch targetArch {
		case X8664:
			box64Path, err := resolveBox64Path(config)
			if err != nil {
				return CommandSpec{}, fmt.Errorf("ARM64 host requires Box64 to run the x86_64 DST server. %w", err)
			}
			// Prepend the binary to args
			newArgs := make([]string, 0, len(argsCopy)+1)
			newArgs = append(newArgs, binary)
			newArgs = append(newArgs, argsCopy...)
			return CommandSpec{
				Binary: box64Path,
				Args:   newArgs,
			}, nil

		case X8632:
			box86Path, err := resolveBox86Path(config)
			if err != nil {
				return CommandSpec{}, fmt.Errorf("ARM64 host requires Box86 to run the x86 32-bit DST server. %w", err)
			}
			newArgs := make([]string, 0, len(argsCopy)+1)
			newArgs = append(newArgs, binary)
			newArgs = append(newArgs, argsCopy...)
			return CommandSpec{
				Binary: box86Path,
				Args:   newArgs,
			}, nil
		}
	}

	return CommandSpec{}, fmt.Errorf("unsupported host architecture: %s (target: %s)", config.HostArch, targetArch)
}

// resolveBox64Path finds the Box64 executable
func resolveBox64Path(config Config) (string, error) {
	// Check environment variable first
	if envPath := os.Getenv("DMP_BOX64_PATH"); envPath != "" {
		if err := validateExecutable(envPath, "DMP_BOX64_PATH"); err != nil {
			return "", err
		}
		return envPath, nil
	}

	// Check explicit config
	if config.Box64Path != "" {
		if err := validateExecutable(config.Box64Path, "Config.Box64Path"); err != nil {
			return "", err
		}
		return config.Box64Path, nil
	}

	// Try common location
	commonPath := "/usr/bin/box64"
	if err := validateExecutable(commonPath, ""); err == nil {
		return commonPath, nil
	}

	// Try PATH lookup
	if path, err := exec.LookPath("box64"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("box64 not found. Install Box64 or set DMP_BOX64_PATH environment variable")
}

// resolveBox86Path finds the Box86 executable
func resolveBox86Path(config Config) (string, error) {
	// Check environment variable first
	if envPath := os.Getenv("DMP_BOX86_PATH"); envPath != "" {
		if err := validateExecutable(envPath, "DMP_BOX86_PATH"); err != nil {
			return "", err
		}
		return envPath, nil
	}

	// Check explicit config
	if config.Box86Path != "" {
		if err := validateExecutable(config.Box86Path, "Config.Box86Path"); err != nil {
			return "", err
		}
		return config.Box86Path, nil
	}

	// Try common location
	commonPath := "/usr/bin/box86"
	if err := validateExecutable(commonPath, ""); err == nil {
		return commonPath, nil
	}

	// Try PATH lookup
	if path, err := exec.LookPath("box86"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("box86 not found. Install Box86 or set DMP_BOX86_PATH environment variable")
}

// validateExecutable checks if a path points to an executable file
func validateExecutable(path string, source string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if source != "" {
				return fmt.Errorf("configured %s does not exist: %s", source, path)
			}
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("cannot stat %s: %w", path, err)
	}

	if info.IsDir() {
		if source != "" {
			return fmt.Errorf("configured %s is a directory, not a file: %s", source, path)
		}
		return fmt.Errorf("path is a directory: %s", path)
	}

	// On Unix-like systems, check if executable bit is set
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			if source != "" {
				return fmt.Errorf("configured %s is not executable: %s", source, path)
			}
			return fmt.Errorf("file is not executable: %s", path)
		}
	}

	return nil
}
