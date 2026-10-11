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
	if spec.Binary != "/bin/sh" {
		t.Fatalf("expected managed shell runner, got %s", spec.Binary)
	}
	if len(spec.Env) != 0 {
		t.Errorf("Box32 runner must inherit environment without setting DEBUGGER, got %v", spec.Env)
	}
	if len(spec.Args) < 7 || spec.Args[0] != "-c" || spec.Args[1] != box32SteamCMDScript {
		t.Fatalf("unexpected managed runner args: %#v", spec.Args)
	}
	if spec.Args[2] != "dmp-steamcmd-box32" || spec.Args[3] != "steamcmd/linux32" ||
		spec.Args[4] != fakeBox64 || spec.Args[5] != "steamcmd/linux32/steamcmd" ||
		spec.Args[6] != "3" {
		t.Errorf("unexpected Box32 positional parameters: %#v", spec.Args[2:7])
	}
	if got := spec.Args[7:]; len(got) != 1 || got[0] != "+quit" {
		t.Errorf("original SteamCMD args were not preserved: %#v", got)
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

func TestSteamCMD_Box86SelfUpdatePreservesDebugger(t *testing.T) {
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

func TestSteamCMD_Box32PathDerivation(t *testing.T) {
	tests := []struct {
		script string
		want   string
	}{
		{script: "./steamcmd.sh", want: "linux32/steamcmd"},
		{script: "steamcmd/steamcmd.sh", want: "steamcmd/linux32/steamcmd"},
		{script: "/tmp/example/steamcmd.sh", want: "/tmp/example/linux32/steamcmd"},
	}
	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			if got := steamCMDLinux32Path(tt.script); got != tt.want {
				t.Errorf("steamCMDLinux32Path(%q) = %q, want %q", tt.script, got, tt.want)
			}
		})
	}
}

func TestSteamCMD_Box32ShellRenderPreservesArgsAndEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell runner test requires Unix")
	}
	clearSteamCMDEnv(t)

	dir := t.TempDir()
	box64 := writeSteamCMDTestExecutable(t, filepath.Join(dir, "fake-box64"), "#!/bin/sh\nexec \"$@\"\n")
	statePath := filepath.Join(dir, "state")
	argsPath := filepath.Join(dir, "args")
	steamRoot := filepath.Join(dir, "steamcmd")
	writeSteamCMDTestExecutable(t, filepath.Join(steamRoot, "linux32", "steamcmd"), `#!/bin/sh
count=0
if [ -f "$STATE_PATH" ]; then count=$(cat "$STATE_PATH"); fi
count=$((count + 1))
printf '%s' "$count" > "$STATE_PATH"
printf '%s\n' "$@" >> "$ARGS_PATH"
printf 'LD_LIBRARY_PATH=%s\n' "$LD_LIBRARY_PATH" >> "$ARGS_PATH"
if [ "$count" -eq 1 ]; then exit 42; fi
exit 0
`)

	installDir := "/root/DST Server/a'b$;`"
	originalArgs := []string{"+force_install_dir", installDir, "+custom", "space value", "single'quote", "$HOME", ";", "`id`", "+quit"}
	spec, err := BuildSteamCMDCommand(filepath.Join(steamRoot, "steamcmd.sh"), originalArgs, SteamCMDConfig{
		HostOS:    "linux",
		HostArch:  "arm64",
		Mode:      SteamCompatBox32,
		Box64Path: box64,
	})
	if err != nil {
		t.Fatalf("build Box32 command: %v", err)
	}
	rendered := ShellRender(spec)
	if strings.Contains(spec.Args[1], installDir) {
		t.Fatal("dynamic SteamCMD argument was interpolated into the constant shell script")
	}
	if !strings.Contains(rendered, "'"+box32SteamCMDScript+"'") {
		t.Fatalf("rendered command does not quote the constant runner script: %s", rendered)
	}

	cmd := exec.Command("/bin/sh", "-c", rendered)
	cmd.Env = MergeEnv(os.Environ(), map[string]string{
		"ARGS_PATH":       argsPath,
		"STATE_PATH":      statePath,
		"LD_LIBRARY_PATH": "/existing/libs",
	})
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execute rendered Box32 command: %v, output=%s", err, output)
	}

	gotLines, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read guest argv log: %v", err)
	}
	wantLines := make([]string, 0, len(originalArgs)*2+2)
	wantLines = append(wantLines, originalArgs...)
	libDir := filepath.Join(steamRoot, "linux32")
	wantLines = append(wantLines, "LD_LIBRARY_PATH="+libDir+":/existing/libs")
	wantLines = append(wantLines, originalArgs...)
	wantLines = append(wantLines, "LD_LIBRARY_PATH="+libDir+":/existing/libs")
	if got := strings.Split(strings.TrimSuffix(string(gotLines), "\n"), "\n"); !equalStrings(got, wantLines) {
		t.Fatalf("guest argv/environment mismatch\n got: %#v\nwant: %#v", got, wantLines)
	}
	if got, err := os.ReadFile(statePath); err != nil || strings.TrimSpace(string(got)) != "2" {
		t.Fatalf("expected two guest launches, state=%q err=%v", got, err)
	}
}

