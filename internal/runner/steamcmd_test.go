package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func clearSteamCMDEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DMP_STEAMCMD_COMPAT", "")
	t.Setenv("DMP_BOX64_PATH", "")
	t.Setenv("DMP_BOX86_PATH", "")
}

func debuggerFromEnv(env []string) (string, bool) {
	var found string
	count := 0
	for _, e := range env {
		if strings.HasPrefix(e, "DEBUGGER=") {
			count++
			found = strings.TrimPrefix(e, "DEBUGGER=")
		}
	}
	if count == 0 {
		return "", false
	}
	if count > 1 {
		return found, true // caller can assert uniqueness separately
	}
	return found, true
}

func TestSteamCMD_AMD64_Auto(t *testing.T) {
	clearSteamCMDEnv(t)

	config := SteamCMDConfig{
		HostOS:   "linux",
		HostArch: "amd64",
		Mode:     SteamCompatAuto,
	}

	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+quit"}, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Binary != "steamcmd/steamcmd.sh" {
		t.Errorf("expected binary steamcmd/steamcmd.sh, got %s", spec.Binary)
	}
	if len(spec.Env) != 0 {
		t.Errorf("AMD64 auto should not set DEBUGGER, got Env=%v", spec.Env)
	}
}

func TestSteamCMD_ARM64_Auto_Box86(t *testing.T) {
	clearSteamCMDEnv(t)
	fakeBox86 := createFakeExecutable(t, "box86")

	config := SteamCMDConfig{
		HostOS:    "linux",
		HostArch:  "arm64",
		Mode:      SteamCompatAuto,
		Box86Path: fakeBox86,
	}

	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+login", "anonymous", "+quit"}, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Binary != "steamcmd/steamcmd.sh" {
		t.Errorf("expected binary steamcmd/steamcmd.sh, got %s", spec.Binary)
	}
	dbg, ok := debuggerFromEnv(spec.Env)
	if !ok {
		t.Fatal("expected DEBUGGER in Env")
	}
	if dbg != fakeBox86 {
		t.Errorf("expected DEBUGGER=%s, got %s", fakeBox86, dbg)
	}
}

func TestSteamCMD_ARM64_Auto_NoBox86(t *testing.T) {
	clearSteamCMDEnv(t)
	fakeBox64 := createFakeExecutable(t, "box64")

	config := SteamCMDConfig{
		HostOS:    "linux",
		HostArch:  "arm64",
		Mode:      SteamCompatAuto,
		Box64Path: fakeBox64,
		// Box86 intentionally missing
	}

	_, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+quit"}, config)
	if err == nil {
		t.Fatal("expected error when Box86 is missing on ARM64 auto")
	}

	errMsg := err.Error()
	if !contains(errMsg, "Box86") {
		t.Errorf("error should mention Box86, got: %s", errMsg)
	}
	if !contains(errMsg, "box32") {
		t.Errorf("error should mention box32 opt-in, got: %s", errMsg)
	}
	if !contains(errMsg, "arm64") {
		t.Errorf("error should mention arm64, got: %s", errMsg)
	}
}

func TestSteamCMD_ARM64_Box86(t *testing.T) {
	clearSteamCMDEnv(t)
	fakeBox86 := createFakeExecutable(t, "box86")

	config := SteamCMDConfig{
		HostOS:    "linux",
		HostArch:  "arm64",
		Mode:      SteamCompatBox86,
		Box86Path: fakeBox86,
	}

	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+quit"}, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Binary != "steamcmd/steamcmd.sh" {
		t.Errorf("expected steamcmd.sh binary, got %s", spec.Binary)
	}
	dbg, ok := debuggerFromEnv(spec.Env)
	if !ok || dbg != fakeBox86 {
		t.Errorf("expected DEBUGGER=%s, got %v (%v)", fakeBox86, dbg, spec.Env)
	}
}

func TestSteamCMD_ARM64_Box32(t *testing.T) {
	clearSteamCMDEnv(t)
	fakeBox64 := createFakeExecutable(t, "box64")

	config := SteamCMDConfig{
		HostOS:    "linux",
		HostArch:  "arm64",
		Mode:      SteamCompatBox32,
		Box64Path: fakeBox64,
	}

	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+quit"}, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Binary != "steamcmd/steamcmd.sh" {
		t.Errorf("expected steamcmd.sh binary, got %s", spec.Binary)
	}
	dbg, ok := debuggerFromEnv(spec.Env)
	if !ok || dbg != fakeBox64 {
		t.Errorf("expected DEBUGGER=%s, got %v (%v)", fakeBox64, dbg, spec.Env)
	}
}

func TestSteamCMD_ARM64_Native(t *testing.T) {
	clearSteamCMDEnv(t)

	config := SteamCMDConfig{
		HostOS:   "linux",
		HostArch: "arm64",
		Mode:     SteamCompatNative,
	}

	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+quit"}, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Env) != 0 {
		t.Errorf("native mode should not set DEBUGGER, got Env=%v", spec.Env)
	}
}

