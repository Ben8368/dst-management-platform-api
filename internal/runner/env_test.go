package runner

import (
	"strings"
	"testing"
)

func TestMergeEnv_PreservesBaseAndOverridesDebugger(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"HOME=/tmp/test",
		"DEBUGGER=/old/debugger",
		"LANG=C",
	}
	merged := MergeEnv(base, map[string]string{
		"DEBUGGER": "/new/box86",
	})

	envMap := map[string]string{}
	debuggerCount := 0
	for _, e := range merged {
		key, value, ok := strings.Cut(e, "=")
		if !ok {
			t.Fatalf("invalid env entry: %q", e)
		}
		if key == "DEBUGGER" {
			debuggerCount++
		}
		envMap[key] = value
	}

	if envMap["PATH"] != "/usr/bin" {
		t.Errorf("PATH not preserved: %v", envMap["PATH"])
	}
	if envMap["HOME"] != "/tmp/test" {
		t.Errorf("HOME not preserved: %v", envMap["HOME"])
	}
	if envMap["LANG"] != "C" {
		t.Errorf("LANG not preserved: %v", envMap["LANG"])
	}
	if envMap["DEBUGGER"] != "/new/box86" {
		t.Errorf("DEBUGGER not overridden: %v", envMap["DEBUGGER"])
	}
	if debuggerCount != 1 {
		t.Errorf("expected exactly one DEBUGGER, got %d in %v", debuggerCount, merged)
	}
}

func TestMergeEnv_EmptyOverrides(t *testing.T) {
	base := []string{"PATH=/bin", "HOME=/home/u"}
	merged := MergeEnv(base, nil)
	if len(merged) != 2 {
		t.Fatalf("expected 2 entries, got %v", merged)
	}
	if merged[0] != "PATH=/bin" || merged[1] != "HOME=/home/u" {
		t.Errorf("unexpected merge result: %v", merged)
	}
}

func TestEnviron_NilWhenNoOverrides(t *testing.T) {
	spec := CommandSpec{Binary: "steamcmd/steamcmd.sh", Args: []string{"+quit"}}
	if env := Environ(spec); env != nil {
		t.Errorf("expected nil Env for native/no-override spec, got %v", env)
	}
}

func TestValidateEnvKey(t *testing.T) {
	if err := validateEnvKey("DEBUGGER"); err != nil {
		t.Errorf("DEBUGGER should be valid: %v", err)
	}
	if err := validateEnvKey("1BAD"); err == nil {
		t.Error("expected error for key starting with digit")
	}
	if err := validateEnvKey("BAD-KEY"); err == nil {
		t.Error("expected error for key with hyphen")
	}
}

func TestShellRender_WithEnv(t *testing.T) {
	spec := CommandSpec{
		Binary: "steamcmd/steamcmd.sh",
		Args:   []string{"+force_install_dir", "/root/DST Server/mods", "+quit"},
		Env:    []string{"DEBUGGER=/usr/bin/box86"},
	}

	result := ShellRender(spec)

	if !strings.HasPrefix(result, "DEBUGGER='/usr/bin/box86'") {
		t.Errorf("expected DEBUGGER prefix, got: %s", result)
	}
	if !strings.Contains(result, "'/root/DST Server/mods'") {
		t.Errorf("expected quoted path with spaces, got: %s", result)
	}
	if strings.Contains(result, "steamcmd.sh DEBUGGER=") {
		t.Errorf("DEBUGGER must not be a command argument: %s", result)
	}
}

func TestShellRender_EnvValueWithSpaces(t *testing.T) {
	spec := CommandSpec{
		Binary: "steamcmd/steamcmd.sh",
		Args:   []string{"+quit"},
		Env:    []string{"DEBUGGER=/opt/my tools/box86"},
	}
	result := ShellRender(spec)
	if !strings.Contains(result, "DEBUGGER='/opt/my tools/box86'") {
		t.Errorf("DEBUGGER value with spaces must be quoted: %s", result)
	}
}