func TestSteamCMD_Box32RestartLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell runner test requires Unix")
	}
	clearSteamCMDEnv(t)
	dir := t.TempDir()
	box64 := writeSteamCMDTestExecutable(t, filepath.Join(dir, "fake-box64"), "#!/bin/sh\nexec \"$@\"\n")
	statePath := filepath.Join(dir, "state")
	steamRoot := filepath.Join(dir, "steamcmd")
	writeSteamCMDTestExecutable(t, filepath.Join(steamRoot, "linux32", "steamcmd"), `#!/bin/sh
count=0
if [ -f "$STATE_PATH" ]; then count=$(cat "$STATE_PATH"); fi
count=$((count + 1))
printf '%s' "$count" > "$STATE_PATH"
exit 42
`)
	spec, err := BuildSteamCMDCommand(filepath.Join(steamRoot, "steamcmd.sh"), nil, SteamCMDConfig{
		HostOS: "linux", HostArch: "arm64", Mode: SteamCompatBox32, Box64Path: box64,
	})
	if err != nil {
		t.Fatalf("build Box32 command: %v", err)
	}
	cmd := exec.Command(spec.Binary, spec.Args...)
	cmd.Env = MergeEnv(os.Environ(), map[string]string{"STATE_PATH": statePath})
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected final exit 42 after restart limit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 42 {
		t.Fatalf("expected exit 42, got err=%v output=%s", err, output)
	}
	if !strings.Contains(string(output), "restart limit") {
		t.Errorf("expected restart limit message, output=%s", output)
	}
	if got, readErr := os.ReadFile(statePath); readErr != nil || strings.TrimSpace(string(got)) != "3" {
		t.Fatalf("expected 3 guest launches before limit, state=%q err=%v", got, readErr)
	}
}

func TestSteamCMD_Box32PropagatesExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell runner test requires Unix")
	}
	clearSteamCMDEnv(t)
	dir := t.TempDir()
	box64 := writeSteamCMDTestExecutable(t, filepath.Join(dir, "fake-box64"), "#!/bin/sh\nexec \"$@\"\n")
	steamRoot := filepath.Join(dir, "steamcmd")
	writeSteamCMDTestExecutable(t, filepath.Join(steamRoot, "linux32", "steamcmd"), "#!/bin/sh\nexit 17\n")
	spec, err := BuildSteamCMDCommand(filepath.Join(steamRoot, "steamcmd.sh"), []string{"+quit"}, SteamCMDConfig{
		HostOS: "linux", HostArch: "arm64", Mode: SteamCompatBox32, Box64Path: box64,
	})
	if err != nil {
		t.Fatalf("build Box32 command: %v", err)
	}
	cmd := exec.Command(spec.Binary, spec.Args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected guest exit code 17")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 17 {
		t.Fatalf("expected exit 17, got err=%v output=%s", err, output)
	}
}

func writeSteamCMDTestExecutable(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create executable directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	return path
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