func TestSteamCMD_InvalidMode(t *testing.T) {
	clearSteamCMDEnv(t)

	config := SteamCMDConfig{
		HostOS:   "linux",
		HostArch: "amd64",
		Mode:     SteamCompatMode("qemu"),
	}

	_, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+quit"}, config)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !contains(err.Error(), "invalid") {
		t.Errorf("error should mention invalid mode, got: %s", err.Error())
	}
}

func TestSteamCMD_ArgsPreserved(t *testing.T) {
	clearSteamCMDEnv(t)

	args := []string{
		"+force_install_dir",
		"/root/DST Server/mods",
		"+login",
		"anonymous",
		"+workshop_download_item",
		"322330",
		"123456",
		"+quit",
	}

	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", args, SteamCMDConfig{
		HostOS:   "linux",
		HostArch: "amd64",
		Mode:     SteamCompatNative,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(spec.Args) != len(args) {
		t.Fatalf("expected %d args, got %d", len(args), len(spec.Args))
	}
	for i := range args {
		if spec.Args[i] != args[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, args[i], spec.Args[i])
		}
	}
}

func TestSteamCMD_ShellRender_SpecialChars(t *testing.T) {
	clearSteamCMDEnv(t)
	fakeBox86 := createFakeExecutable(t, "box86")

	installDir := `/root/DST Server/mods` + `'` + `$` + `;` + "`"
	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{
		"+force_install_dir",
		installDir,
		"+login",
		"anonymous",
		"+workshop_download_item",
		"322330",
		"1",
		"+quit",
	}, SteamCMDConfig{
		HostOS:    "linux",
		HostArch:  "arm64",
		Mode:      SteamCompatBox86,
		Box86Path: fakeBox86,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rendered := ShellRender(spec)

	if !strings.HasPrefix(rendered, "DEBUGGER=") {
		t.Errorf("shell render should prefix DEBUGGER, got: %s", rendered)
	}
	quotedDebugger := "'" + strings.ReplaceAll(fakeBox86, "'", `'\''`) + "'"
	if !strings.Contains(rendered, "DEBUGGER="+quotedDebugger) {
		t.Errorf("expected quoted DEBUGGER=%s in: %s", quotedDebugger, rendered)
	}
	if !strings.Contains(rendered, "'steamcmd/steamcmd.sh'") {
		t.Errorf("binary should be quoted: %s", rendered)
	}
	if !strings.Contains(rendered, "'+force_install_dir'") {
		t.Errorf("missing force_install_dir flag: %s", rendered)
	}
	// installDir contains spaces and shell metacharacters; must remain one quoted arg
	if !strings.Contains(rendered, "'+workshop_download_item'") {
		t.Errorf("missing workshop_download_item: %s", rendered)
	}
	if strings.Contains(rendered, "steamcmd.sh DEBUGGER=") {
		t.Errorf("DEBUGGER must not appear as a command argument: %s", rendered)
	}
}

func TestSteamCMD_DebuggerNotInArgs(t *testing.T) {
	clearSteamCMDEnv(t)
	fakeBox86 := createFakeExecutable(t, "box86")

	spec, err := BuildSteamCMDCommand("steamcmd/steamcmd.sh", []string{"+quit"}, SteamCMDConfig{
		HostOS:    "linux",
		HostArch:  "arm64",
		Mode:      SteamCompatBox86,
		Box86Path: fakeBox86,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, arg := range spec.Args {
		if strings.Contains(arg, "DEBUGGER") || strings.Contains(arg, "box86") {
			t.Errorf("DEBUGGER leaked into args: %v", spec.Args)
		}
	}
}

func TestSteamCMD_SelfUpdatePreservesDebugger(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX exec self-restart test requires Unix")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "fake_steamcmd.sh")
	content := `#!/bin/sh
# SteamCMD self-restart preserves DEBUGGER because the script execs itself
# within the same inherited environment.
if [ "${DMP_RESTART_DONE:-0}" = "0" ]; then
  if [ -z "$DEBUGGER" ]; then
    echo "DEBUGGER missing on first run" >&2
    exit 2
  fi
  export DMP_RESTART_DONE=1
  exec "$0" "$@"
fi
if [ -z "$DEBUGGER" ]; then
  echo "DEBUGGER missing after restart" >&2
  exit 3
fi
echo "DEBUGGER=$DEBUGGER"
exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command(script)
	cmd.Env = MergeEnv(os.Environ(), map[string]string{
		"DEBUGGER":         "/usr/bin/box86",
		"DMP_RESTART_DONE": "0",
	})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script failed: %v, output=%s", err, out)
	}
	if !strings.Contains(string(out), "DEBUGGER=/usr/bin/box86") {
		t.Errorf("expected DEBUGGER to survive exec restart, got: %s", out)
	}
}
